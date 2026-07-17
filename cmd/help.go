package cmd

import (
	"fmt"

	"github.com/edbarbera/gitle/internal/ui"
	"github.com/spf13/cobra"
)

type helpEntry struct{ name, desc string }

type helpGroup struct {
	title   string
	entries []helpEntry
}

// helpGroups curates the commands into friendly, task-based sections rather
// than one long alphabetical list.
var helpGroups = []helpGroup{
	{"Getting started", []helpEntry{
		{"start", "Set up this folder, guided step by step"},
	}},
	{"Everyday", []helpEntry{
		{"save", "Save a snapshot of your work"},
		{"send", "Send your saved work online"},
		{"grab", "Grab everyone's latest work"},
		{"status", "See what's going on right now"},
		{"history", "See your saved points over time"},
	}},
	{"Lines of work (branches)", []helpEntry{
		{"branches", "List your separate lines of work"},
		{"switch", "Switch to another line of work"},
		{"new-branch", "Start a new line of work"},
	}},
	{"Fixing mistakes", []helpEntry{
		{"undo", "Undo your last save  (--hard discards changes)"},
		{"fix-conflicts", "Walk through conflicts step by step"},
	}},
}

// renderHelp prints the aesthetic, grouped command overview.
func renderHelp() {
	ui.Banner()
	for _, g := range helpGroups {
		fmt.Println(ui.Bold(g.title))
		for _, e := range g.entries {
			fmt.Printf("  %s  %s\n", ui.Cyan(fmt.Sprintf("%-14s", e.name)), ui.Dim(e.desc))
		}
		fmt.Println()
	}
	fmt.Println(ui.Dim("Run 'gitle <command> --help' for more on any command."))
	fmt.Println(ui.Dim("Check your version with 'gitle --version'."))
}

func init() {
	// Use our grouped overview for the top-level help, but keep cobra's default
	// per-command help for `gitle <command> --help`.
	defaultHelp := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		if c == rootCmd {
			renderHelp()
			return
		}
		defaultHelp(c, args)
	})
}
