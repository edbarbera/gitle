package cmd

import (
	"fmt"
	"strings"

	"github.com/edbarbera/gitle/internal/gitcmd"
	"github.com/edbarbera/gitle/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:     "status",
	Short:   "See what's going on right now",
	Long:    "A plain-English summary: which project and line of work you're on, what's\nchanged since your last save, and how you compare to everyone else's work.",
	Args:    cobra.NoArgs,
	PreRunE: requireRepo,
	RunE: func(cmd *cobra.Command, args []string) error {
		branch := gitcmd.CurrentBranch()
		hasCommits := gitcmd.HasCommits()

		// --- Header: project + branch + position vs main ---
		if name := gitcmd.RepoName(); name != "" {
			fmt.Printf("📦 %s\n", ui.Bold(name))
		}
		if branch != "" {
			fmt.Printf("   %s %s\n", ui.Dim("on branch"), ui.Cyan(branch))
		}
		if hasCommits && branch != "" {
			if mainB := gitcmd.MainBranch(); mainB != "" && branch != mainB {
				if ahead, behind, ok := gitcmd.AheadBehind(mainB); ok {
					fmt.Printf("   %s %s\n", ui.Dim("compared to "+mainB+":"), describeAheadBehind(ahead, behind))
				}
			}
		}

		// --- Changes since last save ---
		lines, err := gitcmd.StatusPorcelain()
		if err != nil {
			return err
		}
		fmt.Println()
		if !hasCommits {
			ui.Info("You haven't saved anything yet.")
			ui.Hint("Make your first save with %s.", ui.Bold(`gitle save "first version"`))
		} else if len(lines) == 0 {
			ui.Success("Everything is saved — nothing has changed.")
		}
		if len(lines) > 0 {
			printChanges(lines)
			ui.Hint("Save these with %s.", ui.Bold(`gitle save "..."`))
		}

		// --- How you compare to the online copy ---
		if hasCommits {
			printOnlineStatus()
		}
		return nil
	},
}

// printOnlineStatus reports whether the branch is in sync with, ahead of, or
// behind its online copy.
func printOnlineStatus() {
	if !gitcmd.HasUpstream() {
		ui.Hint("Not online yet — share it with %s.", ui.Bold("gitle send"))
		return
	}
	ahead, behind, ok := gitcmd.AheadBehind("@{upstream}")
	switch {
	case !ok:
		return
	case ahead > 0 && behind > 0:
		ui.Warn("%d to send and %d to grab. Run %s, then %s.",
			ahead, behind, ui.Bold("gitle grab"), ui.Bold("gitle send"))
	case ahead > 0:
		ui.Info("%s ready to send with %s.", saves(ahead), ui.Bold("gitle send"))
	case behind > 0:
		ui.Info("%s waiting online — get them with %s.", saves(behind), ui.Bold("gitle grab"))
	default:
		ui.Success("Up to date with online.")
	}
}

// describeAheadBehind renders an ahead/behind pair in colour.
func describeAheadBehind(ahead, behind int) string {
	switch {
	case ahead == 0 && behind == 0:
		return ui.Dim("in sync")
	case ahead > 0 && behind > 0:
		return ui.Green(fmt.Sprintf("%d ahead", ahead)) + ui.Dim(", ") + ui.Yellow(fmt.Sprintf("%d behind", behind))
	case ahead > 0:
		return ui.Green(fmt.Sprintf("%d ahead", ahead))
	default:
		return ui.Yellow(fmt.Sprintf("%d behind", behind))
	}
}

// saves renders a count with the right singular/plural noun.
func saves(n int) string {
	if n == 1 {
		return "1 saved point"
	}
	return fmt.Sprintf("%d saved points", n)
}

// printChanges groups porcelain output into friendly, colour-coded buckets.
func printChanges(lines []string) {
	var added, changed, removed []string
	for _, c := range parseChanges(lines) {
		switch c.label {
		case "New":
			added = append(added, c.path)
		case "Removed":
			removed = append(removed, c.path)
		default:
			changed = append(changed, c.path)
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
