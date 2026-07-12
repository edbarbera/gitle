// Package gitcmd wraps the real git binary. gitle never reimplements git; it
// shells out so users get their own git config, credentials and hooks.
package gitcmd

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
)

// ErrGitMissing means the git binary was not found on PATH.
var ErrGitMissing = errors.New("git is not installed")

// Available reports whether the git binary can be found on PATH.
func Available() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// Run executes `git <args...>` streaming git's own output straight to the
// user's terminal. Used for commands whose native output we want to show
// (history, branches, push, pull).
func Run(args ...string) error {
	if !Available() {
		return ErrGitMissing
	}
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// Capture executes `git <args...>` and returns trimmed stdout. stderr is
// captured and returned inside the error so callers can surface it.
func Capture(args ...string) (string, error) {
	if !Available() {
		return "", ErrGitMissing
	}
	cmd := exec.Command("git", args...)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(errBuf.String()); msg != "" {
			return "", errors.New(msg)
		}
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

// InRepo reports whether the current directory is inside a git working tree.
func InRepo() bool {
	out, err := Capture("rev-parse", "--is-inside-work-tree")
	return err == nil && out == "true"
}

// CurrentBranch returns the checked-out branch name, or "" if detached/unknown.
func CurrentBranch() string {
	out, err := Capture("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil || out == "HEAD" {
		return ""
	}
	return out
}

// HasStagedOrUnstagedChanges reports whether the working tree has any changes
// (staged, unstaged, or untracked) that a commit could capture.
func HasChanges() bool {
	out, err := Capture("status", "--porcelain")
	return err == nil && strings.TrimSpace(out) != ""
}

// HasCommits reports whether the repo has at least one commit yet.
func HasCommits() bool {
	_, err := Capture("rev-parse", "HEAD")
	return err == nil
}

// HasUpstream reports whether the current branch has a configured upstream to
// push/pull against.
func HasUpstream() bool {
	_, err := Capture("rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	return err == nil
}

// BranchExists reports whether a local branch of the given name exists.
func BranchExists(name string) bool {
	err := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+name).Run()
	return err == nil
}
