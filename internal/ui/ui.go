// Package ui prints friendly, plain-English feedback. Non-technical users see
// what happened and what to do next, not raw git jargon.
package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Color is disabled when NO_COLOR is set or stdout is not a terminal-ish env.
var useColor = os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"

const (
	reset  = "\033[0m"
	green  = "\033[32m"
	red    = "\033[31m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
	dim    = "\033[2m"
	bold   = "\033[1m"
)

func paint(code, s string) string {
	if !useColor {
		return s
	}
	return code + s + reset
}

// Success prints a green check line.
func Success(format string, a ...any) {
	fmt.Println(paint(green, "✓ ") + fmt.Sprintf(format, a...))
}

// Info prints a neutral line.
func Info(format string, a ...any) {
	fmt.Println(paint(cyan, "→ ") + fmt.Sprintf(format, a...))
}

// Warn prints a yellow warning line to stderr.
func Warn(format string, a ...any) {
	fmt.Fprintln(os.Stderr, paint(yellow, "! ")+fmt.Sprintf(format, a...))
}

// Error prints a red error line to stderr.
func Error(format string, a ...any) {
	fmt.Fprintln(os.Stderr, paint(red, "✗ ")+fmt.Sprintf(format, a...))
}

// Hint prints a dim follow-up suggestion.
func Hint(format string, a ...any) {
	fmt.Println(paint(dim, "  "+fmt.Sprintf(format, a...)))
}

// Bold returns s emphasised.
func Bold(s string) string { return paint(bold, s) }

// Confirm asks a yes/no question and returns true only on an explicit yes.
// Defaults to no on empty input, so destructive actions require intent.
func Confirm(question string) bool {
	fmt.Print(paint(yellow, "? ") + question + " " + paint(dim, "[y/N] "))
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}
