package tui

import (
	"image/color"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/edbarbera/gitle/internal/gitcmd"
	"github.com/edbarbera/gitle/internal/ops"
	"github.com/edbarbera/gitle/internal/theme"
	"github.com/edbarbera/gitle/internal/ui"
)

// The conflict resolver walks one clashing section at a time, showing both
// versions side by side and asking which to keep. It's the part of git that
// most often defeats people, so nothing here mentions merges, HEADs or
// markers: just "your version", "the version you grabbed", and a choice.

type conflictKeyMap struct {
	Ours     key.Binding
	Theirs   key.Binding
	Both     key.Binding
	Edit     key.Binding
	Next     key.Binding
	Prev     key.Binding
	SkipFile key.Binding
	Abort    key.Binding
	Quit     key.Binding
}

var conflictKeys = conflictKeyMap{
	Ours: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "keep yours"),
	),
	Theirs: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "keep theirs"),
	),
	Both: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "keep both"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit by hand"),
	),
	Next: key.NewBinding(
		key.WithKeys("down", "j", "n"),
		key.WithHelp("↓/j", "next section"),
	),
	Prev: key.NewBinding(
		key.WithKeys("up", "k", "p"),
		key.WithHelp("↑/k", "previous"),
	),
	SkipFile: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "skip this file"),
	),
	Abort: key.NewBinding(
		key.WithKeys("ctrl+x"),
		key.WithHelp("ctrl+x", "undo everything"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c", "esc"),
		key.WithHelp("q", "stop for now"),
	),
}

func (k conflictKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Ours, k.Theirs, k.Both, k.Edit, k.Next, k.SkipFile, k.Quit}
}

func (k conflictKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Ours, k.Theirs, k.Both},
		{k.Edit, k.Next, k.Prev},
		{k.SkipFile, k.Abort, k.Quit},
	}
}

// conflictModel resolves every clashing file, one section at a time.
type conflictModel struct {
	state ops.ConflictState

	// files still to deal with, and where we are in that list.
	files    []string
	fileIdx  int
	file     *ops.ConflictFile
	choices  []ops.Side // one per hunk in the current file
	decided  []bool
	hunkIdx  int
	resolved int
	skipped  []string

	viewport viewport.Model
	help     help.Model

	width, height int
	ready         bool
	done          bool
	aborted       bool
	note          string
	err           error
}

// conflictLoadedMsg carries the next file, parsed.
type conflictLoadedMsg struct {
	file *ops.ConflictFile
	err  error
}

// conflictFinishedMsg reports the outcome of finishing or aborting.
type conflictFinishedMsg struct {
	note string
	err  error
}

// editedMsg comes back after the user's editor closes.
type editedMsg struct{ err error }

// RunConflicts opens the resolver. It returns whether every clash was sorted
// out, so the caller can decide what to say afterwards.
func RunConflicts(state ops.ConflictState) error {
	m := conflictModel{
		state: state,
		files: state.Files,
		help:  help.New(),
	}
	m.viewport = viewport.New()
	_, err := tea.NewProgram(m).Run()
	return err
}

func (m conflictModel) Init() tea.Cmd {
	return loadConflictFile(m.files, 0)
}

// loadConflictFile parses the file at idx, if there is one.
func loadConflictFile(files []string, idx int) tea.Cmd {
	if idx >= len(files) {
		return nil
	}
	return func() tea.Msg {
		f, err := ops.LoadConflictFile(files[idx])
		return conflictLoadedMsg{file: f, err: err}
	}
}

func (m conflictModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		m.resize()
		return m, nil

	case conflictLoadedMsg:
		if msg.err != nil {
			// An unparsable file is left alone rather than mangled; the user
			// can open it themselves.
			m.err = msg.err
			m.skipped = append(m.skipped, m.files[m.fileIdx])
			return m.advanceFile()
		}
		m.file = msg.file
		m.choices = make([]ops.Side, len(msg.file.Hunks))
		m.decided = make([]bool, len(msg.file.Hunks))
		m.hunkIdx = 0
		m.err = nil
		m.refreshViewport()
		// A conflicted file with no markers left has already been sorted out
		// by hand; just mark it and move on.
		if len(msg.file.Hunks) == 0 {
			if err := ops.MarkResolved(msg.file.Path); err != nil {
				m.err = err
				return m, nil
			}
			m.resolved++
			return m.advanceFile()
		}
		return m, nil

	case editedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		path := m.file.Path
		if ops.StillConflicted(path) {
			m.note = "Still some markers left in that file — try again when you're ready."
			return m, nil
		}
		if err := ops.MarkResolved(path); err != nil {
			m.err = err
			return m, nil
		}
		m.resolved++
		return m.advanceFile()

	case conflictFinishedMsg:
		m.done = true
		m.note, m.err = msg.note, msg.err
		return m, tea.Quit

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m conflictModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, conflictKeys.Quit):
		return m, tea.Quit

	case key.Matches(msg, conflictKeys.Abort):
		return m, func() tea.Msg {
			if err := ops.AbortOp(m.state.Op); err != nil {
				return conflictFinishedMsg{err: err}
			}
			return conflictFinishedMsg{note: "Back to where you started — nothing was lost."}
		}

	case key.Matches(msg, conflictKeys.Ours):
		return m.choose(ops.SideOurs)
	case key.Matches(msg, conflictKeys.Theirs):
		return m.choose(ops.SideTheirs)
	case key.Matches(msg, conflictKeys.Both):
		return m.choose(ops.SideBoth)

	case key.Matches(msg, conflictKeys.Edit):
		if m.file == nil {
			return m, nil
		}
		// Hand the terminal over properly, then take it back: anything less
		// and the editor and the interface fight over the screen.
		return m, tea.ExecProcess(ui.EditorCmd(m.file.Path), func(err error) tea.Msg {
			return editedMsg{err: err}
		})

	case key.Matches(msg, conflictKeys.SkipFile):
		if m.file == nil {
			return m, nil
		}
		m.skipped = append(m.skipped, m.file.Path)
		return m.advanceFile()

	case key.Matches(msg, conflictKeys.Next):
		if m.file != nil && m.hunkIdx < len(m.file.Hunks)-1 {
			m.hunkIdx++
			m.refreshViewport()
		}
		return m, nil

	case key.Matches(msg, conflictKeys.Prev):
		if m.hunkIdx > 0 {
			m.hunkIdx--
			m.refreshViewport()
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// choose records a decision for the current section and moves on. Once every
// section has an answer, the file is written and staged.
func (m conflictModel) choose(side ops.Side) (tea.Model, tea.Cmd) {
	if m.file == nil || len(m.file.Hunks) == 0 {
		return m, nil
	}
	m.choices[m.hunkIdx] = side
	m.decided[m.hunkIdx] = true

	if next, ok := m.firstUndecided(); ok {
		m.hunkIdx = next
		m.refreshViewport()
		return m, nil
	}

	// Everything in this file is decided — write it out.
	if err := m.file.Resolve(m.choices); err != nil {
		m.err = err
		return m, nil
	}
	m.resolved++
	return m.advanceFile()
}

func (m conflictModel) firstUndecided() (int, bool) {
	for i, d := range m.decided {
		if !d {
			return i, true
		}
	}
	return 0, false
}

// advanceFile moves to the next conflicted file, or finishes up if that was
// the last one.
func (m conflictModel) advanceFile() (tea.Model, tea.Cmd) {
	m.fileIdx++
	m.file = nil

	if m.fileIdx < len(m.files) {
		return m, loadConflictFile(m.files, m.fileIdx)
	}

	// Anything skipped means the operation can't be completed yet.
	if len(m.skipped) > 0 {
		m.done = true
		m.note = "Some files still need attention — run gitle fix-conflicts again when ready."
		return m, tea.Quit
	}

	op := m.state.Op
	return m, func() tea.Msg {
		if err := ops.FinishOp(op); err != nil {
			return conflictFinishedMsg{err: err}
		}
		return conflictFinishedMsg{note: "All conflicts resolved!"}
	}
}

func (m *conflictModel) resize() {
	m.viewport.SetWidth(max(20, m.width-4))
	m.viewport.SetHeight(max(5, m.height-8))
}

// refreshViewport renders the current section's two versions.
func (m *conflictModel) refreshViewport() {
	if m.file == nil || len(m.file.Hunks) == 0 {
		m.viewport.SetContent("")
		return
	}
	h := m.file.Hunks[m.hunkIdx]
	half := max(10, (m.width-6)/2)

	left := sideBox(m.state.HeadLabel, h.Ours, half, theme.Green)
	right := sideBox(m.state.OtherLabel, h.Theirs, half, theme.Yellow)

	m.viewport.SetContent(lipgloss.JoinHorizontal(lipgloss.Top, left, right))
	m.viewport.GotoTop()
}

// sideBox renders one version of a clashing section.
func sideBox(title string, lines []string, width int, accent color.Color) string {
	body := strings.Join(lines, "\n")
	if len(lines) == 0 {
		body = theme.DimText.Render("(nothing here)")
	}
	head := lipgloss.NewStyle().Bold(true).Foreground(accent).Render(title)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Width(width).
		Padding(0, 1).
		Render(head + "\n\n" + body)
}

func (m conflictModel) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

func (m conflictModel) render() string {
	if !m.ready {
		return "Loading..."
	}
	if m.done {
		return ""
	}

	var sb strings.Builder

	// Header: which file, and how far through we are.
	path := ""
	if m.file != nil {
		path = m.file.Path
	}
	sb.WriteString(theme.Header.Render("Sorting out clashes") + theme.DimText.Render(
		" · file "+strconv.Itoa(min(m.fileIdx+1, len(m.files)))+" of "+strconv.Itoa(len(m.files))))
	sb.WriteString("\n" + theme.Bold.Render(path))
	if m.file != nil && len(m.file.Hunks) > 0 {
		sb.WriteString(theme.DimText.Render(
			"  section " + strconv.Itoa(m.hunkIdx+1) + " of " + strconv.Itoa(len(m.file.Hunks))))
	}
	sb.WriteString("\n\n")

	sb.WriteString(m.viewport.View())
	sb.WriteString("\n")

	switch {
	case m.err != nil:
		sb.WriteString(theme.Error.Render("✗ " + ops.FirstLine(m.err.Error())))
	case m.note != "":
		sb.WriteString(theme.Warn.Render("! " + m.note))
	default:
		sb.WriteString(theme.DimText.Render("Which version do you want to keep here?"))
	}
	sb.WriteString("\n")
	sb.WriteString(m.help.View(conflictKeys))
	return sb.String()
}

// Outcome describes how a resolver session ended, so the caller can print a
// closing line in gitle's own voice.
type Outcome struct {
	Resolved int
	Skipped  []string
	Note     string
	Op       gitcmd.OpKind
}
