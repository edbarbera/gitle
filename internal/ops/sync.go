package ops

import (
	"errors"
	"strings"

	"github.com/edbarbera/gitle/internal/gitcmd"
)

// SendProblem classifies why a send failed. Classifying here — rather than
// leaving every caller to grep git's stderr — means the CLI and the dashboard
// give the same diagnosis and the same advice.
type SendProblem int

const (
	// SendRejected means someone else's work landed online first.
	SendRejected SendProblem = iota
	// SendAuth means the remote wouldn't accept our credentials.
	SendAuth
	// SendUnknown is anything else; Detail carries git's own words.
	SendUnknown
)

// SendError is a failed send, already classified.
type SendError struct {
	Problem SendProblem
	Detail  string // git's first line of stderr, for the unknown case
}

func (e *SendError) Error() string {
	if e.Detail != "" {
		return e.Detail
	}
	switch e.Problem {
	case SendRejected:
		return "there is newer work online"
	case SendAuth:
		return "not signed in"
	default:
		return "could not send"
	}
}

// Errors returned by Send and Grab that describe a situation rather than a
// failure — the caller decides what to say about them.
var (
	// ErrNothingToSend means no saves exist yet.
	ErrNothingToSend = errors.New("nothing saved yet")
	// ErrNoRemote means the project has no online home configured.
	ErrNoRemote = errors.New("no online copy configured")
	// ErrUnsavedChanges means the working tree is dirty, so grabbing could
	// collide with work in progress.
	ErrUnsavedChanges = errors.New("unsaved changes")
	// ErrConflict means incoming work clashed and needs resolving by hand.
	ErrConflict = errors.New("changes clashed")
)

// SendResult reports what a successful send did.
type SendResult struct {
	Branch string
	// FirstPush is true when this send also set up the branch's online
	// counterpart, which is worth mentioning to the user.
	FirstPush bool
}

// SendOptions tunes how a send behaves.
type SendOptions struct {
	// AllowTerminalPrompt lets git ask for credentials on the terminal when no
	// credential helper can answer.
	//
	// This must be false whenever something is drawn over the terminal: git
	// writes its prompt straight to /dev/tty, which would appear through the
	// middle of a rendered frame and read input the UI is also trying to read.
	// With it false, git fails fast with an auth error instead, which the
	// caller can report cleanly.
	AllowTerminalPrompt bool
}

// Send uploads saved work to the online copy. It does not decide whether
// pushing to a shared branch is wise — that's a judgement the caller makes,
// because only the caller can ask.
func Send(opts SendOptions) (SendResult, error) {
	if !gitcmd.HasCommits() {
		return SendResult{}, ErrNothingToSend
	}
	if !gitcmd.HasRemote() {
		return SendResult{}, ErrNoRemote
	}

	branch := gitcmd.CurrentBranch()
	firstPush := !gitcmd.HasUpstream()

	var env []string
	if !opts.AllowTerminalPrompt {
		env = append(env, "GIT_TERMINAL_PROMPT=0")
	}

	args := []string{"push"}
	if firstPush {
		// First push on this branch: -u remembers the destination for next time.
		args = append(args, "-u", "origin", branch)
	}

	stderr, err := gitcmd.RunQuietStderrEnv(env, args...)
	if err != nil {
		return SendResult{}, classifyPush(stderr)
	}
	return SendResult{Branch: branch, FirstPush: firstPush}, nil
}

// classifyPush reads git's stderr and works out which of the few failures
// people actually hit this was.
func classifyPush(stderr string) *SendError {
	low := strings.ToLower(stderr)
	switch {
	case strings.Contains(low, "rejected"),
		strings.Contains(low, "non-fast-forward"),
		strings.Contains(low, "fetch first"):
		return &SendError{Problem: SendRejected}
	case strings.Contains(low, "authentication"),
		strings.Contains(low, "could not read"),
		strings.Contains(low, "permission denied"),
		strings.Contains(low, "access denied"):
		return &SendError{Problem: SendAuth}
	default:
		return &SendError{Problem: SendUnknown, Detail: FirstLine(stderr)}
	}
}

// FirstLine returns the first line of s, trimmed. git error messages are
// often several lines of detail after one useful sentence.
func FirstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

// ErrNoUpstream means this branch has no online counterpart to grab from.
var ErrNoUpstream = errors.New("this line of work isn't online yet")

// Grab downloads and blends in everyone else's latest work.
//
// It refuses to start while there are unsaved changes: a rebase over a dirty
// tree is exactly the situation gitle exists to keep people out of.
func Grab() error {
	if gitcmd.HasChanges() {
		return ErrUnsavedChanges
	}
	if !gitcmd.HasRemote() {
		return ErrNoRemote
	}
	if !gitcmd.HasUpstream() {
		// Without an upstream git prints a nine-line lecture about
		// --set-upstream-to. Catch it first and say something useful instead.
		return ErrNoUpstream
	}

	// --rebase replays your saves on top of the latest, keeping history tidy.
	stderr, err := gitcmd.RunQuietStderr("pull", "--rebase")
	if err != nil {
		if gitcmd.CurrentOp() != gitcmd.OpNone {
			return ErrConflict
		}
		return errors.New(FirstLine(stderr))
	}
	return nil
}
