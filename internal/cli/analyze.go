package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/analysis"
	"github.com/kuraudo-lab/teleskope/internal/compare"
	"github.com/spf13/cobra"
)

func newAnalyzeCommand(stdout, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Generate optional LLM analysis from existing scan results",
	}
	cmd.AddCommand(newAnalyzeScanCommand(stdout, stderr))
	cmd.AddCommand(newAnalyzeCompareCommand(stdout, stderr))
	return cmd
}

func newAnalyzeScanCommand(stdout, stderr io.Writer) *cobra.Command {
	var configPath string
	var output string
	var timeout time.Duration
	var webSearch bool
	var contextOnly bool

	cmd := &cobra.Command{
		Use:   "scan <scan-dir-or-snapshot.json>",
		Short: "Analyze one scan result with an OpenAI-compatible model",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !validAnalysisOutput(output) {
				return fmt.Errorf("unsupported output %q; expected human, markdown, json, or context", output)
			}
			progress := newProgress(stderr)
			progress.Step("loading scan result")
			snapshot, err := compare.LoadSnapshot(args[0])
			if err != nil {
				return err
			}
			req := analysis.Request{UseCase: analysis.UseCaseScan, Snapshot: snapshot, WebSearch: webSearch}
			if strings.EqualFold(output, "context") || contextOnly {
				data, err := analysis.BuildContext(req)
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(stdout, string(data))
				return err
			}
			return runLLMAnalysis(cmd.Context(), stdout, stderr, progress, configPath, timeout, req, output)
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "", "path to config file (default $HOME/.teleskope/config.yaml)")
	cmd.Flags().StringVarP(&output, "output", "o", "human", "output format: human, markdown, json, context")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "LLM request timeout override")
	cmd.Flags().BoolVar(&webSearch, "web-search", false, "allow the prompt to request web-search-backed conclusions when the provider supports it")
	cmd.Flags().BoolVar(&contextOnly, "context", false, "print the trimmed model context without calling the LLM provider")
	return cmd
}

func newAnalyzeCompareCommand(stdout, stderr io.Writer) *cobra.Command {
	var configPath string
	var sourcePath string
	var targetPath string
	var output string
	var timeout time.Duration
	var webSearch bool
	var contextOnly bool
	var advisory bool

	cmd := &cobra.Command{
		Use:   "compare [--source <scan-dir-or-snapshot.json>] [--target <scan-dir-or-snapshot.json>]",
		Short: "Analyze migration differences with an OpenAI-compatible model",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 2 {
				return fmt.Errorf("analyze compare accepts at most two positional paths")
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
			if !validAnalysisOutput(output) {
				return fmt.Errorf("unsupported output %q; expected human, markdown, json, or context", output)
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
			progress.Step("running deterministic comparison")
			report := compare.Analyze(source, target)
			useCase := analysis.UseCaseCompare
			if advisory {
				useCase = analysis.UseCaseAdvisory
			}
			req := analysis.Request{UseCase: useCase, Source: source, Target: target, CompareReport: &report, WebSearch: webSearch}
			if strings.EqualFold(output, "context") || contextOnly {
				data, err := analysis.BuildContext(req)
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(stdout, string(data))
				return err
			}
			return runLLMAnalysis(cmd.Context(), stdout, stderr, progress, configPath, timeout, req, output)
		},
	}
	cmd.Flags().StringVar(&sourcePath, "source", "", "source scan report directory or snapshot.json")
	cmd.Flags().StringVar(&targetPath, "target", "", "target scan report directory or snapshot.json")
	cmd.Flags().StringVar(&configPath, "config", "", "path to config file (default $HOME/.teleskope/config.yaml)")
	cmd.Flags().StringVarP(&output, "output", "o", "human", "output format: human, markdown, json, context")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "LLM request timeout override")
	cmd.Flags().BoolVar(&webSearch, "web-search", false, "allow the prompt to request web-search-backed conclusions when the provider supports it")
	cmd.Flags().BoolVar(&contextOnly, "context", false, "print the trimmed model context without calling the LLM provider")
	cmd.Flags().BoolVar(&advisory, "advisory", false, "use the web advisory prompt instead of the CLI compare prompt")
	return cmd
}

func runLLMAnalysis(ctx context.Context, stdout, _ io.Writer, progress *progress, configPath string, timeout time.Duration, req analysis.Request, output string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	progress.Step("loading LLM configuration")
	cfg, err := analysis.LoadConfig(configPath)
	if err != nil {
		return err
	}
	if timeout > 0 {
		cfg.LLM.Timeout = timeout
	}
	if cfg.LLM.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.LLM.Timeout)
		defer cancel()
	}
	if modelContext, err := analysis.BuildContext(req); err == nil {
		progress.Detail("model context=%d bytes model=%s timeout=%s", len(modelContext), cfg.LLM.Model, cfg.LLM.Timeout)
	}
	progress.Step("requesting LLM analysis")
	result, err := analysis.NewOpenAICompatible(cfg.LLM).Analyze(ctx, req)
	if err != nil {
		return err
	}
	switch strings.ToLower(output) {
	case "json":
		return analysis.WriteJSON(stdout, result)
	case "markdown":
		_, err = fmt.Fprint(stdout, analysis.Markdown(result))
		return err
	default:
		return analysis.WriteHuman(stdout, result)
	}
}

func validAnalysisOutput(output string) bool {
	switch strings.ToLower(output) {
	case "human", "markdown", "json", "context":
		return true
	default:
		return false
	}
}
