package cmd

import (
	"errors"
	"os"

	"github.com/edbarbera/gitle/internal/gitcmd"
	"github.com/edbarbera/gitle/internal/tui"
	"github.com/edbarbera/gitle/internal/ui"
	"github.com/spf13/cobra"
)

// errSilent signals that a command already printed its own message and
// Execute should exit non-zero without printing anything further.
var errSilent = errors.New("")

var rootCmd = &cobra.Command{
	Use:   "gitle",
	Short: "gitle — git made friendly",
	Long: `gitle is a friendly wrapper around git.

It gives everyday version-control tasks plain-English names and keeps you on
good habits automatically, so you can save and share your work without
memorising git.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          cobra.NoArgs,
	// Every command needs git on PATH.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if !gitcmd.Available() {
			return gitcmd.ErrGitMissing
		}
		return nil
	},
	// Bare `gitle` opens the dashboard in a terminal, and falls back to the
	// command overview anywhere else — piped, scripted, or redirected, where
	// a full-screen interface would be meaningless or actively unhelpful.
	RunE: func(cmd *cobra.Command, args []string) error {
		if !ui.Interactive() {
			renderHelp()
			return nil
		}
		if !gitcmd.InRepo() {
			// Nothing to show a dashboard of yet. Point at the setup wizard
			// rather than opening an empty screen.
			ui.Info("This folder isn't set up for gitle yet.")
			ui.Hint("Run %s here to get started.", ui.Bold("gitle start"))
			return nil
		}
		return tui.Run()
	},
}

// Execute runs the CLI and translates any error into a friendly message.
// version is the resolved release version, shown by `gitle --version`.
func Execute(version string) {
	rootCmd.Version = version
	rootCmd.SetVersionTemplate("gitle {{.Version}}\n")
	if err := rootCmd.Execute(); err != nil {
		if err == errSilent {
			os.Exit(1)
		}
		if err == gitcmd.ErrGitMissing {
			ui.Error("git is not installed on this computer.")
			ui.Hint("gitle needs git under the hood. Install it from https://git-scm.com/downloads")
		} else {
			ui.Error("%s", err)
		}
		os.Exit(1)
	}
}

// requireRepo is a shared guard for commands that must run inside a repo.
func requireRepo(cmd *cobra.Command, args []string) error {
	if !gitcmd.InRepo() {
		ui.Error("This folder isn't set up for gitle yet.")
		ui.Hint("Run %s here first to start tracking your work.", ui.Bold("gitle start"))
		os.Exit(1)
	}
	return nil
}
