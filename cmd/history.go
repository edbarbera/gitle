package cmd

import (
	"github.com/edbarbera/gitle/internal/gitcmd"
	"github.com/edbarbera/gitle/internal/ui"
	"github.com/spf13/cobra"
)

var historyCmd = &cobra.Command{
	Use:     "history",
	Short:   "See your saved points over time",
	Long:    "Shows the trail of every saved point, newest first, so you can see how the\nproject changed. Git calls this the log.",
	Args:    cobra.NoArgs,
	PreRunE: requireRepo,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !gitcmd.HasCommits() {
			ui.Info("No history yet — save your first snapshot with %s.", ui.Bold(`gitle save "..."`))
			return nil
		}
		return gitcmd.Run("log", "--oneline", "--graph", "--decorate", "--all")
	},
}

func init() {
	rootCmd.AddCommand(historyCmd)
}
