package ui

import (
	"errors"
	"os"
	"os/exec"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/edbarbera/gitle/internal/theme"
)

// formTheme is gitle's look for every prompt. It's a ThemeFunc rather than a
// fixed set of styles so huh can pick the light or dark variant itself, from
// the terminal it's about to draw on.
var formTheme = huh.ThemeFunc(huh.ThemeCharm)

// accessible reports whether to fall back to plain, screen-reader-friendly
// prompts instead of the drawn ones. ACCESSIBLE is the convention huh uses.
func accessible() bool { return os.Getenv("ACCESSIBLE") != "" }

// run executes a one-field form. Aborting (Ctrl-C or Esc) is reported as
// false so callers can apply their own cancellation behaviour; every other
// failure is treated the same way, because a prompt that can't be drawn must
// never be worse than a prompt that was declined.
func run(field huh.Field) (ok bool) {
	form := huh.NewForm(huh.NewGroup(field)).
		WithTheme(formTheme).
		WithAccessible(accessible())

	err := form.Run()
	if err != nil {
		if !errors.Is(err, huh.ErrUserAborted) {
			Error("%s", err)
		}
		return false
	}
	return true
}

// Confirm asks a yes/no question and returns true only on an explicit yes.
// Defaults to no on cancellation, so destructive actions require intent.
func Confirm(question string) bool {
	return ConfirmDefault(question, false)
}

// ConfirmDefault asks a yes/no question with a chosen default. When there's no
// terminal (piped/scripted), it returns the default without prompting.
func ConfirmDefault(question string, def bool) bool {
	if !Interactive() {
		return def
	}
	answer := def
	if !run(huh.NewConfirm().Title(question).Value(&answer)) {
		return def
	}
	return answer
}

// Ask prompts for a line of free text, pre-filled with def. Returning empty is
// a real answer — it means the user cleared the field on purpose — so callers
// that need a value must check for it. Without a terminal it returns def.
func Ask(question, def string) string {
	if !Interactive() {
		return def
	}
	answer := def
	if !run(huh.NewInput().Title(question).Value(&answer)) {
		return def
	}
	return answer
}

// AskLong prompts for multi-line text, pre-filled with def. Used where a
// single line is too cramped, like a longer description of a save.
func AskLong(question, def string) string {
	if !Interactive() {
		return def
	}
	answer := def
	if !run(huh.NewText().Title(question).Lines(5).Value(&answer)) {
		return def
	}
	return answer
}

// Pick shows a checklist with everything ticked by default and returns the
// indices left ticked. With no terminal (piped/scripted) it selects
// everything, preserving the "save all" default. Cancelling selects nothing.
func Pick(prompt string, items []string) []int {
	if len(items) == 0 {
		return nil
	}
	if !Interactive() {
		return allIndices(len(items))
	}

	options := make([]huh.Option[int], len(items))
	for i, it := range items {
		// Everything starts ticked: the common case is saving all your work,
		// and unticking the odd file is less work than ticking the rest.
		options[i] = huh.NewOption(it, i).Selected(true)
	}
	chosen := allIndices(len(items))

	if !run(huh.NewMultiSelect[int]().
		Title(prompt).
		Options(options...).
		Filterable(true).
		Height(listHeight(len(items))).
		Value(&chosen)) {
		return nil
	}
	return chosen
}

// Choose shows a menu where exactly one item can be picked. Returns the chosen
// index, or -1 if there's no terminal to ask on or the user cancels.
func Choose(prompt string, items []string) int {
	if len(items) == 0 || !Interactive() {
		return -1
	}

	options := make([]huh.Option[int], len(items))
	for i, it := range items {
		options[i] = huh.NewOption(it, i)
	}
	chosen := 0

	if !run(huh.NewSelect[int]().
		Title(prompt).
		Options(options...).
		Height(listHeight(len(items))).
		Value(&chosen)) {
		return -1
	}
	return chosen
}

// listHeight caps how tall a list gets before it scrolls, leaving room for the
// title and help line on a standard 24-row terminal.
func listHeight(items int) int {
	const max = 15
	if items > max {
		return max
	}
	return items + 2
}

// spinnerDone tells the spinner model the work has finished.
type spinnerDone struct{}

type spinnerModel struct {
	spin  spinner.Model
	title string
}

func (m spinnerModel) Init() tea.Cmd { return m.spin.Tick }

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case spinnerDone:
		return m, tea.Quit
	case tea.KeyPressMsg:
		// Ctrl-C dismisses the spinner, but the work itself keeps running to
		// completion — interrupting a half-finished git operation is worse
		// than waiting for it.
		return m, tea.Quit
	}
	var cmd tea.Cmd
	m.spin, cmd = m.spin.Update(msg)
	return m, cmd
}

func (m spinnerModel) View() tea.View {
	return tea.NewView(m.spin.View() + " " + m.title)
}

// Spinner runs fn while showing an animated status line, so slow work (a
// network call, a big diff) doesn't look like a hang. Without a terminal it
// just runs fn. fn's error is returned untouched either way.
func Spinner(title string, fn func() error) error {
	if !Interactive() {
		return fn()
	}

	m := spinnerModel{
		spin:  spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(theme.Info)),
		title: title,
	}
	p := tea.NewProgram(m)

	// Buffered so the worker never blocks on a UI that has already gone away.
	result := make(chan error, 1)
	go func() {
		result <- fn()
		p.Send(spinnerDone{})
	}()

	// Even if the UI fails to start, wait for the work: the caller asked for
	// fn to run, and the animation was only ever decoration.
	_, _ = p.Run()
	return <-result
}

// OpenEditor opens path in the user's configured editor ($GIT_EDITOR or
// $EDITOR, falling back to vi) and waits for it to close.
func OpenEditor(path string) error {
	cmd := editorCommand(path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// EditorCmd is OpenEditor for use inside a Bubble Tea program: the returned
// command must be handed to tea.ExecProcess so the running UI releases the
// terminal first and restores it afterwards.
func EditorCmd(path string) *exec.Cmd { return editorCommand(path) }

func editorCommand(path string) *exec.Cmd {
	editor := os.Getenv("GIT_EDITOR")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}
	// Run through a shell so editors configured with arguments (e.g. "code
	// --wait") work, while still passing path safely as a single argument.
	return exec.Command("sh", "-c", editor+` "$1"`, "sh", path)
}

func allIndices(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
}
