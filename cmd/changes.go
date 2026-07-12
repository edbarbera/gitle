package cmd

import (
	"fmt"
	"strings"

	"github.com/edbarbera/gitle/internal/ui"
)

// fileChange is one changed path with a friendly, colour-coded label.
type fileChange struct {
	path  string
	label string              // "New", "Changed", "Removed"
	color func(string) string // colour to render the path in
}

// parseChanges turns `git status --porcelain` lines into friendly categories.
func parseChanges(lines []string) []fileChange {
	var out []fileChange
	for _, line := range lines {
		if len(line) < 4 {
			continue
		}
		x, y := line[0], line[1]
		path := strings.TrimSpace(line[3:])
		// Renames show as "old -> new"; keep the new name for staging.
		if i := strings.Index(path, " -> "); i >= 0 {
			path = path[i+4:]
		}

		var fc fileChange
		fc.path = path
		switch {
		case x == '?': // untracked
			fc.label, fc.color = "New", ui.Green
		case x == 'D' || y == 'D': // deleted
			fc.label, fc.color = "Removed", ui.Red
		case x == 'A' || y == 'A': // added
			fc.label, fc.color = "New", ui.Green
		default: // modified, renamed, copied, type-changed
			fc.label, fc.color = "Changed", ui.Yellow
		}
		out = append(out, fc)
	}
	return out
}

// pickLabel renders a change as a padded, colour-coded checklist entry.
func (c fileChange) pickLabel() string {
	return fmt.Sprintf("%-8s %s", c.label+":", c.color(c.path))
}
