package cmd

import (
	"github.com/edbarbera/gitle/internal/gitcmd"
	"github.com/edbarbera/gitle/internal/ops"
	"github.com/edbarbera/gitle/internal/tui"
	"github.com/edbarbera/gitle/internal/ui"
	"github.com/spf13/cobra"
)

var fixConflictsCmd = &cobra.Command{
	Use:   "fix-conflicts",
	Short: "Walk through conflicts step by step",
	Long: `Walks you through any clashes left by gitle grab (or a merge), one section at
a time, showing both versions side by side so you can pick what to keep — no
need to touch the raw markers yourself.`,
	Args:    cobra.NoArgs,
	PreRunE: requireRepo,
	RunE: func(cmd *cobra.Command, args []string) error {
		state, err := ops.Conflicts()
		if err != nil {
			return err
		}
		if state.Op == gitcmd.OpNone {
			ui.Info("No conflicts right now — nothing to fix.")
			ui.Hint("If %s ever says something clashed, run this again.", ui.Bold("gitle grab"))
			return nil
		}

		if len(state.Files) == 0 {
			// Mid-operation, but everything is already resolved: just finish.
			if err := ops.FinishOp(state.Op); err != nil {
				ui.Error("Couldn't finish automatically.")
				ui.Hint("git said: %s", err)
				return errSilent
			}
			ui.Success("All conflicts resolved!")
			return nil
		}

		if !ui.Interactive() {
			ui.Error("This needs a terminal so it can ask you what to keep.")
			ui.Hint("Run %s in a normal terminal window.", ui.Bold("gitle fix-conflicts"))
			return errSilent
		}

		if err := tui.RunConflicts(state); err != nil {
			return err
		}

		// The resolver draws its own closing state; report where things stand
		// now that the screen has been handed back.
		return reportConflictOutcome()
	},
}

// reportConflictOutcome prints a plain-English summary once the resolver has
// exited, reading the repo rather than trusting what happened on screen.
func reportConflictOutcome() error {
	state, err := ops.Conflicts()
	if err != nil {
		return err
	}
	switch {
	case state.Op == gitcmd.OpNone:
		ui.Success("All conflicts resolved!")
		if gitcmd.HasUpstream() {
			ui.Hint("Send it online with %s.", ui.Bold("gitle send"))
		}
	case len(state.Files) > 0:
		ui.Info("Some files still need attention — run %s again when ready.", ui.Bold("gitle fix-conflicts"))
	default:
		ui.Info("Everything is resolved, but the %s hasn't been finished yet.", opName(state.Op))
		ui.Hint("Run %s again to wrap it up.", ui.Bold("gitle fix-conflicts"))
	}
	return nil
}

// opName names an in-progress operation in words a beginner can follow.
func opName(op gitcmd.OpKind) string {
	switch op {
	case gitcmd.OpRebase:
		return "update"
	case gitcmd.OpCherryPick:
		return "copied save"
	default:
		return "merge"
	}
}

func init() {
	rootCmd.AddCommand(fixConflictsCmd)
}
