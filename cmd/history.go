package cmd

import (
	"github.com/edbarbera/gitle/internal/ops"
	"github.com/edbarbera/gitle/internal/ui"
	"github.com/spf13/cobra"
)

var historyLimit int

var historyCmd = &cobra.Command{
	Use:     "history",
	Short:   "See your saved points over time",
	Long:    "Shows the trail of every saved point, newest first, so you can see how the\nproject changed. Git calls this the log.",
	Args:    cobra.NoArgs,
	PreRunE: requireRepo,
	RunE: func(cmd *cobra.Command, args []string) error {
		commits, err := ops.History(historyLimit)
		if err != nil {
			return err
		}
		if len(commits) == 0 {
			ui.Info("No history yet — save your first snapshot with %s.", ui.Bold(`gitle save "..."`))
			return nil
		}

		for _, c := range commits {
			line := ui.Dim(c.Hash) + "  " + c.Subject
			if c.Refs != "" {
				line += " " + ui.Cyan("("+c.Refs+")")
			}
			ui.Plain("%s", line)
			ui.Plain("%s", ui.Dim("          "+c.Author+", "+c.When))
		}
		return nil
	},
}

func init() {
	historyCmd.Flags().IntVarP(&historyLimit, "number", "n", 20, "how many saved points to show (0 for all)")
	rootCmd.AddCommand(historyCmd)
}
