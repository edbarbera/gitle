package cmd

import (
	"strings"

	"github.com/edbarbera/gitle/internal/ops"
	"github.com/edbarbera/gitle/internal/ui"
	"github.com/spf13/cobra"
)

var branchesCmd = &cobra.Command{
	Use:     "branches",
	Short:   "List the separate lines of work",
	Long:    "Shows every branch — a separate line of work you can switch between. The one\nmarked with ❯ is where you are now.",
	Args:    cobra.NoArgs,
	PreRunE: requireRepo,
	RunE: func(cmd *cobra.Command, args []string) error {
		branches, err := ops.Branches()
		if err != nil {
			return err
		}
		if len(branches) == 0 {
			ui.Info("No lines of work yet — make your first save to start one.")
			return nil
		}

		var local, remote []ops.Branch
		for _, b := range branches {
			if b.Remote {
				remote = append(remote, b)
			} else {
				local = append(local, b)
			}
		}

		printBranchGroup("On this computer", local)
		printBranchGroup("Online", remote)

		ui.Hint("Switch with %s, or start a new one with %s.",
			ui.Bold("gitle switch <name>"), ui.Bold("gitle new-branch <name>"))
		return nil
	},
}

func printBranchGroup(title string, branches []ops.Branch) {
	if len(branches) == 0 {
		return
	}
	ui.Plain("%s", ui.Bold(title))
	for _, b := range branches {
		// Strip git's "remotes/" bookkeeping prefix: "origin/main" is the name
		// people actually recognise.
		name := strings.TrimPrefix(b.Name, "remotes/")
		if b.Current {
			ui.Plain("  %s %s", ui.Cyan("❯"), ui.Cyan(name))
			continue
		}
		ui.Plain("    %s", name)
	}
	ui.Blank()
}

func init() {
	rootCmd.AddCommand(branchesCmd)
}
