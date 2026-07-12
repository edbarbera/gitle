package cmd

import (
	"github.com/edbarbera/gitle/internal/gitcmd"
	"github.com/edbarbera/gitle/internal/ui"
	"github.com/spf13/cobra"
)

var grabCmd = &cobra.Command{
	Use:     "grab",
	Short:   "Grab the latest work from online",
	Long:    "Downloads everyone's latest saved work from the shared copy online and blends\nit with yours. Git calls this a pull.",
	Args:    cobra.NoArgs,
	PreRunE: requireRepo,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Unsaved changes can collide with incoming work. Ask the user to save
		// first so nothing is lost.
		if gitcmd.HasChanges() {
			ui.Warn("You have unsaved changes.")
			ui.Hint("Save them first with %s, then grab again.", ui.Bold(`gitle save "..."`))
			return errSilent
		}

		ui.Info("Grabbing the latest work from online...")
		// --rebase replays your saves on top of the latest, keeping history tidy.
		if err := gitcmd.Run("pull", "--rebase"); err != nil {
			ui.Warn("Some changes clashed and need a person to sort out.")
			ui.Hint("This is normal on shared projects. Ask a teammate to help merge,")
			ui.Hint("or undo the attempt with %s.", ui.Bold("git rebase --abort"))
			return errSilent
		}

		ui.Success("You're up to date with everyone's latest work.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(grabCmd)
}
