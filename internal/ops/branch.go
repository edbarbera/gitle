package ops

import (
	"errors"
	"strings"

	"github.com/edbarbera/gitle/internal/gitcmd"
)

var (
	// ErrBranchMissing means the named line of work doesn't exist.
	ErrBranchMissing = errors.New("no line of work by that name")
	// ErrBranchExists means the name is already taken.
	ErrBranchExists = errors.New("a line of work by that name already exists")
)

// Branch is one line of work.
type Branch struct {
	Name    string
	Current bool
	Remote  bool // lives online rather than on this machine
}

// Branches lists every line of work, local first, current one marked.
func Branches() ([]Branch, error) {
	// The custom format avoids parsing git's decorated `branch -a` output,
	// which changes shape depending on column width and the current branch.
	out, err := gitcmd.Capture("branch", "-a", "--format=%(refname:short)%09%(HEAD)")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}

	var branches []Branch
	for _, line := range strings.Split(out, "\n") {
		name, head, _ := strings.Cut(line, "\t")
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		// Skip the symbolic "origin/HEAD -> origin/main" entry: it's a pointer
		// to another branch in this same list, not a line of work of its own.
		if strings.Contains(name, "->") {
			continue
		}
		branches = append(branches, Branch{
			Name:    name,
			Current: strings.TrimSpace(head) == "*",
			Remote:  strings.HasPrefix(name, "remotes/"),
		})
	}
	return branches, nil
}

// Switch moves onto an existing line of work.
//
// Unsaved changes are deliberately not blocked here: git carries them across
// when it can, and stopping the user would be more annoying than helpful. The
// caller is expected to warn.
func Switch(name string) error {
	if !gitcmd.BranchExists(name) {
		return ErrBranchMissing
	}
	return gitcmd.RunQuiet("checkout", name)
}

// NewBranch creates a fresh line of work and moves onto it.
func NewBranch(name string) error {
	if gitcmd.BranchExists(name) {
		return ErrBranchExists
	}
	return gitcmd.RunQuiet("checkout", "-b", name)
}
