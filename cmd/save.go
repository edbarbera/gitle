package cmd

import (
	"github.com/edbarbera/gitle/internal/gitcmd"
	"github.com/edbarbera/gitle/internal/ui"
	"github.com/spf13/cobra"
)

var saveCmd = &cobra.Command{
	Use:     "save \"what you changed\"",
	Short:   "Save a snapshot of your work",
	Long:    "Records all your current changes as a saved point you can always come back to.\nGit calls this a commit.",
	Example: `  gitle save "fixed the login bug"`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: requireRepo,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 || args[0] == "" {
			ui.Error("Please describe what you changed.")
			ui.Hint("Example: %s", ui.Bold(`gitle save "fixed the login bug"`))
			return errSilent
		}
		message := args[0]

		if !gitcmd.HasChanges() {
			ui.Info("Nothing to save — your work is already up to date.")
			return nil
		}

		if err := gitcmd.Run("add", "-A"); err != nil {
			return err
		}
		if err := gitcmd.Run("commit", "-m", message); err != nil {
			return err
		}

		ui.Success("Saved: %q", message)
		if !gitcmd.HasUpstream() {
			ui.Hint("Share it online with %s once you're ready.", ui.Bold("gitle send"))
		} else {
			ui.Hint("Send it online with %s.", ui.Bold("gitle send"))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(saveCmd)
}
