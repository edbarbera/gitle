package cmd

import (
	"fmt"
	"strings"

	"github.com/edbarbera/gitle/internal/ops"
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
		s, err := ops.CurrentStatus()
		if err != nil {
			return err
		}
		printStatus(s)
		return nil
	},
}

// printStatus renders a status snapshot as gitle's plain-English report.
func printStatus(s ops.Status) {
	// --- Header: project + branch + position vs main ---
	if s.Name != "" {
		ui.Plain("📦 %s", ui.Bold(s.Name))
	}
	if s.Branch != "" {
		ui.Plain("   %s %s", ui.Dim("on branch"), ui.Cyan(s.Branch))
	}
	if s.VsMain != nil {
		ui.Plain("   %s %s", ui.Dim("compared to "+s.MainBranch+":"), describeAheadBehind(*s.VsMain))
	}

	// --- Changes since last save ---
	ui.Blank()
	if !s.HasCommits {
		ui.Info("You haven't saved anything yet.")
		ui.Hint("Make your first save with %s.", ui.Bold(`gitle save "first version"`))
	} else if len(s.Changes) == 0 {
		ui.Success("Everything is saved — nothing has changed.")
	}
	if len(s.Changes) > 0 {
		printChanges(s.Changes)
		ui.Hint("Save these with %s.", ui.Bold(`gitle save "..."`))
	}

	// --- How you compare to the online copy ---
	if s.HasCommits {
		printOnlineStatus(s)
	}
}

// printOnlineStatus reports whether the branch is in sync with, ahead of, or
// behind its online copy.
func printOnlineStatus(s ops.Status) {
	if !s.HasUpstream {
		ui.Hint("Not online yet — share it with %s.", ui.Bold("gitle send"))
		return
	}
	if s.VsUpstream == nil {
		return
	}
	ahead, behind := s.VsUpstream.Ahead, s.VsUpstream.Behind
	switch {
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
func describeAheadBehind(ab ops.AheadBehind) string {
	switch {
	case ab.InSync():
		return ui.Dim("in sync")
	case ab.Ahead > 0 && ab.Behind > 0:
		return ui.Green(fmt.Sprintf("%d ahead", ab.Ahead)) + ui.Dim(", ") + ui.Yellow(fmt.Sprintf("%d behind", ab.Behind))
	case ab.Ahead > 0:
		return ui.Green(fmt.Sprintf("%d ahead", ab.Ahead))
	default:
		return ui.Yellow(fmt.Sprintf("%d behind", ab.Behind))
	}
}

// saves renders a count with the right singular/plural noun.
func saves(n int) string {
	if n == 1 {
		return "1 saved point"
	}
	return fmt.Sprintf("%d saved points", n)
}

// printChanges groups changes into friendly, colour-coded buckets.
func printChanges(changes []ops.Change) {
	var added, changed, removed []string
	for _, c := range changes {
		switch c.Kind {
		case ops.ChangeNew:
			added = append(added, c.Path)
		case ops.ChangeRemoved:
			removed = append(removed, c.Path)
		default:
			changed = append(changed, c.Path)
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
