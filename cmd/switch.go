package cmd

import (
	"errors"

	"github.com/edbarbera/gitle/internal/gitcmd"
	"github.com/edbarbera/gitle/internal/ops"
	"github.com/edbarbera/gitle/internal/ui"
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:     "switch <name>",
	Short:   "Switch to another line of work",
	Long:    "Moves you onto an existing branch (a separate line of work).\nSave your current work first so nothing gets left behind.",
	Example: "  gitle switch feature-login",
	Args:    cobra.ExactArgs(1),
	PreRunE: requireRepo,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if gitcmd.HasChanges() {
			ui.Warn("You have unsaved changes that would come with you.")
			ui.Hint("Consider saving first with %s.", ui.Bold(`gitle save "..."`))
		}

		err := ops.Switch(name)
		switch {
		case err == nil:
			ui.Success("Switched to %s.", ui.Bold(name))
			return nil

		case errors.Is(err, ops.ErrBranchMissing):
			ui.Error("There's no line of work called %q.", name)
			ui.Hint("See what exists with %s, or create it with %s.",
				ui.Bold("gitle branches"), ui.Bold("gitle new-branch "+name))
			return errSilent

		default:
			return err
		}
	},
}

func init() {
	rootCmd.AddCommand(switchCmd)
}
