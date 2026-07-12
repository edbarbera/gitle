// Package ui prints friendly, plain-English feedback. Non-technical users see
// what happened and what to do next, not raw git jargon.
package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// Color is disabled when NO_COLOR is set or stdout is not a terminal-ish env.
var useColor = os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"

// stdin is a single shared buffered reader. It must be shared across every
// prompt: bufio reads ahead in chunks, so a fresh reader per prompt would
// swallow and discard any input queued behind the current line.
var stdin = bufio.NewReader(os.Stdin)

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

// Green, Yellow, Red return s in the given colour (respecting NO_COLOR).
func Green(s string) string  { return paint(green, s) }
func Yellow(s string) string { return paint(yellow, s) }
func Red(s string) string    { return paint(red, s) }

// Plain prints a line with no prefix.
func Plain(format string, a ...any) {
	fmt.Printf(format+"\n", a...)
}

// IsInteractive reports whether we're attached to a real terminal, so we know
// whether prompting the user makes sense (vs. being piped or scripted).
func IsInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// Confirm asks a yes/no question and returns true only on an explicit yes.
// Defaults to no on empty input, so destructive actions require intent.
func Confirm(question string) bool {
	return ConfirmDefault(question, false)
}

// ConfirmDefault asks a yes/no question with a chosen default. When there's no
// terminal (piped/scripted), it returns the default without prompting.
func ConfirmDefault(question string, def bool) bool {
	if !IsInteractive() {
		return def
	}
	hint := "[y/N]"
	if def {
		hint = "[Y/n]"
	}
	fmt.Print(paint(yellow, "? ") + question + " " + paint(dim, hint+" "))
	line, err := stdin.ReadString('\n')
	if err != nil {
		return def
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	default:
		return def
	}
}

// Ask prompts for a line of free text, showing def as the fallback. Returns def
// on empty input or when there's no terminal.
func Ask(question, def string) string {
	if !IsInteractive() {
		return def
	}
	if def != "" {
		fmt.Print(paint(yellow, "? ") + question + " " + paint(dim, "["+def+"] "))
	} else {
		fmt.Print(paint(yellow, "? ") + question + " ")
	}
	line, err := stdin.ReadString('\n')
	if err != nil {
		return def
	}
	if line = strings.TrimSpace(line); line != "" {
		return line
	}
	return def
}

// Pick shows an arrow-key checklist with everything ticked by default. Move the
// highlight with ↑/↓ (or j/k), toggle it with Space, press Enter to confirm.
// It returns the indices left ticked. With no terminal (piped/scripted) it
// selects everything, preserving the "save all" default.
func Pick(prompt string, items []string) []int {
	if len(items) == 0 {
		return nil
	}
	if !IsInteractive() {
		return allIndices(len(items))
	}
	fd := int(os.Stdin.Fd())
	old, err := term.MakeRaw(fd)
	if err != nil {
		// Terminal won't switch to per-key mode; fall back to saving all.
		return allIndices(len(items))
	}
	defer term.Restore(fd, old)

	selected := make([]bool, len(items))
	setAll(selected, true)
	cursor := 0

	// draw renders the list in place, redrawing over the previous frame.
	draw := func(first bool) {
		if !first {
			fmt.Fprintf(os.Stdout, "\033[%dA", len(items)+2) // jump back to top
		}
		fmt.Fprintf(os.Stdout, "\r\033[K%s\r\n", paint(bold, "[?] "+prompt))
		for i, it := range items {
			pointer := "  "
			if i == cursor {
				pointer = paint(cyan, "❯ ")
			}
			mark := paint(dim, "○")
			if selected[i] {
				mark = paint(green, "●")
			}
			fmt.Fprintf(os.Stdout, "\r\033[K%s%s %s\r\n", pointer, mark, it)
		}
		fmt.Fprintf(os.Stdout, "\r\033[K%s\r\n",
			paint(dim, "↑/↓ move · space toggle · a all · n none · enter confirm"))
	}

	draw(true)
	for {
		b, err := stdin.ReadByte()
		if err != nil {
			break
		}
		done := false
		switch b {
		case 3: // Ctrl-C: cancel, select nothing
			setAll(selected, false)
			done = true
		case '\r', '\n':
			done = true
		case ' ':
			selected[cursor] = !selected[cursor]
		case 'a', 'A':
			setAll(selected, true)
		case 'n', 'N':
			setAll(selected, false)
		case 'k':
			if cursor > 0 {
				cursor--
			}
		case 'j':
			if cursor < len(items)-1 {
				cursor++
			}
		case 0x1b: // escape sequence — arrow keys arrive as ESC [ A/B
			if b2, _ := stdin.ReadByte(); b2 == '[' {
				switch b3, _ := stdin.ReadByte(); b3 {
				case 'A':
					if cursor > 0 {
						cursor--
					}
				case 'B':
					if cursor < len(items)-1 {
						cursor++
					}
				}
			}
		}
		if done {
			break
		}
		draw(false)
	}
	return trueIndices(selected)
}

func allIndices(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
}

func setAll(b []bool, v bool) {
	for i := range b {
		b[i] = v
	}
}

func trueIndices(b []bool) []int {
	var out []int
	for i, v := range b {
		if v {
			out = append(out, i)
		}
	}
	return out
}

// Banner prints the playful gitle welcome art.
func Banner() {
	art := "" +
		"   __ _(_) |_| | ___\n" +
		"  / _` | | __| |/ _ \\\n" +
		" | (_| | | |_| |  __/\n" +
		"  \\__, |_|\\__|_|\\___|\n" +
		"  |___/"
	fmt.Println(paint(cyan, art))
	fmt.Println(paint(bold, "  git, made friendly") + " ✨")
	fmt.Println()
}

// Step prints a numbered wizard step header.
func Step(n, total int, title string) {
	fmt.Printf("\n%s %s\n", paint(cyan, fmt.Sprintf("[%d/%d]", n, total)), paint(bold, title))
}

// Celebrate prints a cheerful closing line.
func Celebrate(format string, a ...any) {
	fmt.Println("\n" + paint(green, "🎉 ") + paint(bold, fmt.Sprintf(format, a...)))
}
