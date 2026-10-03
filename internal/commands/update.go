package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"

	"github.com/wahyurudiyan/sunny-go/internal/selfupdate"
)

var (
	updateCheck   bool
	updateVersion string
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update sgo itself to the latest (or a specific) version",
	Long: `Update the sgo binary via 'go install ` + selfupdate.Package + `@<version>' —
the same command README's own Install section already tells you to run
by hand. Requires the Go toolchain on PATH, same as installing sgo in
the first place did.

With no flags, resolves the latest published version from the Go module
proxy, compares it against this binary's own version, and either
reports "already up to date" or reinstalls. '--check' reports without
installing anything. '--version' installs a specific version instead of
latest (including a downgrade), skipping the up-to-date check.

Only reports something meaningful once sgo itself has real tagged
releases published — see ARCHITECTURE.md §23.

Examples:
  sgo update
  sgo update --check
  sgo update --version v0.2.0
`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runUpdate(updateCheck, updateVersion); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func runUpdate(checkOnly bool, explicitVersion string) error {
	current, ok := selfupdate.NormalizeVersion(Version)
	if !ok {
		return fmt.Errorf("sgo's own build version %q is not a valid semver version", Version)
	}

	if explicitVersion != "" {
		target, ok := selfupdate.NormalizeVersion(explicitVersion)
		if !ok {
			return fmt.Errorf("%q is not a valid version (expected something like v0.2.0 or 0.2.0)", explicitVersion)
		}

		if checkOnly {
			fmt.Printf("Requested version: %s (currently running %s)\n", target, current)
			return nil
		}

		return installAndVerify(current, target)
	}

	latest, found, err := selfupdate.Latest(selfupdate.Module)
	if err != nil {
		return fmt.Errorf("checking for the latest sgo version: %w", err)
	}
	if !found {
		fmt.Printf("No tagged releases found for %s yet — nothing to compare against.\n", selfupdate.Module)
		return nil
	}

	upToDate := semver.Compare(current, latest) >= 0

	if checkOnly {
		if upToDate {
			fmt.Printf("Already up to date (%s).\n", current)
		} else {
			fmt.Printf("A new version is available: %s -> %s\n", current, latest)
		}
		return nil
	}

	if upToDate {
		fmt.Printf("Already up to date (%s).\n", current)
		return nil
	}

	return installAndVerify(current, latest)
}

func installAndVerify(current, target string) error {
	fmt.Printf("Updating %s -> %s ...\n", current, target)

	if err := selfupdate.Install(os.Stdout, selfupdate.Package, target); err != nil {
		return err
	}

	fmt.Printf("✅ Updated to %s\n", target)
	warnIfBinMismatch()

	return nil
}

// warnIfBinMismatch prints a warning, never a fatal error, when the
// directory `go install` just wrote to differs from the directory this
// currently-running binary lives in — the update itself already
// succeeded either way. A mismatch means the user has this sgo on PATH
// from somewhere other than their GOBIN (a copy, a symlink, a second Go
// environment), so re-running 'sgo update' just installed a new binary
// they aren't actually running yet.
func warnIfBinMismatch() {
	installDir, running, mismatched, err := selfupdate.BinMismatch("")
	if err != nil {
		fmt.Printf("⚠️  Couldn't double-check the install location: %v\n", err)
		return
	}
	if !mismatched {
		return
	}

	fmt.Printf(
		"⚠️  Installed the update to %s, but this sgo is running from %s — update your PATH, or copy the new binary over this one, to actually pick it up.\n",
		filepath.Join(installDir, filepath.Base(running)), running,
	)
}

func init() {
	updateCmd.Flags().BoolVar(&updateCheck, "check", false, "Report the latest available version without installing it")
	updateCmd.Flags().StringVar(&updateVersion, "version", "", "Install a specific version instead of latest")

	rootCmd.AddCommand(updateCmd)
}
