package cmd

import (
	"errors"

	"github.com/edbarbera/gitle/internal/ops"
	"github.com/edbarbera/gitle/internal/ui"
	"github.com/spf13/cobra"
)

var undoHardFlag bool

var undoCmd = &cobra.Command{
	Use:   "undo",
	Short: "Undo your last save",
	Long: `Removes your most recent saved point but keeps all the file changes it
contained, so nothing is lost. You can save again once you've fixed things up.

Use --hard to instead throw away your current uncommitted changes and go back
to your last save. That one cannot be undone, so gitle always asks first.`,
	Args:    cobra.NoArgs,
	PreRunE: requireRepo,
	RunE: func(cmd *cobra.Command, args []string) error {
		if undoHardFlag {
			return runUndoHard()
		}

		last, err := ops.LastSaveMessage()
		if errors.Is(err, ops.ErrNothingToUndo) {
			ui.Info("There's nothing to undo yet — you haven't saved anything.")
			return nil
		}
		if err != nil {
			return err
		}

		ui.Warn("This will undo your last save: %q", last)
		ui.Hint("Your file changes are kept — only the saved point is removed.")
		if !ui.Confirm("Undo it?") {
			ui.Info("Left everything as it was.")
			return nil
		}

		if err := ops.UndoLastSave(); err != nil {
			return err
		}
		ui.Success("Undid your last save. Your changes are still here.")
		ui.Hint("Save again with %s when you're ready.", ui.Bold(`gitle save "..."`))
		return nil
	},
}

// runUndoHard throws away all uncommitted changes after a clear warning and an
// explicit confirmation. This is destructive and cannot be reversed.
func runUndoHard() error {
	changes, err := ops.Changes()
	if err != nil {
		return err
	}
	if len(changes) == 0 {
		ui.Info("Nothing to discard — you have no uncommitted changes.")
		return nil
	}

	ui.Warn("This will permanently discard uncommitted changes.")
	ui.Plain("    Files affected:")
	for _, c := range changes {
		ui.Plain("      %s %s", c.Kind.Label()+":", changeColor(c.Kind)(c.Path))
	}
	ui.Warn("This cannot be undone.")

	if !ui.Interactive() {
		ui.Error("Refusing to discard without a confirmation.")
		ui.Hint("Run %s in a terminal so it can ask you first.", ui.Bold("gitle undo --hard"))
		return errSilent
	}
	if !ui.Confirm("Are you sure?") {
		ui.Info("Phew — nothing was discarded.")
		return nil
	}

	if err := ops.Discard(); err != nil {
		return err
	}
	ui.Success("Discarded all uncommitted changes. This folder now matches your last save.")
	return nil
}

func init() {
	undoCmd.Flags().BoolVar(&undoHardFlag, "hard", false, "throw away uncommitted changes instead of undoing the last save")
	rootCmd.AddCommand(undoCmd)
}
