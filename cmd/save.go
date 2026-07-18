package cmd

import (
	"github.com/edbarbera/gitle/internal/ai"
	"github.com/edbarbera/gitle/internal/gitcmd"
	"github.com/edbarbera/gitle/internal/ops"
	"github.com/edbarbera/gitle/internal/ui"
	"github.com/spf13/cobra"
)

var (
	saveAll bool
	saveAI  bool
)

var saveCmd = &cobra.Command{
	Use:   "save [\"what you changed\"]",
	Short: "Save a snapshot of your work",
	Long: `Records your changes as a saved point you can always come back to.

In a terminal, gitle first shows a checklist of what changed so you can pick
exactly which files to include, then asks for a description. Use --all to skip
the checklist and save everything. Use --ai to have a free AI model draft that
description for you (needs OPENROUTER_API_KEY — see openrouter.ai for a free
key); you can still edit or replace whatever it suggests. Git calls the
result a commit.`,
	Example: `  gitle save
  gitle save "fixed the login bug"
  gitle save --all "quick save of everything"
  gitle save --ai`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: requireRepo,
	RunE: func(cmd *cobra.Command, args []string) error {
		message := ""
		if len(args) == 1 {
			message = args[0]
		}

		changes, err := ops.Changes()
		if err != nil {
			return err
		}
		if len(changes) == 0 {
			ui.Info("Nothing to save — your work is already up to date.")
			return nil
		}

		var paths []string
		if saveAll {
			// Skip the checklist: include every change.
			paths = ops.Paths(changes)
		} else {
			// Let the user pick which files to include (all ticked by default).
			// Without a terminal this returns everything, preserving "save all".
			picked := ui.Pick("Which changes do you want to save?", pickLabels(changes))
			if len(picked) == 0 {
				ui.Info("Nothing selected — nothing was saved.")
				return nil
			}
			paths = make([]string, len(picked))
			for i, idx := range picked {
				paths[i] = changes[idx].Path
			}
		}

		// Safety rail: flag secrets / oversized files before they're committed.
		if !reviewRisks(paths) {
			ui.Info("Nothing was saved.")
			return nil
		}

		// Stage before asking for a message, so an --ai suggestion can see a
		// real diff and nothing unpicked sneaks into the eventual commit.
		if err := ops.Stage(paths); err != nil {
			return err
		}

		if message == "" {
			if !ui.Interactive() {
				ui.Error("Please describe what you changed.")
				ui.Hint("Example: %s", ui.Bold(`gitle save "fixed the login bug"`))
				return errSilent
			}
			suggestion := ""
			if saveAI {
				suggestion = suggestMessage()
			}
			message = ui.Ask("Describe what you changed:", suggestion)
			if message == "" {
				ui.Error("A short description is needed to save.")
				return errSilent
			}
		}

		result, err := ops.Commit(message, paths)
		if err != nil {
			return err
		}

		ui.Success("Saved %d file(s): %q", len(result.Paths), result.Message)
		if result.Leftover {
			ui.Hint("Some changes were left unsaved — run %s again when ready.", ui.Bold("gitle save"))
		}
		if result.HasUpstream {
			ui.Hint("Send it online with %s.", ui.Bold("gitle send"))
		} else {
			ui.Hint("Share it online with %s once you're ready.", ui.Bold("gitle send"))
		}
		return nil
	},
}

// suggestMessage asks the ai package for a draft commit message from the
// staged diff. Every failure — no key, network trouble, an odd response —
// is swallowed here: --ai must never be able to block or break a save, it
// can only pre-fill the same prompt the user would otherwise see blank.
func suggestMessage() string {
	if !ai.Available() {
		ui.Hint("Set %s to let gitle draft this for you (free key at openrouter.ai).", ui.Bold("OPENROUTER_API_KEY"))
		return ""
	}
	diff, err := gitcmd.DiffCached()
	if err != nil || diff == "" {
		return ""
	}

	var msg string
	// The model call crosses the network, so show something moving rather
	// than letting the terminal sit there looking hung.
	_ = ui.Spinner("Drafting a description...", func() error {
		var err error
		msg, err = ai.SuggestMessage(diff)
		return err
	})
	return msg
}

func init() {
	saveCmd.Flags().BoolVarP(&saveAll, "all", "a", false, "save every change without showing the checklist")
	saveCmd.Flags().BoolVar(&saveAI, "ai", false, "draft the description for you (needs a free OPENROUTER_API_KEY)")
	rootCmd.AddCommand(saveCmd)
}
