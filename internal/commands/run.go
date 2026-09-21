package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/wahyurudiyan/sunny-go/internal/dashboard"
	"github.com/wahyurudiyan/sunny-go/internal/run"
	"github.com/wahyurudiyan/sunny-go/internal/run/envsource"
)

var (
	runDebug     bool
	runDebugPort int
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the current project's service",
	Long: `Run the current project's service the way "go run" would, with
configuration loaded from .env (ARCHITECTURE.md §15) — a real
OS/CI environment variable already set always wins over a leftover
.env value, so it's never silently shadowed.

--debug additionally serves a 127.0.0.1-only dashboard showing every
configuration key in effect, its source, and whether it's writable.
Editing a writable value there writes it back to .env and restarts
the service with the new value in effect.

Example:
  sgo run
  sgo run --debug
  sgo run --debug --debug-port 4748
`,
	Run: func(cmd *cobra.Command, args []string) {
		dir, err := os.Getwd()
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

		cmdDir, err := run.FindCmdDir(dir)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

		sources := []envsource.Source{&envsource.DotEnvSource{Path: filepath.Join(dir, ".env")}}

		ctx := context.Background()
		env, err := envsource.Merge(ctx, sources)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

		sup := &run.Supervisor{ProjectDir: dir, CmdDir: cmdDir}
		if err := sup.Start(env); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

		if runDebug {
			fmt.Printf("🐞 sgo run --debug: dashboard at http://127.0.0.1:%d\n", runDebugPort)
			go func() {
				if err := dashboard.Serve(runDebugPort, sup, sources); err != nil {
					fmt.Printf("❌ dashboard error: %v\n", err)
				}
			}()

			// A dashboard-driven edit restarts the child (Supervisor.Restart),
			// which is documented as incompatible with also calling Wait: once
			// Restart replaces the child, the channel Wait is blocked on is
			// stale. So --debug's run loop just waits for Ctrl-C rather than
			// for a child that may have already been swapped out from under it.
			<-interrupt
			_ = sup.Stop()
			return
		}

		done := make(chan error, 1)
		go func() { done <- sup.Wait() }()

		select {
		case <-interrupt:
			_ = sup.Stop()
		case err := <-done:
			if err != nil {
				fmt.Printf("❌ %s exited: %v\n", cmdDir, err)
				os.Exit(1)
			}
		}
	},
}

func init() {
	runCmd.Flags().BoolVar(&runDebug, "debug", false, "Serve a localhost config dashboard alongside the service")
	runCmd.Flags().IntVar(&runDebugPort, "debug-port", 4748, "Port to bind the --debug dashboard to")
	rootCmd.AddCommand(runCmd)
}
