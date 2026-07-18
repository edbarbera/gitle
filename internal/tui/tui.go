// Package tui is gitle's full-screen dashboard: one live view of what's
// changed, which lines of work exist, and what's been saved, with the everyday
// actions a keypress away.
//
// It renders through Bubble Tea and never prints directly. All of its git work
// goes through internal/ops, inside tea.Cmds, so a slow network call can't
// freeze the interface.
package tui

import (
	"errors"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/edbarbera/gitle/internal/ops"
	"github.com/edbarbera/gitle/internal/theme"
)

// pane identifies one of the stacked lists down the left-hand side.
type pane int

const (
	paneChanges pane = iota
	paneBranches
	paneHistory
	paneCount
)

func (p pane) title() string {
	switch p {
	case paneChanges:
		return "Changes"
	case paneBranches:
		return "Lines of work"
	default:
		return "History"
	}
}

// Model is the dashboard.
type Model struct {
	status   ops.Status
	branches []ops.Branch
	history  []ops.SavedPoint

	focus  pane
	cursor [paneCount]int

	detail  viewport.Model
	help    help.Model
	spin    spinner.Model
	form    *saveForm
	showAll bool // expanded help overlay

	width, height int
	ready         bool
	busy          bool
	note          string // transient footer message
	err           error
}

// New builds a dashboard ready to run.
func New() Model {
	h := help.New()
	return Model{
		detail: viewport.New(),
		help:   h,
		spin:   spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(theme.Info)),
	}
}

// Run starts the dashboard and blocks until the user quits.
func Run() error {
	// The alt screen means quitting restores whatever was on the terminal
	// before, rather than leaving a dead frame in the scrollback.
	p := tea.NewProgram(New())
	_, err := p.Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(load(), m.spin.Tick)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		m.ready = true
		return m, nil

	case loadedMsg:
		m.busy = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.status, m.branches, m.history = msg.status, msg.branches, msg.history
		m.clampCursors()
		return m, m.refreshDetail()

	case detailMsg:
		if msg.err != nil {
			m.detail.SetContent("Couldn't read that: " + msg.err.Error())
			return m, nil
		}
		m.detail.SetContent(msg.content)
		m.detail.GotoTop()
		return m, nil

	case doneMsg:
		m.busy = false
		if msg.err != nil {
			m.err = msg.err
			m.note = ""
		} else {
			m.note = msg.note
			m.err = nil
		}
		// Whatever just happened almost certainly changed the repo, so pull
		// fresh state rather than trying to patch what's on screen.
		return m, load()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	// Anything unrecognised still needs to reach the form and the viewport.
	if m.form != nil {
		return m.updateForm(msg)
	}
	var cmd tea.Cmd
	m.detail, cmd = m.detail.Update(msg)
	return m, cmd
}

// handleKey routes a keypress, giving an open form first refusal.
func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.form != nil {
		return m.updateForm(msg)
	}

	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, keys.Help):
		m.showAll = !m.showAll
		m.help.ShowAll = m.showAll
		m.layout()
		return m, nil

	case key.Matches(msg, keys.Refresh):
		m.busy = true
		m.note = "Refreshing..."
		return m, load()

	case key.Matches(msg, keys.Next):
		m.focus = (m.focus + 1) % paneCount
		return m, m.refreshDetail()

	case key.Matches(msg, keys.Prev):
		m.focus = (m.focus + paneCount - 1) % paneCount
		return m, m.refreshDetail()

	case key.Matches(msg, keys.Up):
		if m.cursor[m.focus] > 0 {
			m.cursor[m.focus]--
		}
		return m, m.refreshDetail()

	case key.Matches(msg, keys.Down):
		if m.cursor[m.focus] < m.paneLength(m.focus)-1 {
			m.cursor[m.focus]++
		}
		return m, m.refreshDetail()

	case key.Matches(msg, keys.Enter):
		return m.activate()

	case key.Matches(msg, keys.Save):
		return m.startSave()

	case key.Matches(msg, keys.Send):
		if m.busy {
			return m, nil
		}
		m.busy, m.note, m.err = true, "Sending your work online...", nil
		return m, send()

	case key.Matches(msg, keys.Grab):
		if m.busy {
			return m, nil
		}
		m.busy, m.note, m.err = true, "Grabbing the latest work...", nil
		return m, grab()
	}

	var cmd tea.Cmd
	m.detail, cmd = m.detail.Update(msg)
	return m, cmd
}

// activate does the obvious thing with whatever the cursor is on.
func (m Model) activate() (tea.Model, tea.Cmd) {
	switch m.focus {
	case paneBranches:
		b, ok := m.selectedBranch()
		if !ok || b.Current {
			return m, nil
		}
		if b.Remote {
			// Checking out a remote-tracking ref detaches HEAD, which is a
			// confusing state to strand someone in. Say so instead.
			m.err = errors.New("that line of work lives online — grab it first")
			return m, nil
		}
		m.busy, m.note, m.err = true, "Switching...", nil
		return m, switchBranch(b.Name)
	default:
		// Changes and history already show their detail on cursor move.
		return m, m.refreshDetail()
	}
}

// refreshDetail asks for the preview matching the current selection.
func (m Model) refreshDetail() tea.Cmd {
	switch m.focus {
	case paneChanges:
		if c, ok := m.selectedChange(); ok {
			return loadFileDiff(c.Path)
		}
	case paneHistory:
		if s, ok := m.selectedSavedPoint(); ok {
			return loadCommitDiff(s.Hash)
		}
	case paneBranches:
		return func() tea.Msg {
			return detailMsg{content: branchDetail(m.branches, m.cursor[paneBranches])}
		}
	}
	return func() tea.Msg { return detailMsg{content: ""} }
}

func (m *Model) clampCursors() {
	for p := pane(0); p < paneCount; p++ {
		if n := m.paneLength(p); m.cursor[p] >= n {
			m.cursor[p] = max(0, n-1)
		}
	}
}

func (m Model) paneLength(p pane) int {
	switch p {
	case paneChanges:
		return len(m.status.Changes)
	case paneBranches:
		return len(m.branches)
	default:
		return len(m.history)
	}
}

func (m Model) selectedChange() (ops.Change, bool) {
	i := m.cursor[paneChanges]
	if i < 0 || i >= len(m.status.Changes) {
		return ops.Change{}, false
	}
	return m.status.Changes[i], true
}

func (m Model) selectedBranch() (ops.Branch, bool) {
	i := m.cursor[paneBranches]
	if i < 0 || i >= len(m.branches) {
		return ops.Branch{}, false
	}
	return m.branches[i], true
}

func (m Model) selectedSavedPoint() (ops.SavedPoint, bool) {
	i := m.cursor[paneHistory]
	if i < 0 || i >= len(m.history) {
		return ops.SavedPoint{}, false
	}
	return m.history[i], true
}
