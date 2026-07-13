package cmd

import (
	"github.com/edbarbera/gitle/internal/gitcmd"
	"github.com/edbarbera/gitle/internal/ui"
	"github.com/spf13/cobra"
)

var saveAll bool

var saveCmd = &cobra.Command{
	Use:   "save [\"what you changed\"]",
	Short: "Save a snapshot of your work",
	Long: `Records your changes as a saved point you can always come back to.

In a terminal, gitle first shows a checklist of what changed so you can pick
exactly which files to include, then asks for a description. Use --all to skip
the checklist and save everything. Git calls the result a commit.`,
	Example: `  gitle save
  gitle save "fixed the login bug"
  gitle save --all "quick save of everything"`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: requireRepo,
	RunE: func(cmd *cobra.Command, args []string) error {
		message := ""
		if len(args) == 1 {
			message = args[0]
		}

		lines, err := gitcmd.StatusPorcelain()
		if err != nil {
			return err
		}
		if len(lines) == 0 {
			ui.Info("Nothing to save — your work is already up to date.")
			return nil
		}
		changes := parseChanges(lines)

		var paths []string
		if saveAll {
			// Skip the checklist: include every change.
			for _, c := range changes {
				paths = append(paths, c.path)
			}
		} else {
			// Let the user pick which files to include (all ticked by default).
			// Without a terminal this returns everything, preserving "save all".
			labels := make([]string, len(changes))
			for i, c := range changes {
				labels[i] = c.pickLabel()
			}
			picked := ui.Pick("Which changes do you want to save?", labels)
			if len(picked) == 0 {
				ui.Info("Nothing selected — nothing was saved.")
				return nil
			}
			paths = make([]string, len(picked))
			for i, idx := range picked {
				paths[i] = changes[idx].path
			}
		}

		// Safety rail: flag secrets / oversized files before they're committed.
		if !reviewRisks(paths) {
			ui.Info("Nothing was saved.")
			return nil
		}

		// Ask for a description now, after picking, if none was given.
		if message == "" {
			if !ui.IsInteractive() {
				ui.Error("Please describe what you changed.")
				ui.Hint("Example: %s", ui.Bold(`gitle save "fixed the login bug"`))
				return errSilent
			}
			message = ui.Ask("Describe what you changed:", "")
			if message == "" {
				ui.Error("A short description is needed to save.")
				return errSilent
			}
		}

		// Stage exactly the picked paths (covers new, changed and removed),
		// then commit only those paths so nothing unpicked sneaks in.
		addArgs := append([]string{"add", "-A", "--"}, paths...)
		if err := gitcmd.Run(addArgs...); err != nil {
			return err
		}
		commitArgs := append([]string{"commit", "-m", message, "--"}, paths...)
		if err := gitcmd.Run(commitArgs...); err != nil {
			return err
		}

		ui.Success("Saved %d file(s): %q", len(paths), message)
		if gitcmd.HasChanges() {
			ui.Hint("Some changes were left unsaved — run %s again when ready.", ui.Bold("gitle save"))
		}
		if gitcmd.HasUpstream() {
			ui.Hint("Send it online with %s.", ui.Bold("gitle send"))
		} else {
			ui.Hint("Share it online with %s once you're ready.", ui.Bold("gitle send"))
		}
		return nil
	},
}

func init() {
	saveCmd.Flags().BoolVarP(&saveAll, "all", "a", false, "save every change without showing the checklist")
	rootCmd.AddCommand(saveCmd)
}
