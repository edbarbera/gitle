package cmd

import (
	"errors"

	"github.com/edbarbera/gitle/internal/ops"
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
		err := ui.Spinner("Grabbing the latest work from online...", ops.Grab)

		switch {
		case err == nil:
			ui.Success("You're up to date with everyone's latest work.")
			return nil

		case errors.Is(err, ops.ErrUnsavedChanges):
			// Unsaved changes can collide with incoming work, so gitle asks
			// the user to save first rather than risk losing any.
			ui.Warn("You have unsaved changes.")
			ui.Hint("Save them first with %s, then grab again.", ui.Bold(`gitle save "..."`))
			return errSilent

		case errors.Is(err, ops.ErrConflict):
			ui.Warn("Some changes clashed with yours — that's normal on shared projects.")
			ui.Hint("Walk through them with %s.", ui.Bold("gitle fix-conflicts"))
			return errSilent

		case errors.Is(err, ops.ErrNoRemote):
			ui.Info("This project isn't online yet, so there's nothing to grab.")
			ui.Hint("Put it online with %s.", ui.Bold("gitle send"))
			return nil

		case errors.Is(err, ops.ErrNoUpstream):
			ui.Info("This line of work isn't online yet, so there's nothing to grab.")
			ui.Hint("Send it up first with %s.", ui.Bold("gitle send"))
			return nil

		default:
			ui.Error("Couldn't grab the latest work.")
			ui.Hint("git said: %s", err)
			return errSilent
		}
	},
}

func init() {
	rootCmd.AddCommand(grabCmd)
}
