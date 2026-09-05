// Package cli implements the Teleskope command-line interface.
package cli

import (
	"flag"
	"fmt"
	"io"
)

// Version is overridden at build time for release builds.
var Version = "dev"

// Run executes the CLI and returns its process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("teleskope", flag.ContinueOnError)
	flags.SetOutput(stderr)
	version := flags.Bool("version", false, "print version")
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Teleskope — EKS and Kubernetes inventory and resource topology")
		fmt.Fprintln(flags.Output(), "\nUsage: teleskope [options]")
		fmt.Fprintln(flags.Output(), "\nOptions:")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "unexpected argument: %s\n", flags.Arg(0))
		return 2
	}
	if *version {
		fmt.Fprintln(stdout, "teleskope", Version)
		return 0
	}
	flags.SetOutput(stdout)
	flags.Usage()
	return 0
}
