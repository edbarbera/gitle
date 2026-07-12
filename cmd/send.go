package cmd

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/edbarbera/gitle/internal/gitcmd"
	"github.com/edbarbera/gitle/internal/ui"
	"github.com/spf13/cobra"
)

var sendCmd = &cobra.Command{
	Use:     "send",
	Short:   "Send your saved work online",
	Long:    "Uploads your saved points to the shared copy online (for example GitHub),\nso teammates and backups stay in sync. Git calls this a push.",
	Args:    cobra.NoArgs,
	PreRunE: requireRepo,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !gitcmd.HasCommits() {
			ui.Info("Nothing to send yet — save some work first with %s.", ui.Bold(`gitle save "..."`))
			return nil
		}

		if !gitcmd.HasRemote() {
			return offerCreateRepo()
		}

		branch := gitcmd.CurrentBranch()
		ui.Info("Sending your work online...")

		var err error
		if gitcmd.HasUpstream() {
			err = gitcmd.Run("push")
		} else {
			// First push on this branch: remember the destination for next time.
			err = gitcmd.Run("push", "-u", "origin", branch)
		}
		if err != nil {
			return err
		}

		ui.Success("Sent everything online.")
		return nil
	},
}

// offerCreateRepo handles the "no online home yet" case. If GitHub's `gh` tool
// is available it offers to create the repo and push in one step; otherwise it
// explains the easiest way forward.
func offerCreateRepo() error {
	ui.Warn("This project isn't online yet.")

	if !ghAvailable() {
		ui.Hint("Easiest way: install GitHub's free tool from %s", ui.Bold("https://cli.github.com"))
		ui.Hint("Then run %s again and I'll offer to set it up for you.", ui.Bold("gitle send"))
		ui.Hint("Already have a repo online? Connect it with %s.", ui.Bold("git remote add origin <link>"))
		return errSilent
	}

	if !ui.IsInteractive() {
		ui.Hint("Create one with %s, or connect an existing repo with %s.",
			ui.Bold("gh repo create"), ui.Bold("git remote add origin <link>"))
		return errSilent
	}

	if !ui.ConfirmDefault("Create a new GitHub repo for this project now?", true) {
		ui.Info("No problem — connect one later with %s.", ui.Bold("git remote add origin <link>"))
		return errSilent
	}

	name := ui.Ask("What should it be called?", currentDirName())
	visibility := "--private"
	if !ui.ConfirmDefault("Keep it private?", true) {
		visibility = "--public"
	}

	ui.Info("Creating %s on GitHub and sending your work up...", ui.Bold(name))
	// --source=. adds this folder as origin; --push uploads current commits.
	if err := runGH("repo", "create", name, "--source=.", "--remote=origin", "--push", visibility); err != nil {
		ui.Error("Couldn't create the repo.")
		ui.Hint("If it says you're not logged in, run %s once, then try again.", ui.Bold("gh auth login"))
		return errSilent
	}

	ui.Success("Created and sent everything online! 🎉")
	return nil
}

// ghAvailable reports whether the GitHub CLI is installed.
func ghAvailable() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}

// runGH runs the GitHub CLI with the user's terminal attached, so its own
// prompts and output (including auth) are visible.
func runGH(args ...string) error {
	cmd := exec.Command("gh", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// currentDirName returns the current folder's name, a sensible default repo name.
func currentDirName() string {
	wd, err := os.Getwd()
	if err != nil {
		return "my-project"
	}
	return filepath.Base(wd)
}

func init() {
	rootCmd.AddCommand(sendCmd)
}
