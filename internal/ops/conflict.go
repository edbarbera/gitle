package ops

import (
	"errors"
	"os"
	"strings"

	"github.com/edbarbera/gitle/internal/gitcmd"
)

// Side is which version of a clashing section to keep.
type Side int

const (
	// SideOurs keeps the version already here.
	SideOurs Side = iota
	// SideTheirs keeps the version that arrived.
	SideTheirs
	// SideBoth keeps them one after the other.
	SideBoth
)

// Hunk is one <<<<<<< ... ======= ... >>>>>>> section: the same few lines
// written two different ways.
type Hunk struct {
	Ours   []string
	Theirs []string
}

// segment is either a literal chunk of the file or a placeholder for one
// hunk, kept in original order so the file can be rebuilt.
type segment struct {
	text       string
	isConflict bool
	hunkIdx    int
}

// ConflictFile is a file with clashes, parsed into the parts that clash and
// the parts that don't.
type ConflictFile struct {
	Path  string
	Hunks []Hunk

	segments []segment
}

// ErrUnparsable means a file's conflict markers didn't make sense, so gitle
// won't risk rewriting it.
var ErrUnparsable = errors.New("couldn't make sense of the conflict markers")

// ConflictState is everything the resolver needs to know at startup.
type ConflictState struct {
	Op    gitcmd.OpKind
	Files []string
	// HeadLabel and OtherLabel name the two sides in plain English.
	HeadLabel  string
	OtherLabel string
}

// Conflicts reports the in-progress operation and what's left to resolve.
func Conflicts() (ConflictState, error) {
	op := gitcmd.CurrentOp()
	state := ConflictState{Op: op}
	state.HeadLabel, state.OtherLabel = SideLabels(op)
	if op == gitcmd.OpNone {
		return state, nil
	}
	files, err := gitcmd.ConflictedFiles()
	if err != nil {
		return state, err
	}
	state.Files = files
	return state, nil
}

// SideLabels names the two sides of a conflict in plain English.
//
// Which side is "yours" depends on the operation: in a merge or cherry-pick,
// HEAD is your current work; in a rebase your commits are the ones being
// replayed on top of HEAD, so the meaning flips.
func SideLabels(op gitcmd.OpKind) (head, other string) {
	switch op {
	case gitcmd.OpRebase:
		return "the version you grabbed", "your version"
	default:
		return "your version", "the version you grabbed"
	}
}

// LoadConflictFile reads a file and splits it into hunks.
func LoadConflictFile(path string) (*ConflictFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	hunks, segments, err := parseConflicts(string(data))
	if err != nil {
		return nil, err
	}
	return &ConflictFile{Path: path, Hunks: hunks, segments: segments}, nil
}

// Resolve writes the file back with one choice applied per hunk, then marks it
// resolved. choices must have one entry per hunk.
func (f *ConflictFile) Resolve(choices []Side) error {
	if len(choices) != len(f.Hunks) {
		return errors.New("a choice is needed for every clashing section")
	}
	resolved := make([]string, len(f.Hunks))
	for i, h := range f.Hunks {
		switch choices[i] {
		case SideOurs:
			resolved[i] = strings.Join(h.Ours, "\n")
		case SideTheirs:
			resolved[i] = strings.Join(h.Theirs, "\n")
		default:
			both := append(append([]string{}, h.Ours...), h.Theirs...)
			resolved[i] = strings.Join(both, "\n")
		}
	}
	if err := os.WriteFile(f.Path, []byte(rebuild(f.segments, resolved)), 0o644); err != nil {
		return err
	}
	return MarkResolved(f.Path)
}

// KeepWholeFile resolves a file by taking one side of it entirely.
func KeepWholeFile(path string, side Side) error {
	var err error
	switch side {
	case SideOurs:
		err = gitcmd.RunQuiet("checkout", "--ours", "--", path)
	case SideTheirs:
		err = gitcmd.RunQuiet("checkout", "--theirs", "--", path)
	default:
		return errors.New("a whole file has to come from one side or the other")
	}
	if err != nil {
		return err
	}
	return MarkResolved(path)
}

// MarkResolved tells git a file is sorted out.
func MarkResolved(path string) error {
	return gitcmd.RunQuiet("add", "--", path)
}

// StillConflicted reports whether a file the user edited by hand still has
// markers left in it.
func StillConflicted(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "<<<<<<< ")
}

// FinishOp completes the in-progress operation once every file is resolved.
func FinishOp(op gitcmd.OpKind) error {
	var args []string
	switch op {
	case gitcmd.OpMerge:
		args = []string{"commit", "--no-edit"}
	case gitcmd.OpRebase:
		args = []string{"rebase", "--continue"}
	case gitcmd.OpCherryPick:
		args = []string{"cherry-pick", "--continue"}
	default:
		return nil
	}
	// A rebase that continues into another conflicted commit needs the editor
	// kept out of the way; --no-edit above covers merge, and GIT_EDITOR=true
	// stops the others opening one over a drawn interface.
	stderr, err := gitcmd.RunQuietStderrEnv([]string{"GIT_EDITOR=true"}, args...)
	if err != nil {
		return errors.New(FirstLine(stderr))
	}
	return nil
}

// AbortOp undoes the in-progress operation, putting everything back exactly as
// it was before it started.
func AbortOp(op gitcmd.OpKind) error {
	switch op {
	case gitcmd.OpMerge:
		return gitcmd.RunQuiet("merge", "--abort")
	case gitcmd.OpRebase:
		return gitcmd.RunQuiet("rebase", "--abort")
	case gitcmd.OpCherryPick:
		return gitcmd.RunQuiet("cherry-pick", "--abort")
	default:
		return nil
	}
}

// parseConflicts splits file content into literal segments and conflict
// hunks, in order, so the file can be reconstructed once each hunk is
// resolved. It understands the optional diff3 "|||||||" base section but
// discards it — gitle only ever shows the two sides, never the base.
func parseConflicts(content string) ([]Hunk, []segment, error) {
	lines := strings.Split(content, "\n")
	var hunks []Hunk
	var segs []segment
	var cur []string

	i := 0
	for i < len(lines) {
		if !strings.HasPrefix(lines[i], "<<<<<<< ") {
			cur = append(cur, lines[i])
			i++
			continue
		}

		if len(cur) > 0 {
			segs = append(segs, segment{text: strings.Join(cur, "\n")})
			cur = nil
		}
		i++

		var ours []string
		for i < len(lines) && !strings.HasPrefix(lines[i], "=======") && !strings.HasPrefix(lines[i], "||||||| ") {
			ours = append(ours, lines[i])
			i++
		}
		if i < len(lines) && strings.HasPrefix(lines[i], "||||||| ") {
			for i < len(lines) && !strings.HasPrefix(lines[i], "=======") {
				i++
			}
		}
		if i >= len(lines) {
			return nil, nil, ErrUnparsable
		}
		i++ // skip =======

		var theirs []string
		for i < len(lines) && !strings.HasPrefix(lines[i], ">>>>>>> ") {
			theirs = append(theirs, lines[i])
			i++
		}
		if i >= len(lines) {
			return nil, nil, ErrUnparsable
		}
		i++ // skip >>>>>>> line

		hunks = append(hunks, Hunk{Ours: ours, Theirs: theirs})
		segs = append(segs, segment{isConflict: true, hunkIdx: len(hunks) - 1})
	}
	if len(cur) > 0 {
		segs = append(segs, segment{text: strings.Join(cur, "\n")})
	}
	return hunks, segs, nil
}

// rebuild reassembles a file's content from its segments, substituting each
// conflict placeholder with its resolved text.
func rebuild(segs []segment, resolved []string) string {
	var b strings.Builder
	for idx, s := range segs {
		if s.isConflict {
			b.WriteString(resolved[s.hunkIdx])
		} else {
			b.WriteString(s.text)
		}
		if idx < len(segs)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}
