package ops

import (
	"github.com/edbarbera/gitle/internal/gitcmd"
)

// AheadBehind is how far one line of work has diverged from another.
type AheadBehind struct {
	Ahead  int
	Behind int
}

// InSync reports whether the two sides are level.
func (ab AheadBehind) InSync() bool { return ab.Ahead == 0 && ab.Behind == 0 }

// Status is a complete snapshot of where the repo stands: everything the
// `status` command prints and everything the dashboard header shows, gathered
// in one pass so the two can never disagree.
type Status struct {
	Name        string // repository folder name
	Branch      string // current branch, "" when detached or empty
	MainBranch  string // "main" or "master", "" if neither exists
	HasCommits  bool
	HasRemote   bool
	HasUpstream bool

	Changes []Change

	// VsMain and VsUpstream are nil when the comparison doesn't apply — no
	// main branch, no upstream, or nothing saved yet to compare.
	VsMain     *AheadBehind
	VsUpstream *AheadBehind

	// Op is the merge/rebase/cherry-pick in progress, if any.
	Op gitcmd.OpKind
}

// Conflicted reports whether the repo is mid-operation with work to resolve.
func (s Status) Conflicted() bool { return s.Op != gitcmd.OpNone }

// CurrentStatus gathers the repo's state. Individual lookups that fail are
// reported as "unknown" rather than fatal — a status report should still show
// what it can rather than refuse to render.
func CurrentStatus() (Status, error) {
	s := Status{
		Name:        gitcmd.RepoName(),
		Branch:      gitcmd.CurrentBranch(),
		MainBranch:  gitcmd.MainBranch(),
		HasCommits:  gitcmd.HasCommits(),
		HasRemote:   gitcmd.HasRemote(),
		HasUpstream: gitcmd.HasUpstream(),
		Op:          gitcmd.CurrentOp(),
	}

	changes, err := Changes()
	if err != nil {
		return s, err
	}
	s.Changes = changes

	if s.HasCommits && s.Branch != "" && s.MainBranch != "" && s.Branch != s.MainBranch {
		if ahead, behind, ok := gitcmd.AheadBehind(s.MainBranch); ok {
			s.VsMain = &AheadBehind{Ahead: ahead, Behind: behind}
		}
	}
	if s.HasCommits && s.HasUpstream {
		if ahead, behind, ok := gitcmd.AheadBehind("@{upstream}"); ok {
			s.VsUpstream = &AheadBehind{Ahead: ahead, Behind: behind}
		}
	}
	return s, nil
}
