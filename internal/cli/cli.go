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
	"github.com/kuraudo-lab/teleskope/internal/render"
	"github.com/spf13/cobra"
)

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

	cmd.AddCommand(newScanCommand(stdout))
	return cmd
}

func newScanCommand(stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Collect inventory from a target environment",
	}
	cmd.AddCommand(newScanEKSCommand(stdout))
	return cmd
}

func newScanEKSCommand(stdout io.Writer) *cobra.Command {
	var opts awseks.Options
	var output string
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "eks",
		Short: "Collect read-only EKS control plane inventory",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if opts.ClusterName == "" {
				return fmt.Errorf("--cluster is required")
			}
			if output == "" {
				output = "human"
			}
			if !validOutput(output) {
				return fmt.Errorf("unsupported output %q; expected human or json", output)
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

			snapshot, err := awseks.Collect(ctx, opts)
			if err != nil {
				return err
			}
			switch strings.ToLower(output) {
			case "json":
				return render.JSON(stdout, snapshot)
			default:
				return render.Human(stdout, snapshot)
			}
		},
	}

	cmd.Flags().StringVarP(&opts.ClusterName, "cluster", "c", "", "EKS cluster name")
	cmd.Flags().StringVarP(&output, "output", "o", "human", "output format: human, json")
	cmd.Flags().DurationVar(&timeout, "timeout", 2*time.Minute, "collection timeout")
	return cmd
}

func validOutput(output string) bool {
	switch strings.ToLower(output) {
	case "human", "json":
		return true
	default:
		return false
	}
}
