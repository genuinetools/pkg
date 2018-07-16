package cli_test

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/genuinetools/pkg/cli"
)

func ExampleNewProgram_withSingleAction() {
	// Create a new cli program.
	p := cli.NewProgram()
	p.Name = "yo"
	p.Description = `A tool that prints "yo"`

	// Set the GitCommit and Version.
	p.GitCommit = "ef2f64f"
	p.Version = "v0.1.0"

	// Setup the global flags.
	var (
		debug bool
	)
	p.FlagSet = flag.NewFlagSet("global", flag.ExitOnError)
	p.FlagSet.BoolVar(&debug, "d", false, "enable debug logging")

	// Set the before function.
	p.Before = func(ctx context.Context) error {
		// Set the log level.
		if debug {
			// Setup your logger here...
		}

		return nil
	}

	// Set the main program action.
	p.Action = func(ctx context.Context, args []string) error {
		// On ^C, or SIGTERM handle exit.
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		signal.Notify(c, syscall.SIGTERM)
		go func() {
			for sig := range c {
				log.Printf("Received %s, exiting.", sig.String())
				os.Exit(0)
			}
		}()

		fmt.Fprintln(os.Stdout, "yo")
		return nil
	}

	// Run our program.
	p.Run()

}
