package cmd

import (
	"github.com/edbarbera/gitle/internal/gitcmd"
	"github.com/edbarbera/gitle/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:     "status",
	Short:   "See what's going on right now",
	Long:    "A plain-English summary of where you are: which line of work you're on and\nwhether you have changes waiting to be saved.",
	Args:    cobra.NoArgs,
	PreRunE: requireRepo,
	RunE: func(cmd *cobra.Command, args []string) error {
		if branch := gitcmd.CurrentBranch(); branch != "" {
			ui.Info("You're on the %s line of work.", ui.Bold(branch))
		}

		if !gitcmd.HasCommits() {
			ui.Info("You haven't saved anything yet.")
			ui.Hint("Make your first save with %s.", ui.Bold(`gitle save "first version"`))
			return nil
		}

		if gitcmd.HasChanges() {
			ui.Warn("You have unsaved changes.")
			ui.Hint("Save them with %s.", ui.Bold(`gitle save "..."`))
		} else {
			ui.Success("Everything is saved.")
		}

		if !gitcmd.HasUpstream() {
			ui.Hint("This work isn't online yet. Put it online with %s.", ui.Bold("gitle send"))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
