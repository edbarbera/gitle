package cmd

import (
	"strings"

	"github.com/edbarbera/gitle/internal/gitcmd"
	"github.com/edbarbera/gitle/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:     "status",
	Short:   "See what's going on right now",
	Long:    "A plain-English summary of where you are: which line of work you're on and\nwhat's changed since your last save, colour-coded.",
	Args:    cobra.NoArgs,
	PreRunE: requireRepo,
	RunE: func(cmd *cobra.Command, args []string) error {
		if branch := gitcmd.CurrentBranch(); branch != "" {
			ui.Info("You're on the %s line of work.", ui.Bold(branch))
		}

		if !gitcmd.HasCommits() {
			ui.Info("You haven't saved anything yet.")
			ui.Hint("Make your first save with %s.", ui.Bold(`gitle save "first version"`))
		}

		lines, err := gitcmd.StatusPorcelain()
		if err != nil {
			return err
		}

		if len(lines) == 0 {
			ui.Success("Everything is saved — nothing has changed.")
		} else {
			printChanges(lines)
			ui.Hint("Save these with %s.", ui.Bold(`gitle save "..."`))
		}

		if gitcmd.HasCommits() && !gitcmd.HasUpstream() {
			ui.Hint("This work isn't online yet. Put it online with %s.", ui.Bold("gitle send"))
		}
		return nil
	},
}

// printChanges groups porcelain output into friendly, colour-coded buckets.
func printChanges(lines []string) {
	var added, changed, removed []string
	for _, line := range lines {
		if len(line) < 4 {
			continue
		}
		x, y, path := line[0], line[1], strings.TrimSpace(line[3:])
		switch {
		case x == '?': // untracked
			added = append(added, path)
		case x == 'D' || y == 'D': // deleted
			removed = append(removed, path)
		case x == 'A' || y == 'A': // newly added
			added = append(added, path)
		default: // modified, renamed, copied, etc.
			changed = append(changed, path)
		}
	}

	ui.Warn("You have unsaved changes:")
	printGroup("New", added, ui.Green)
	printGroup("Changed", changed, ui.Yellow)
	printGroup("Removed", removed, ui.Red)
}

func printGroup(label string, files []string, color func(string) string) {
	if len(files) == 0 {
		return
	}
	colored := make([]string, len(files))
	for i, f := range files {
		colored[i] = color(f)
	}
	ui.Plain("  %-8s %s", label+":", strings.Join(colored, ", "))
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
