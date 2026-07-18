// Package ops holds what gitle's commands actually do, separated from how the
// result is shown.
//
// Every function here takes its inputs explicitly and returns a value: nothing
// in this package prints, prompts, or writes to the terminal. That's what lets
// the same logic serve both the line-at-a-time CLI, which renders results as
// text, and the full-screen dashboard, which renders them as a frame — and it
// means none of it can corrupt a UI that's drawing over the terminal.
//
// The one hard rule: never call gitcmd.Run from here. It streams git's output
// straight to stdout. Use gitcmd.RunQuiet instead.
package ops

import (
	"strings"

	"github.com/edbarbera/gitle/internal/gitcmd"
)

// ChangeKind is how a file differs from the last save.
type ChangeKind int

const (
	// ChangeNew is a file git hasn't seen before, or one newly added.
	ChangeNew ChangeKind = iota
	// ChangeModified is an existing file with different contents.
	ChangeModified
	// ChangeRemoved is a file that has been deleted.
	ChangeRemoved
)

// Label is the plain-English name gitle shows for this kind of change.
func (k ChangeKind) Label() string {
	switch k {
	case ChangeNew:
		return "New"
	case ChangeRemoved:
		return "Removed"
	default:
		return "Changed"
	}
}

// Change is one path that differs from the last save.
type Change struct {
	Path string
	Kind ChangeKind
}

// Changes reports every uncommitted change in the working tree.
func Changes() ([]Change, error) {
	lines, err := gitcmd.StatusPorcelain()
	if err != nil {
		return nil, err
	}
	return parseChanges(lines), nil
}

// parseChanges turns `git status --porcelain` lines into friendly categories.
// The first two columns are the staged and unstaged status codes; the path
// starts at column 4.
func parseChanges(lines []string) []Change {
	var out []Change
	for _, line := range lines {
		if len(line) < 4 {
			continue
		}
		x, y := line[0], line[1]
		path := strings.TrimSpace(line[3:])
		// Renames show as "old -> new"; keep the new name, since that's the
		// path any later `git add` has to name.
		if i := strings.Index(path, " -> "); i >= 0 {
			path = path[i+4:]
		}

		var kind ChangeKind
		switch {
		case x == '?': // untracked
			kind = ChangeNew
		case x == 'D' || y == 'D': // deleted
			kind = ChangeRemoved
		case x == 'A' || y == 'A': // added
			kind = ChangeNew
		default: // modified, renamed, copied, type-changed
			kind = ChangeModified
		}
		out = append(out, Change{Path: path, Kind: kind})
	}
	return out
}

// Paths pulls the bare path list out of a set of changes.
func Paths(changes []Change) []string {
	out := make([]string, len(changes))
	for i, c := range changes {
		out[i] = c.Path
	}
	return out
}
