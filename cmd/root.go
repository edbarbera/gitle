package cmd

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/edbarbera/gitle/internal/gitcmd"
	"github.com/edbarbera/gitle/internal/telemetry"
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
	// Every command needs git on PATH.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if !gitcmd.Available() {
			return gitcmd.ErrGitMissing
		}
		return nil
	},
}

// Execute runs the CLI and translates any error into a friendly message.
// version is the resolved release version, shown by `gitle --version`.
// otlpAuthHeader is the Grafana Cloud auth header baked in at build time;
// pass "" to run with telemetry disabled (e.g. local `go build`).
func Execute(version, otlpAuthHeader string) {
	rootCmd.Version = version
	rootCmd.SetVersionTemplate("gitle {{.Version}}\n")

	ctx := context.Background()
	shutdown := telemetry.Start(ctx, otlpAuthHeader, version)

	start := time.Now()
	matched, err := rootCmd.ExecuteC()
	telemetry.RecordInvocation(ctx, matched.CommandPath(), time.Since(start), errCategory(err))
	shutdown()

	if err != nil {
		if err == errSilent || err == errNotARepo {
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

// errCategory buckets an error into a small fixed label for telemetry.
// Message text is never sent — only the category.
func errCategory(err error) string {
	switch {
	case err == nil:
		return ""
	case err == gitcmd.ErrGitMissing:
		return "git_missing"
	case err == errNotARepo:
		return "not_a_repo"
	case err == errSilent:
		return "silent"
	default:
		return "command_error"
	}
}

// errNotARepo signals that a command needed a repo and didn't find one.
// requireRepo already prints the friendly message, so Execute treats this
// like errSilent: exit non-zero without printing anything further.
var errNotARepo = errors.New("")

// requireRepo is a shared guard for commands that must run inside a repo.
func requireRepo(cmd *cobra.Command, args []string) error {
	if !gitcmd.InRepo() {
		ui.Error("This folder isn't set up for gitle yet.")
		ui.Hint("Run %s here first to start tracking your work.", ui.Bold("gitle start"))
		return errNotARepo
	}
	return nil
}
