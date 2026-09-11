// Package cli implements the Teleskope command-line interface.
package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/awseks"
	"github.com/kuraudo-lab/teleskope/internal/buildinfo"
	"github.com/kuraudo-lab/teleskope/internal/compare"
	"github.com/kuraudo-lab/teleskope/internal/inventory"
	"github.com/kuraudo-lab/teleskope/internal/k8s"
	"github.com/kuraudo-lab/teleskope/internal/render"
	"github.com/kuraudo-lab/teleskope/internal/report"
	"github.com/spf13/cobra"
)

const defaultCollectionTimeout = 5 * time.Minute

// Run executes the CLI and returns its process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	cmd := NewRootCommand(stdout, stderr)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

// NewRootCommand builds the root Cobra command.
func NewRootCommand(stdout, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "teleskope",
		Short:         "EKS and Kubernetes inventory explorer",
		Long:          "Teleskope collects EKS and Kubernetes inventory evidence for human review.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	cmd.Flags().BoolP("version", "v", false, "print version")
	cmd.PersistentFlags().String("profile", "", "AWS shared config profile")
	cmd.PersistentFlags().String("region", "", "AWS region override")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		version, _ := cmd.Flags().GetBool("version")
		if version {
			fmt.Fprintln(stdout, "teleskope", buildinfo.Version)
			return nil
		}
		return cmd.Help()
	}

	cmd.AddCommand(newScanCommand(stdout, stderr))
	cmd.AddCommand(newCompareCommand(stdout, stderr))
	cmd.AddCommand(newAnalyzeCommand(stdout, stderr))
	cmd.AddCommand(newAdvisoryCommand(stdout, stderr))
	cmd.AddCommand(newServeCommand(stdout, stderr))
	return cmd
}

func newScanCommand(stdout, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Collect inventory from a target environment",
	}
	cmd.AddCommand(newScanEKSCommand(stdout, stderr))
	cmd.AddCommand(newScanK8sCommand(stdout, stderr))
	return cmd
}

func newScanEKSCommand(stdout, stderr io.Writer) *cobra.Command {
	var opts awseks.Options
	var kubeOpts k8s.Options
	var output string
	var outputDir string
	var timeout time.Duration
	var skipKubernetes bool

	cmd := &cobra.Command{
		Use:   "eks",
		Short: "Collect read-only EKS and Kubernetes inventory",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if opts.ClusterName == "" {
				return fmt.Errorf("--cluster is required")
			}
			if output == "" {
				output = "report"
			}
			if !validOutput(output) {
				return fmt.Errorf("unsupported output %q; expected report, human, or json", output)
			}

			profile, _ := cmd.Root().PersistentFlags().GetString("profile")
			region, _ := cmd.Root().PersistentFlags().GetString("region")
			opts.Profile = profile
			opts.Region = region

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			if timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}

			progress := newProgress(stderr)
			opts.Progress = progress.Detail
			kubeOpts.Progress = progress.Detail
			progress.Step("collecting EKS inventory for %s", opts.ClusterName)
			snapshot, err := awseks.Collect(ctx, opts)
			if err != nil {
				return err
			}
			if !skipKubernetes {
				progress.Step("collecting Kubernetes API inventory")
				kubernetes, coverage, err := k8s.Collect(ctx, kubeOpts)
				if err != nil {
					return err
				}
				snapshot.Kubernetes = kubernetes
				snapshot.Coverage = append(snapshot.Coverage, coverage...)
			}
			switch strings.ToLower(output) {
			case "report":
				progress.Step("writing report artifacts")
				artifact, err := report.WriteDirectory(snapshot, report.Options{
					BaseDir: outputDir,
					Target:  opts.ClusterName,
				})
				if err != nil {
					return err
				}
				progress.Done("report ready: %s", artifact.Dir)
				fmt.Fprintf(stdout, "report %s\n", artifact.Dir)
				return nil
			case "json":
				progress.Step("rendering JSON")
				return render.JSON(stdout, snapshot)
			default:
				progress.Step("rendering human summary")
				return render.Human(stdout, snapshot)
			}
		},
	}

	cmd.Flags().StringVarP(&opts.ClusterName, "cluster", "c", "", "EKS cluster name")
	cmd.Flags().StringVar(&kubeOpts.Kubeconfig, "kubeconfig", "", "path to kubeconfig for Kubernetes API collection")
	cmd.Flags().StringVar(&kubeOpts.Context, "kube-context", "", "kubeconfig context for Kubernetes API collection")
	cmd.Flags().BoolVar(&skipKubernetes, "skip-kubernetes", false, "skip Kubernetes API collection")
	cmd.Flags().StringVarP(&output, "output", "o", "report", "output format: report, human, json")
	cmd.Flags().StringVar(&outputDir, "output-dir", ".", "parent directory for timestamped report output")
	cmd.Flags().DurationVar(&timeout, "timeout", defaultCollectionTimeout, "collection timeout")
	return cmd
}

func newScanK8sCommand(stdout, stderr io.Writer) *cobra.Command {
	var opts k8s.Options
	var output string
	var outputDir string
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "k8s",
		Short: "Collect read-only Kubernetes API inventory",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if output == "" {
				output = "report"
			}
			if !validOutput(output) {
				return fmt.Errorf("unsupported output %q; expected report, human, or json", output)
			}

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			if timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}

			progress := newProgress(stderr)
			opts.Progress = progress.Detail
			progress.Step("collecting Kubernetes API inventory")
			kubernetes, coverage, err := k8s.Collect(ctx, opts)
			if err != nil {
				return err
			}
			snapshot := &inventory.Snapshot{
				SchemaVersion: "teleskope.io/snapshot/v1alpha1",
				CollectedAt:   time.Now().UTC(),
				Source: inventory.Source{
					Tool:    "teleskope",
					Version: buildinfo.Version,
					Mode:    "out-of-cluster/run-once",
				},
				Kubernetes: kubernetes,
				Coverage:   coverage,
			}
			switch strings.ToLower(output) {
			case "report":
				progress.Step("writing report artifacts")
				artifact, err := report.WriteDirectory(snapshot, report.Options{
					BaseDir: outputDir,
					Target:  firstNonEmpty(opts.Context, kubernetes.Context, "kubernetes"),
				})
				if err != nil {
					return err
				}
				progress.Done("report ready: %s", artifact.Dir)
				fmt.Fprintf(stdout, "report %s\n", artifact.Dir)
				return nil
			case "json":
				progress.Step("rendering JSON")
				return render.JSON(stdout, snapshot)
			default:
				progress.Step("rendering human summary")
				return render.Human(stdout, snapshot)
			}
		},
	}

	cmd.Flags().StringVar(&opts.Kubeconfig, "kubeconfig", "", "path to kubeconfig for Kubernetes API collection")
	cmd.Flags().StringVar(&opts.Context, "kube-context", "", "kubeconfig context for Kubernetes API collection")
	cmd.Flags().StringVarP(&output, "output", "o", "report", "output format: report, human, json")
	cmd.Flags().StringVar(&outputDir, "output-dir", ".", "parent directory for timestamped report output")
	cmd.Flags().DurationVar(&timeout, "timeout", defaultCollectionTimeout, "collection timeout")
	return cmd
}

func validOutput(output string) bool {
	switch strings.ToLower(output) {
	case "report", "human", "json":
		return true
	default:
		return false
	}
}

func newCompareCommand(stdout, stderr io.Writer) *cobra.Command {
	var sourcePath string
	var targetPath string
	var output string

	cmd := &cobra.Command{
		Use:   "compare [--source <scan-dir-or-snapshot.json>] [--target <scan-dir-or-snapshot.json>]",
		Short: "Compare two scan results for migration planning",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 2 {
				return fmt.Errorf("compare accepts at most two positional paths")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && sourcePath == "" {
				sourcePath = args[0]
			}
			if len(args) > 1 && targetPath == "" {
				targetPath = args[1]
			}
			if sourcePath == "" {
				return fmt.Errorf("--source is required")
			}
			if targetPath == "" {
				return fmt.Errorf("--target is required")
			}
			if !validCompareOutput(output) {
				return fmt.Errorf("unsupported output %q; expected human, markdown, or json", output)
			}

			progress := newProgress(stderr)
			progress.Step("loading source scan result")
			source, err := compare.LoadSnapshot(sourcePath)
			if err != nil {
				return err
			}
			progress.Step("loading target scan result")
			target, err := compare.LoadSnapshot(targetPath)
			if err != nil {
				return err
			}
			progress.Step("analyzing migration differences")
			report := compare.Analyze(source, target)
			switch strings.ToLower(output) {
			case "json":
				return compare.WriteJSON(stdout, report)
			case "markdown":
				_, err = fmt.Fprint(stdout, compare.Markdown(report))
				return err
			default:
				return compare.WriteHuman(stdout, report)
			}
		},
	}

	cmd.Flags().StringVar(&sourcePath, "source", "", "source scan report directory or snapshot.json")
	cmd.Flags().StringVar(&targetPath, "target", "", "target scan report directory or snapshot.json")
	cmd.Flags().StringVarP(&output, "output", "o", "human", "output format: human, markdown, json")
	return cmd
}

func validCompareOutput(output string) bool {
	switch strings.ToLower(output) {
	case "human", "markdown", "json":
		return true
	default:
		return false
	}
}

type progress struct {
	w       io.Writer
	current int
}

func newProgress(w io.Writer) *progress {
	return &progress{w: w}
}

func (p *progress) Step(format string, args ...any) {
	if p == nil || p.w == nil {
		return
	}
	p.current++
	fmt.Fprintf(p.w, "◆ %02d %s\n", p.current, fmt.Sprintf(format, args...))
}

func (p *progress) Detail(format string, args ...any) {
	if p == nil || p.w == nil {
		return
	}
	fmt.Fprintf(p.w, "  ├─ %s\n", fmt.Sprintf(format, args...))
}

func (p *progress) Done(format string, args ...any) {
	if p == nil || p.w == nil {
		return
	}
	fmt.Fprintf(p.w, "✓ %s\n", fmt.Sprintf(format, args...))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
