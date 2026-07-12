package cmd

import (
	"github.com/edbarbera/gitle/internal/gitcmd"
	"github.com/edbarbera/gitle/internal/ui"
	"github.com/spf13/cobra"
)

var branchesCmd = &cobra.Command{
	Use:     "branches",
	Short:   "List the separate lines of work",
	Long:    "Shows every branch — a separate line of work you can switch between. The one\nwith a * is where you are now.",
	Args:    cobra.NoArgs,
	PreRunE: requireRepo,
	RunE: func(cmd *cobra.Command, args []string) error {
		current := gitcmd.CurrentBranch()
		if current != "" {
			ui.Info("You're currently on: %s", ui.Bold(current))
		}
		if err := gitcmd.Run("branch", "-a"); err != nil {
			return err
		}
		ui.Hint("Switch with %s, or start a new one with %s.",
			ui.Bold("gitle switch <name>"), ui.Bold("gitle new-branch <name>"))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(branchesCmd)
}
