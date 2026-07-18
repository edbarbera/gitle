package cmd

import (
	"github.com/edbarbera/gitle/internal/ops"
	"github.com/edbarbera/gitle/internal/ui"
)

// reviewRisks warns about secrets and oversized files among the paths about to
// be saved, then asks the user to confirm. Returns true when it's safe (or
// confirmed) to proceed.
func reviewRisks(paths []string) bool {
	risks := ops.ScanRisks(paths)
	if !risks.Any() {
		return true
	}

	if len(risks.Secrets) > 0 {
		ui.Warn("These look like private/secret files:")
		for _, p := range risks.Secrets {
			ui.Hint("  • %s", p)
		}
		ui.Hint("Saving them can leak passwords or keys — especially once you send online.")
	}
	if len(risks.Large) > 0 {
		ui.Warn("These files are large and may bloat your project:")
		for _, f := range risks.Large {
			ui.Hint("  • %s (%s)", f.Path, f.Size)
		}
	}

	if !ui.Interactive() {
		ui.Warn("Saving anyway — no terminal here to ask.")
		return true
	}
	return ui.ConfirmDefault("Save these anyway?", false)
}
