package ops

import (
	"errors"

	"github.com/edbarbera/gitle/internal/gitcmd"
)

// ErrNoMessage is returned when a save is attempted without a description.
var ErrNoMessage = errors.New("a save needs a short description")

// SaveResult describes a completed save and what the user might sensibly do
// next, so the caller can offer the right follow-up without asking git again.
type SaveResult struct {
	Paths       []string
	Message     string
	Leftover    bool // changes remain unsaved in the working tree
	HasUpstream bool // an online copy exists to send to
}

// Stage makes the index hold exactly the given paths, covering new, changed
// and removed files alike.
//
// It's separate from Commit because `gitle save` stages before asking for a
// description: an AI-drafted message needs a real staged diff to read.
//
// The index is cleared first so it ends up holding the picked paths and
// nothing else. gitle treats the index as its own working space — the people
// it's built for never stage anything by hand — and clearing it is what lets
// Commit record the index directly, which in turn is the only way to save a
// deleted file reliably. Clearing the index changes no file on disk.
func Stage(paths []string) error {
	if err := clearIndex(); err != nil {
		return err
	}
	if len(paths) == 0 {
		return nil
	}
	return gitcmd.RunQuiet(append([]string{"add", "-A", "--"}, paths...)...)
}

// clearIndex unstages everything without touching the working tree.
func clearIndex() error {
	if !gitcmd.HasCommits() {
		// Before the first save there's no HEAD to reset against, so empty the
		// index directly. It's a no-op when nothing is staged yet.
		_ = gitcmd.RunQuiet("rm", "-r", "--cached", "-q", "--ignore-unmatch", ".")
		return nil
	}
	return gitcmd.RunQuiet("reset", "-q")
}

// StageAll stages every change in the working tree.
func StageAll() error {
	if err := clearIndex(); err != nil {
		return err
	}
	return gitcmd.RunQuiet("add", "-A")
}

// Commit records whatever Stage put in the index as a saved point. paths is
// only used to describe the result.
//
// It deliberately does not pass the paths to git as a pathspec. That form
// makes git re-read each named file from the working tree and ignore the
// index, so a file the user deleted — which by definition isn't in the
// working tree — fails with "pathspec did not match any files" and the save
// can never complete. Committing the index instead handles deletions the same
// way as any other change.
func Commit(message string, paths []string) (SaveResult, error) {
	if message == "" {
		return SaveResult{}, ErrNoMessage
	}
	if err := gitcmd.RunQuiet("commit", "-m", message); err != nil {
		return SaveResult{}, err
	}
	return SaveResult{
		Paths:       paths,
		Message:     message,
		Leftover:    gitcmd.HasChanges(),
		HasUpstream: gitcmd.HasUpstream(),
	}, nil
}

// Save stages and commits in one step, for callers that already know both the
// paths and the message.
func Save(message string, paths []string) (SaveResult, error) {
	if message == "" {
		return SaveResult{}, ErrNoMessage
	}
	if err := Stage(paths); err != nil {
		return SaveResult{}, err
	}
	return Commit(message, paths)
}

// SaveAll stages and commits everything in the working tree. Used for the very
// first save, where there's nothing to pick between.
func SaveAll(message string) (SaveResult, error) {
	if message == "" {
		return SaveResult{}, ErrNoMessage
	}
	changes, err := Changes()
	if err != nil {
		return SaveResult{}, err
	}
	if err := StageAll(); err != nil {
		return SaveResult{}, err
	}
	if err := gitcmd.RunQuiet("commit", "-m", message); err != nil {
		return SaveResult{}, err
	}
	return SaveResult{
		Paths:       Paths(changes),
		Message:     message,
		Leftover:    gitcmd.HasChanges(),
		HasUpstream: gitcmd.HasUpstream(),
	}, nil
}
