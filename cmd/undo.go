package cmd

import (
	"github.com/edbarbera/gitle/internal/gitcmd"
	"github.com/edbarbera/gitle/internal/ui"
	"github.com/spf13/cobra"
)

var undoCmd = &cobra.Command{
	Use:     "undo",
	Short:   "Undo your last save",
	Long:    "Removes your most recent saved point but keeps all the file changes it contained,\nso nothing is lost. You can save again once you've fixed things up.",
	Args:    cobra.NoArgs,
	PreRunE: requireRepo,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !gitcmd.HasCommits() {
			ui.Info("There's nothing to undo yet — you haven't saved anything.")
			return nil
		}

		last, err := gitcmd.Capture("log", "-1", "--pretty=%s")
		if err != nil {
			return err
		}

		ui.Warn("This will undo your last save: %q", last)
		ui.Hint("Your file changes are kept — only the saved point is removed.")
		if !ui.Confirm("Undo it?") {
			ui.Info("Left everything as it was.")
			return nil
		}

		if _, err := gitcmd.Capture("rev-parse", "HEAD~1"); err != nil {
			// No parent: this is the very first save. Remove the pointer so the
			// repo goes back to "nothing saved yet"; the files stay untouched.
			if err := gitcmd.Run("update-ref", "-d", "HEAD"); err != nil {
				return err
			}
		} else {
			// --soft keeps every change staged in the working tree; nothing is lost.
			if err := gitcmd.Run("reset", "--soft", "HEAD~1"); err != nil {
				return err
			}
		}
		ui.Success("Undid your last save. Your changes are still here.")
		ui.Hint("Save again with %s when you're ready.", ui.Bold(`gitle save "..."`))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(undoCmd)
}
