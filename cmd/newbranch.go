package cmd

import (
	"errors"

	"github.com/edbarbera/gitle/internal/ops"
	"github.com/edbarbera/gitle/internal/ui"
	"github.com/spf13/cobra"
)

var newBranchCmd = &cobra.Command{
	Use:     "new-branch <name>",
	Short:   "Start a new line of work",
	Long:    "Creates a fresh branch — a separate line of work — and moves you onto it,\nso you can try things without touching the main work.",
	Example: "  gitle new-branch feature-login",
	Args:    cobra.ExactArgs(1),
	PreRunE: requireRepo,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		err := ops.NewBranch(name)
		switch {
		case err == nil:
			ui.Success("Created and switched to %s.", ui.Bold(name))
			ui.Hint("Save work here with %s; it stays separate until you merge.", ui.Bold(`gitle save "..."`))
			return nil

		case errors.Is(err, ops.ErrBranchExists):
			ui.Error("A line of work called %q already exists.", name)
			ui.Hint("Jump onto it with %s instead.", ui.Bold("gitle switch "+name))
			return errSilent

		default:
			return err
		}
	},
}

func init() {
	rootCmd.AddCommand(newBranchCmd)
}
