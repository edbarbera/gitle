package cmd

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/edbarbera/gitle/internal/gitcmd"
	"github.com/edbarbera/gitle/internal/ops"
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

		// Rail: pushing straight to a shared branch like main is worth pausing on.
		if ops.ProtectedBranches[branch] {
			ui.Warn("You're sending straight to '%s'.", branch)
			ui.Hint("On shared projects it's safer to make a branch first (%s) and send that.",
				ui.Bold("gitle new-branch <name>"))
			if ui.Interactive() && !ui.ConfirmDefault("Send to "+branch+" anyway?", false) {
				ui.Info("Held off. Start a branch with %s.", ui.Bold("gitle new-branch <name>"))
				return errSilent
			}
		}

		// The spinner draws over the terminal, so git must not try to prompt
		// for credentials underneath it. Without a spinner (piped, scripted)
		// git is free to ask as it always has.
		opts := ops.SendOptions{AllowTerminalPrompt: !ui.Interactive()}

		var result ops.SendResult
		err := ui.Spinner("Sending your work online...", func() error {
			var err error
			result, err = ops.Send(opts)
			return err
		})
		if err != nil {
			return explainSendError(err)
		}

		ui.Success("Sent everything online.")
		if result.FirstPush {
			ui.Hint("'%s' now has an online home — future sends go there automatically.", result.Branch)
		}
		return nil
	},
}

// explainSendError turns a classified send failure into plain English and the
// right next step — most importantly, "someone else sent work first".
func explainSendError(err error) error {
	var sendErr *ops.SendError
	if !errors.As(err, &sendErr) {
		return err
	}
	switch sendErr.Problem {
	case ops.SendRejected:
		ui.Error("Couldn't send — there's newer work online you don't have yet.")
		ui.Hint("Grab it first with %s, then send again.", ui.Bold("gitle grab"))
	case ops.SendAuth:
		ui.Error("Couldn't send — GitHub needs you to sign in.")
		ui.Hint("If you use the gh tool, run %s once, then try again.", ui.Bold("gh auth login"))
	default:
		ui.Error("Couldn't send your work.")
		if sendErr.Detail != "" {
			ui.Hint("git said: %s", sendErr.Detail)
		}
	}
	return errSilent
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

	if !ui.Interactive() {
		ui.Hint("Create one with %s, or connect an existing repo with %s.",
			ui.Bold("gh repo create"), ui.Bold("git remote add origin <link>"))
		return errSilent
	}

	if !ui.ConfirmDefault("Create a new GitHub repo for this project now?", true) {
		ui.Info("No problem — connect one later with %s.", ui.Bold("git remote add origin <link>"))
		return errSilent
	}

	name := ui.Ask("What should it be called?", currentDirName())
	if name == "" {
		ui.Info("No name given — nothing was created.")
		return errSilent
	}
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
