package ops

import (
	"errors"

	"github.com/edbarbera/gitle/internal/gitcmd"
)

// ErrNothingToUndo means there are no saved points to remove.
var ErrNothingToUndo = errors.New("nothing saved yet")

// LastSaveMessage returns the description of the most recent saved point, so
// the caller can name it when asking for confirmation.
func LastSaveMessage() (string, error) {
	if !gitcmd.HasCommits() {
		return "", ErrNothingToUndo
	}
	return gitcmd.Capture("log", "-1", "--pretty=%s")
}

// UndoLastSave removes the most recent saved point while keeping every file
// change it contained. Nothing is lost: the changes go back to being unsaved
// work, ready to be saved again.
func UndoLastSave() error {
	if !gitcmd.HasCommits() {
		return ErrNothingToUndo
	}
	if _, err := gitcmd.Capture("rev-parse", "HEAD~1"); err != nil {
		// No parent: this is the very first save. Remove the pointer so the
		// repo goes back to "nothing saved yet"; the files stay untouched.
		return gitcmd.RunQuiet("update-ref", "-d", "HEAD")
	}
	// --soft keeps every change in the working tree; only the save disappears.
	return gitcmd.RunQuiet("reset", "--soft", "HEAD~1")
}

// Discard throws away all uncommitted changes, returning the folder to its
// last saved state.
//
// This destroys work irreversibly. It does not ask — callers must confirm with
// the user before calling it.
func Discard() error {
	// Revert tracked files to the last save...
	if gitcmd.HasCommits() {
		if err := gitcmd.RunQuiet("reset", "--hard", "HEAD"); err != nil {
			return err
		}
	} else {
		if err := gitcmd.RunQuiet("reset", "-q"); err != nil {
			return err
		}
	}
	// ...then remove files git was never tracking in the first place.
	return gitcmd.RunQuiet("clean", "-fd")
}
