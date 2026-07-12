package cmd

import (
	"os"

	"github.com/edbarbera/gitle/internal/gitcmd"
	"github.com/edbarbera/gitle/internal/ui"
	"github.com/spf13/cobra"
)

const onboardSteps = 5

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Set up this folder (guided)",
	Long: `Walks you through setting up version control for this folder: naming yourself,
keeping junk out, making a first save, and connecting to GitHub.

Run it once at the very beginning. It's safe to run again — it skips anything
that's already done.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.Banner()

		if err := stepInit(); err != nil {
			return err
		}
		if err := stepIdentity(); err != nil {
			return err
		}
		if err := stepGitignore(); err != nil {
			return err
		}
		if err := stepFirstSave(); err != nil {
			return err
		}
		if err := stepConnect(); err != nil {
			return err
		}

		ui.Celebrate("All set — this folder is now yours to save and share!")
		ui.Hint("Save your work anytime:   %s", ui.Bold(`gitle save "what changed"`))
		ui.Hint("See where you stand:      %s", ui.Bold("gitle status"))
		ui.Hint("Send it online:           %s", ui.Bold("gitle send"))
		return nil
	},
}

// stepInit creates the repo (or notes it already exists).
func stepInit() error {
	ui.Step(1, onboardSteps, "Start tracking your work")
	if gitcmd.InRepo() {
		ui.Info("This folder is already being tracked — nice, carrying on.")
		return nil
	}
	// -b main names the default line of work "main", the modern convention.
	if err := gitcmd.Run("init", "-b", "main"); err != nil {
		return err
	}
	ui.Success("Done! gitle is now watching this folder for changes.")
	return nil
}

// stepIdentity makes sure git knows who is making saves.
func stepIdentity() error {
	ui.Step(2, onboardSteps, "Who are you?")
	name := gitcmd.ConfigGet("user.name")
	email := gitcmd.ConfigGet("user.email")

	if name != "" && email != "" {
		ui.Success("Your saves will be signed as %s <%s>.", ui.Bold(name), email)
		return nil
	}

	if !ui.IsInteractive() {
		ui.Warn("No name/email set yet, so saves can't be signed.")
		ui.Hint("Set them once with:")
		ui.Hint("  git config --global user.name \"Your Name\"")
		ui.Hint("  git config --global user.email \"you@example.com\"")
		return nil
	}

	ui.Info("This gets stamped on everything you save, so people know it was you.")
	name = ui.Ask("What's your name?", name)
	email = ui.Ask("What's your email?", email)

	if name == "" || email == "" {
		ui.Warn("Skipped — you can set your name and email later.")
		return nil
	}
	if err := gitcmd.ConfigSetGlobal("user.name", name); err != nil {
		return err
	}
	if err := gitcmd.ConfigSetGlobal("user.email", email); err != nil {
		return err
	}
	ui.Success("Thanks, %s! Your saves will be signed from now on.", ui.Bold(name))
	return nil
}

// stepGitignore offers a starter .gitignore tuned to the project.
func stepGitignore() error {
	ui.Step(3, onboardSteps, "Keep junk and secrets out")
	if exists(".gitignore") {
		ui.Info("You already have a .gitignore — leaving it as-is.")
		return nil
	}

	p := detectProject()
	if p.matched {
		ui.Info("Looks like a %s project.", ui.Bold(p.name))
	}
	ui.Hint("A .gitignore tells gitle which files to skip — like passwords and clutter.")

	if !ui.ConfirmDefault("Create a starter .gitignore?", true) {
		ui.Info("Skipped — you can add one later.")
		return nil
	}
	if err := os.WriteFile(".gitignore", []byte(p.gitignoreBody()), 0o644); err != nil {
		return err
	}
	ui.Success("Created a .gitignore for you.")
	return nil
}

// stepFirstSave offers to make the very first snapshot.
func stepFirstSave() error {
	ui.Step(4, onboardSteps, "Make your first save")
	if gitcmd.HasCommits() {
		ui.Info("You've already saved before — skipping.")
		return nil
	}
	if !gitcmd.HasChanges() {
		ui.Info("No files here yet. Add some, then run %s.", ui.Bold(`gitle save "..."`))
		return nil
	}

	if !ui.IsInteractive() {
		ui.Hint("Save your work anytime with %s.", ui.Bold(`gitle save "..."`))
		return nil
	}
	if !ui.ConfirmDefault("Save everything here as your first snapshot?", true) {
		ui.Info("No problem — save whenever you're ready with %s.", ui.Bold(`gitle save "..."`))
		return nil
	}

	message := ui.Ask("Describe it in a few words:", "first version")
	if err := gitcmd.Run("add", "-A"); err != nil {
		return err
	}
	if err := gitcmd.Run("commit", "-m", message); err != nil {
		return err
	}
	ui.Success("Saved: %q", message)
	return nil
}

// stepConnect optionally wires up a GitHub remote.
func stepConnect() error {
	ui.Step(5, onboardSteps, "Connect to GitHub (optional)")
	if gitcmd.HasRemote() {
		ui.Info("Already connected to an online home — you're good.")
		return nil
	}

	if !ui.IsInteractive() {
		ui.Hint("Connect later with %s.", ui.Bold("git remote add origin <link>"))
		return nil
	}

	ui.Info("Have a repo on GitHub? Paste its link to connect it.")
	ui.Hint("No repo yet? Just press Enter — make one anytime at https://github.com/new")
	link := ui.Ask("GitHub link (Enter to skip):", "")
	if link == "" {
		ui.Info("Skipped — you can connect it later.")
		return nil
	}

	if err := gitcmd.AddRemote("origin", link); err != nil {
		ui.Warn("Couldn't connect that link: %s", err)
		ui.Hint("Double-check it and try %s.", ui.Bold("git remote add origin <link>"))
		return nil
	}
	ui.Success("Connected! Put your work online anytime with %s.", ui.Bold("gitle send"))
	return nil
}

func init() {
	rootCmd.AddCommand(startCmd)
}
