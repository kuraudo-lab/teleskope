package cli

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/awseks"
	"github.com/kuraudo-lab/teleskope/internal/buildinfo"
	"github.com/kuraudo-lab/teleskope/internal/inventory"
	"github.com/kuraudo-lab/teleskope/internal/k8s"
	"github.com/kuraudo-lab/teleskope/internal/live"
	"github.com/spf13/cobra"
)

const defaultKubernetesServeInterval = 5 * time.Minute

func newServeCommand(stdout, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{Use: "serve", Short: "Serve a live read-only inventory with periodic collection"}
	cmd.AddCommand(newServeTarget("k8s", stdout, stderr), newServeTarget("eks", stdout, stderr))
	return cmd
}

func newServeTarget(target string, stdout, stderr io.Writer) *cobra.Command {
	var kubeOpts k8s.Options
	var awsOpts awseks.Options
	var listen string
	var interval, awsInterval, timeout time.Duration
	var skipKubernetes bool
	cmd := &cobra.Command{
		Use: target, Short: "Serve live " + target + " inventory",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if interval <= 0 || awsInterval <= 0 || timeout <= 0 {
				return fmt.Errorf("intervals and timeout must be positive")
			}
			if target == "eks" && awsOpts.ClusterName == "" {
				return fmt.Errorf("--cluster is required")
			}
			awsOpts.Profile, _ = cmd.Root().PersistentFlags().GetString("profile")
			awsOpts.Region, _ = cmd.Root().PersistentFlags().GetString("region")
			var logMu sync.Mutex
			log := func(format string, args ...any) {
				logMu.Lock()
				defer logMu.Unlock()
				fmt.Fprintf(stderr, format+"\n", args...)
			}
			var store *live.Store
			kubeOpts.Progress = func(format string, args ...any) {
				log("[kubernetes] "+format, args...)
				if store != nil {
					store.AddEvent("kubernetes", "debug", format, args...)
				}
			}
			awsOpts.Progress = func(format string, args ...any) {
				log("[eks] "+format, args...)
				if store != nil {
					store.AddEvent("eks", "debug", format, args...)
				}
			}
			var sources []live.Source
			if !skipKubernetes {
				var collector *k8s.Collector
				sources = append(sources, live.Source{Name: "kubernetes", Interval: interval, Timeout: timeout,
					Collect: func(ctx context.Context) (*inventory.Snapshot, error) {
						var err error
						if collector == nil {
							collector, err = k8s.NewCollector(kubeOpts)
						}
						if err != nil {
							return nil, err
						}
						data, coverage, err := collector.Collect(ctx)
						snapshot := liveKubernetesSnapshot(data, coverage)
						if err != nil && !hasLiveKubernetesData(data, coverage) {
							return nil, err
						}
						return snapshot, err
					},
					Log: log,
				})
			}
			if target == "eks" {
				var collector *awseks.Collector
				sources = append(sources, live.Source{Name: "eks", Interval: awsInterval, Timeout: timeout,
					Collect: func(ctx context.Context) (*inventory.Snapshot, error) {
						var err error
						if collector == nil {
							collector, err = awseks.NewCollector(ctx, awsOpts)
						}
						if err != nil {
							return nil, err
						}
						return collector.Collect(ctx)
					},
					Log: log,
				})
			}
			store, err := live.New(sources)
			if err != nil {
				return err
			}
			listener, err := net.Listen("tcp", listen)
			if err != nil {
				return fmt.Errorf("listen: %w", err)
			}
			defer listener.Close()
			log("live serve starting target=%s listen=%s interval=%s aws_interval=%s timeout=%s sources=%s", target, listener.Addr(), interval, awsInterval, timeout, sourceNames(sources))
			store.AddEvent("serve", "info", "starting target=%s listen=%s interval=%s aws_interval=%s timeout=%s sources=%s", target, listener.Addr(), interval, awsInterval, timeout, sourceNames(sources))
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			done := make(chan struct{})
			go func() { defer close(done); store.Run(ctx) }()
			server := &http.Server{
				Handler: store.Handler(), ReadHeaderTimeout: 5 * time.Second,
				WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
			}
			serverDone := make(chan error, 1)
			go func() { serverDone <- server.Serve(listener) }()
			fmt.Fprintf(stdout, "Live inventory: http://%s\n", listener.Addr())
			select {
			case err = <-serverDone:
				cancel()
			case <-ctx.Done():
				log("live serve shutting down")
				store.AddEvent("serve", "info", "shutting down")
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
				if shutdownErr := server.Shutdown(shutdownCtx); shutdownErr != nil {
					_ = server.Close()
				}
				shutdownCancel()
				err = <-serverDone
			}
			<-done
			if err == http.ErrServerClosed {
				return nil
			}
			return err
		},
	}
	cmd.Flags().StringVar(&listen, "listen", "127.0.0.1:8080", "HTTP listen address (no built-in authentication)")
	cmd.Flags().DurationVar(&interval, "interval", defaultKubernetesServeInterval, "delay between completed Kubernetes scans (plus up to 10% jitter)")
	cmd.Flags().DurationVar(&timeout, "timeout", defaultCollectionTimeout, "timeout for each collection attempt")
	cmd.Flags().StringVar(&kubeOpts.Kubeconfig, "kubeconfig", "", "path to kubeconfig")
	cmd.Flags().StringVar(&kubeOpts.Context, "kube-context", "", "kubeconfig context")
	awsInterval = 15 * time.Minute
	if target == "eks" {
		cmd.Flags().StringVarP(&awsOpts.ClusterName, "cluster", "c", "", "EKS cluster name")
		cmd.Flags().DurationVar(&awsInterval, "aws-interval", awsInterval, "delay between completed AWS scans (plus up to 10% jitter)")
		cmd.Flags().BoolVar(&skipKubernetes, "skip-kubernetes", false, "collect only AWS-side inventory")
	}
	return cmd
}

func liveKubernetesSnapshot(data inventory.Kubernetes, coverage []inventory.CoverageItem) *inventory.Snapshot {
	return &inventory.Snapshot{
		SchemaVersion: "teleskope.io/snapshot/v1alpha1", CollectedAt: time.Now().UTC(),
		Source:     inventory.Source{Tool: "teleskope", Version: buildinfo.Version, Mode: "live/poll"},
		Kubernetes: data, Coverage: coverage,
	}
}

func hasLiveKubernetesData(data inventory.Kubernetes, coverage []inventory.CoverageItem) bool {
	return len(coverage) > 0 ||
		data.Version.GitVersion != "" ||
		len(data.APIResources) > 0 ||
		len(data.CustomResourceDefinitions) > 0 ||
		len(data.CustomResourceInstances) > 0 ||
		len(data.CustomResourceCounts) > 0 ||
		len(data.APIServices) > 0 ||
		len(data.Namespaces) > 0 ||
		len(data.Nodes) > 0 ||
		len(data.ServiceAccounts) > 0 ||
		len(data.Workloads) > 0 ||
		len(data.Pods) > 0 ||
		len(data.RunningImages) > 0 ||
		len(data.RunningContainers) > 0 ||
		len(data.Services) > 0 ||
		len(data.EndpointSlices) > 0 ||
		len(data.IngressClasses) > 0 ||
		len(data.Ingresses) > 0 ||
		len(data.GatewayClasses) > 0 ||
		len(data.Gateways) > 0 ||
		len(data.GatewayRoutes) > 0 ||
		len(data.StorageClasses) > 0 ||
		len(data.PersistentVolumes) > 0 ||
		len(data.PersistentVolumeClaims) > 0 ||
		len(data.CSIDrivers) > 0 ||
		len(data.CSINodes) > 0 ||
		len(data.VolumeAttachments) > 0 ||
		len(data.RuntimeClasses) > 0 ||
		len(data.ConfigMaps) > 0 ||
		len(data.Secrets) > 0 ||
		len(data.RBAC.Roles) > 0 ||
		len(data.RBAC.RoleBindings) > 0 ||
		len(data.RBAC.ClusterRoles) > 0 ||
		len(data.RBAC.ClusterRoleBindings) > 0 ||
		len(data.Policies.HorizontalPodAutoscalers) > 0 ||
		len(data.Policies.PodDisruptionBudgets) > 0 ||
		len(data.Policies.NetworkPolicies) > 0 ||
		len(data.Policies.ResourceQuotas) > 0 ||
		len(data.Policies.LimitRanges) > 0
}

func sourceNames(sources []live.Source) string {
	names := make([]string, 0, len(sources))
	for _, source := range sources {
		names = append(names, source.Name)
	}
	return strings.Join(names, ",")
}
