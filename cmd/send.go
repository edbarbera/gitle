package cmd

import (
	"github.com/edbarbera/gitle/internal/gitcmd"
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

		if remotes, err := gitcmd.Capture("remote"); err != nil || remotes == "" {
			ui.Error("No online location is set up for this project yet.")
			ui.Hint("Ask whoever set up the project for the link, then connect it with:")
			ui.Hint("  %s", ui.Bold("git remote add origin <link>"))
			return errSilent
		}

		branch := gitcmd.CurrentBranch()
		ui.Info("Sending your work online...")

		var err error
		if gitcmd.HasUpstream() {
			err = gitcmd.Run("push")
		} else {
			// First push on this branch: remember the destination for next time.
			err = gitcmd.Run("push", "-u", "origin", branch)
		}
		if err != nil {
			return err
		}

		ui.Success("Sent everything online.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(sendCmd)
}
