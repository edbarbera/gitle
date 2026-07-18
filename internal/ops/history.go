package ops

import (
	"strconv"
	"strings"

	"github.com/edbarbera/gitle/internal/gitcmd"
)

// SavedPoint is one entry in the project's history — what git calls a commit.
type SavedPoint struct {
	Hash    string // short hash
	Subject string // the description the user typed
	Author  string
	When    string // relative, e.g. "2 hours ago"
	Refs    string // branch/tag names pointing here, "" if none
}

// History returns saved points, newest first. limit of 0 means no limit.
func History(limit int) ([]SavedPoint, error) {
	if !gitcmd.HasCommits() {
		return nil, nil
	}

	// Unit separator between fields and record separator between commits:
	// neither can appear in a commit message, so the split is unambiguous
	// however people write their descriptions.
	const format = "--pretty=format:%h\x1f%s\x1f%an\x1f%cr\x1f%D\x1e"

	args := []string{"log", format}
	if limit > 0 {
		args = append(args, "-n", strconv.Itoa(limit))
	}
	out, err := gitcmd.Capture(args...)
	if err != nil {
		return nil, err
	}

	var commits []SavedPoint
	for _, record := range strings.Split(out, "\x1e") {
		record = strings.TrimLeft(record, "\n")
		if record == "" {
			continue
		}
		f := strings.Split(record, "\x1f")
		if len(f) < 5 {
			continue
		}
		commits = append(commits, SavedPoint{
			Hash:    f[0],
			Subject: f[1],
			Author:  f[2],
			When:    f[3],
			Refs:    f[4],
		})
	}
	return commits, nil
}

// CommitDiff returns the changes a saved point introduced.
func CommitDiff(hash string) (string, error) {
	return gitcmd.Capture("show", "--stat", "--patch", "--format=%s%n%n%an, %cr%n", hash)
}

// FileDiff returns the unsaved changes in one file. New files that git has
// never seen are shown as if they were added, so the preview isn't blank.
func FileDiff(path string) (string, error) {
	diff, err := gitcmd.Capture("diff", "HEAD", "--", path)
	if err == nil && strings.TrimSpace(diff) != "" {
		return diff, nil
	}
	// Untracked: git diff says nothing about a file it doesn't know, so ask
	// explicitly for it to be treated as new. --no-index exits non-zero
	// whenever it finds a difference, which here is always, so take the
	// output and ignore the status.
	out, _ := gitcmd.CaptureAllowFail("diff", "--no-index", "--", "/dev/null", path)
	return out, nil
}
