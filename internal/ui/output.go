// Package ui prints friendly, plain-English feedback and asks the questions
// gitle needs answered. Non-technical users see what happened and what to do
// next, not raw git jargon.
//
// Everything here is for gitle's *line-at-a-time* mode: one command runs,
// prints, and exits. The full-screen dashboard in internal/tui owns the
// screen itself and must not call these printers while it's running.
package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/term"

	"github.com/edbarbera/gitle/internal/theme"
)

// Output goes through colorprofile writers rather than straight to the file.
// They downsample or strip styling to match the destination, which is how
// gitle honours NO_COLOR, TERM=dumb, CLICOLOR_FORCE and redirection to a file
// without checking any of that itself.
var (
	stdout io.Writer = colorprofile.NewWriter(os.Stdout, os.Environ())
	stderr io.Writer = colorprofile.NewWriter(os.Stderr, os.Environ())
)

// SetOutput redirects gitle's output, for tests and for callers that need to
// capture what would have been printed.
func SetOutput(out, err io.Writer) {
	stdout, stderr = out, err
}

// Success prints a green check line.
func Success(format string, a ...any) {
	fmt.Fprintln(stdout, theme.Success.Render("✓ ")+fmt.Sprintf(format, a...))
}

// Info prints a neutral line.
func Info(format string, a ...any) {
	fmt.Fprintln(stdout, theme.Info.Render("→ ")+fmt.Sprintf(format, a...))
}

// Warn prints a yellow warning line to stderr.
func Warn(format string, a ...any) {
	fmt.Fprintln(stderr, theme.Warn.Render("! ")+fmt.Sprintf(format, a...))
}

// Error prints a red error line to stderr.
func Error(format string, a ...any) {
	fmt.Fprintln(stderr, theme.Error.Render("✗ ")+fmt.Sprintf(format, a...))
}

// Hint prints a dim follow-up suggestion.
func Hint(format string, a ...any) {
	fmt.Fprintln(stdout, theme.Hint.Render("  "+fmt.Sprintf(format, a...)))
}

// Plain prints a line with no prefix.
func Plain(format string, a ...any) {
	fmt.Fprintf(stdout, format+"\n", a...)
}

// Blank prints an empty line.
func Blank() {
	fmt.Fprintln(stdout)
}

// Bold returns s emphasised.
func Bold(s string) string { return theme.Bold.Render(s) }

// Green, Yellow, Red, Cyan and Dim return s in the given style. Styling is
// stripped later by the output writer when the destination can't show it.
func Green(s string) string  { return theme.GreenText.Render(s) }
func Yellow(s string) string { return theme.YellowText.Render(s) }
func Red(s string) string    { return theme.RedText.Render(s) }
func Cyan(s string) string   { return theme.CyanText.Render(s) }
func Dim(s string) string    { return theme.DimText.Render(s) }

// Interactive reports whether gitle can hold a conversation: both ends of the
// pipe must be a real terminal. Checking stdin alone isn't enough — with
// `gitle save > log.txt` the keyboard is still attached but drawing a prompt
// into a file helps nobody.
//
// GITLE_NO_TUI=1 forces the plain, non-interactive path. It's the escape
// hatch for scripts, for tests, and for terminals where the UI misbehaves.
func Interactive() bool {
	if os.Getenv("GITLE_NO_TUI") != "" {
		return false
	}
	return term.IsTerminal(os.Stdin.Fd()) && term.IsTerminal(os.Stdout.Fd())
}

// IsInteractive is the old name for Interactive.
//
// Deprecated: use Interactive.
func IsInteractive() bool { return Interactive() }

// Banner prints the playful gitle welcome art.
func Banner() {
	art := "" +
		"       _ _   _\n" +
		"  __ _(_) |_| | ___\n" +
		" / _` | | __| |/ _ \\\n" +
		"| (_| | | |_| |  __/\n" +
		" \\__, |_|\\__|_|\\___|\n" +
		" |___/"
	// Styled line by line: rendering the block in one go pads every line out
	// to the width of the longest, leaving trailing spaces on each.
	for _, line := range strings.Split(art, "\n") {
		fmt.Fprintln(stdout, theme.Banner.Render(line))
	}
	fmt.Fprintln(stdout, theme.Bold.Render("  git, made friendly")+" ✨")
	fmt.Fprintln(stdout)
}

// Step prints a numbered wizard step header.
func Step(n, total int, title string) {
	fmt.Fprintf(stdout, "\n%s %s\n",
		theme.Info.Render(fmt.Sprintf("[%d/%d]", n, total)),
		theme.Bold.Render(title))
}

// Celebrate prints a cheerful closing line.
func Celebrate(format string, a ...any) {
	fmt.Fprintln(stdout, "\n"+theme.Success.Render("🎉 ")+theme.Bold.Render(fmt.Sprintf(format, a...)))
}
