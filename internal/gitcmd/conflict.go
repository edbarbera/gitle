package gitcmd

import (
	"os"
	"path/filepath"
	"strings"
)

// OpKind identifies which git operation left the repo mid-conflict.
type OpKind int

const (
	OpNone OpKind = iota
	OpMerge
	OpRebase
	OpCherryPick
)

// gitDir returns the repo's .git directory, or "" if it can't be found.
func gitDir() string {
	out, err := Capture("rev-parse", "--git-dir")
	if err != nil {
		return ""
	}
	return out
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// CurrentOp reports which operation (if any) is stopped mid-way with
// conflicts to resolve. Rebase and cherry-pick are checked before merge
// since a rebase can internally pause the same way a merge does.
func CurrentOp() OpKind {
	dir := gitDir()
	if dir == "" {
		return OpNone
	}
	switch {
	case exists(filepath.Join(dir, "rebase-merge")), exists(filepath.Join(dir, "rebase-apply")):
		return OpRebase
	case exists(filepath.Join(dir, "CHERRY_PICK_HEAD")):
		return OpCherryPick
	case exists(filepath.Join(dir, "MERGE_HEAD")):
		return OpMerge
	default:
		return OpNone
	}
}

// ConflictedFiles lists paths that still have unresolved conflict markers.
func ConflictedFiles() ([]string, error) {
	out, err := Capture("diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

// CheckoutOurs replaces path with the HEAD side of the conflict.
func CheckoutOurs(path string) error { return Run("checkout", "--ours", "--", path) }

// CheckoutTheirs replaces path with the incoming side of the conflict.
func CheckoutTheirs(path string) error { return Run("checkout", "--theirs", "--", path) }

// StageFile marks path as resolved.
func StageFile(path string) error { return Run("add", "--", path) }

// ContinueMerge finishes a merge once every conflict is resolved.
func ContinueMerge() (string, error) { return RunCaptureStderr("commit", "--no-edit") }

// ContinueRebase resumes a paused rebase once every conflict is resolved.
func ContinueRebase() (string, error) { return RunCaptureStderr("rebase", "--continue") }

// ContinueCherryPick resumes a paused cherry-pick once every conflict is resolved.
func ContinueCherryPick() (string, error) { return RunCaptureStderr("cherry-pick", "--continue") }

// AbortMerge, AbortRebase and AbortCherryPick undo an in-progress operation
// and put the working tree back exactly how it was before it started.
func AbortMerge() error      { return Run("merge", "--abort") }
func AbortRebase() error     { return Run("rebase", "--abort") }
func AbortCherryPick() error { return Run("cherry-pick", "--abort") }
