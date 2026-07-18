package cmd

import (
	"fmt"

	"github.com/edbarbera/gitle/internal/ops"
	"github.com/edbarbera/gitle/internal/ui"
)

// changeColor picks the colour a kind of change is shown in: green for
// additions, yellow for edits, red for deletions.
func changeColor(k ops.ChangeKind) func(string) string {
	switch k {
	case ops.ChangeNew:
		return ui.Green
	case ops.ChangeRemoved:
		return ui.Red
	default:
		return ui.Yellow
	}
}

// pickLabel renders a change as a padded, colour-coded checklist entry.
func pickLabel(c ops.Change) string {
	return fmt.Sprintf("%-8s %s", c.Kind.Label()+":", changeColor(c.Kind)(c.Path))
}

// pickLabels renders a whole set of changes for a checklist.
func pickLabels(changes []ops.Change) []string {
	out := make([]string, len(changes))
	for i, c := range changes {
		out[i] = pickLabel(c)
	}
	return out
}
