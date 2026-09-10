package cli

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kuraudo-lab/teleskope/internal/advisory"
	"github.com/spf13/cobra"
)

func newAdvisoryCommand(stdout, stderr io.Writer) *cobra.Command {
	var listen string
	cmd := &cobra.Command{
		Use:   "advisory",
		Short: "Serve a local web UI for comparing two scan results",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			listener, err := net.Listen("tcp", listen)
			if err != nil {
				return fmt.Errorf("listen: %w", err)
			}
			defer listener.Close()
			server := &http.Server{
				Handler:           advisory.Handler(),
				ReadHeaderTimeout: 5 * time.Second,
				WriteTimeout:      30 * time.Second,
				IdleTimeout:       60 * time.Second,
			}
			serverDone := make(chan error, 1)
			go func() { serverDone <- server.Serve(listener) }()
			fmt.Fprintf(stdout, "Migration advisory: http://%s\n", listener.Addr())
			fmt.Fprintf(stderr, "advisory serve starting listen=%s\n", listener.Addr())
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			select {
			case err = <-serverDone:
			case <-ctx.Done():
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
				if shutdownErr := server.Shutdown(shutdownCtx); shutdownErr != nil {
					_ = server.Close()
				}
				shutdownCancel()
				err = <-serverDone
			}
			if err == http.ErrServerClosed {
				return nil
			}
			return err
		},
	}
	cmd.Flags().StringVar(&listen, "listen", "127.0.0.1:8080", "HTTP listen address (no built-in authentication)")
	return cmd
}
