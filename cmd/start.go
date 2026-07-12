package cmd

import (
	"github.com/edbarbera/gitle/internal/gitcmd"
	"github.com/edbarbera/gitle/internal/ui"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start tracking this folder",
	Long:  "Sets up the current folder so gitle can track your work. Only needed once,\nat the very beginning. Git calls this initialising a repository.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if gitcmd.InRepo() {
			ui.Info("This folder is already set up — you're good to go.")
			ui.Hint("Save your work anytime with %s.", ui.Bold(`gitle save "..."`))
			return nil
		}
		// -b main names the default line of work "main", the modern convention.
		if err := gitcmd.Run("init", "-b", "main"); err != nil {
			return err
		}
		ui.Success("This folder is now tracked by gitle.")
		ui.Hint("Make your first save with %s.", ui.Bold(`gitle save "first version"`))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
