// Package gitcmd wraps the real git binary. gitle never reimplements git; it
// shells out so users get their own git config, credentials and hooks.
package gitcmd

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

// StatusPorcelain returns `git status --porcelain` as raw lines. Unlike
// Capture it preserves each line's leading status characters (which are
// significant and include spaces), trimming only the trailing newline.
func StatusPorcelain() ([]string, error) {
	if !Available() {
		return nil, ErrGitMissing
	}
	cmd := exec.Command("git", "status", "--porcelain")
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(errBuf.String()); msg != "" {
			return nil, errors.New(msg)
		}
		return nil, err
	}
	raw := strings.TrimRight(out.String(), "\n")
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
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

// RunCaptureStderr runs git streaming stdout to the terminal (and keeping the
// user's stdin/tty for auth prompts), while capturing stderr so the caller can
// translate git's error into plain English. Returns the captured stderr.
func RunCaptureStderr(args ...string) (string, error) {
	if !Available() {
		return "", ErrGitMissing
	}
	cmd := exec.Command("git", args...)
	var errBuf bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = &errBuf
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	return errBuf.String(), err
}

// HasUpstream reports whether the current branch has a configured upstream to
// push/pull against.
func HasUpstream() bool {
	_, err := Capture("rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	return err == nil
}

// ConfigGet returns the value of a git config key, or "" if unset.
func ConfigGet(key string) string {
	out, err := Capture("config", key)
	if err != nil {
		return ""
	}
	return out
}

// ConfigSetGlobal writes a git config key at global (per-user) scope, so it
// applies to every repo on this machine.
func ConfigSetGlobal(key, value string) error {
	return Run("config", "--global", key, value)
}

// HasRemote reports whether any remote is configured.
func HasRemote() bool {
	out, err := Capture("remote")
	return err == nil && out != ""
}

// AddRemote adds a named remote pointing at url.
func AddRemote(name, url string) error {
	return Run("remote", "add", name, url)
}

// BranchExists reports whether a local branch of the given name exists.
func BranchExists(name string) bool {
	err := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+name).Run()
	return err == nil
}

// RepoName returns the name of the repository's top-level folder.
func RepoName() string {
	top, err := Capture("rev-parse", "--show-toplevel")
	if err != nil {
		return ""
	}
	return filepath.Base(top)
}

// MainBranch returns the name of the project's main line of work ("main" or
// "master"), or "" if neither exists.
func MainBranch() string {
	switch {
	case BranchExists("main"):
		return "main"
	case BranchExists("master"):
		return "master"
	default:
		return ""
	}
}

// AheadBehind counts how far HEAD is ahead of and behind the given ref (e.g.
// "main" or "@{upstream}"). ok is false if the comparison can't be made.
func AheadBehind(ref string) (ahead, behind int, ok bool) {
	out, err := Capture("rev-list", "--left-right", "--count", ref+"...HEAD")
	if err != nil {
		return 0, 0, false
	}
	parts := strings.Fields(out) // "<behind>\t<ahead>"
	if len(parts) != 2 {
		return 0, 0, false
	}
	behind, _ = strconv.Atoi(parts[0])
	ahead, _ = strconv.Atoi(parts[1])
	return ahead, behind, true
}
