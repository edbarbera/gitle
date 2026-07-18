package tui

import (
	"strconv"

	tea "charm.land/bubbletea/v2"

	"github.com/edbarbera/gitle/internal/ops"
)

// Every git call happens inside a tea.Cmd and reports back as one of these
// messages. Nothing in this file may run on the update loop directly: a push
// over a slow link would freeze the whole dashboard.

// loadedMsg carries a full refresh of everything the panes show.
type loadedMsg struct {
	status   ops.Status
	branches []ops.Branch
	history  []ops.SavedPoint
	err      error
}

// detailMsg is the content for the preview pane.
type detailMsg struct {
	content string
	err     error
}

// doneMsg reports the outcome of an action the user triggered.
type doneMsg struct {
	// note is shown in the footer on success.
	note string
	err  error
}

// load gathers everything the dashboard displays in one pass.
func load() tea.Cmd {
	return func() tea.Msg {
		status, err := ops.CurrentStatus()
		if err != nil {
			return loadedMsg{err: err}
		}
		branches, err := ops.Branches()
		if err != nil {
			return loadedMsg{err: err}
		}
		// The history pane only ever shows a screenful; asking for the whole
		// log on every refresh would be slow in a long-lived project.
		history, err := ops.History(100)
		if err != nil {
			return loadedMsg{err: err}
		}
		return loadedMsg{status: status, branches: branches, history: history}
	}
}

// loadFileDiff fetches the unsaved changes in one file.
func loadFileDiff(path string) tea.Cmd {
	return func() tea.Msg {
		diff, err := ops.FileDiff(path)
		return detailMsg{content: diff, err: err}
	}
}

// loadCommitDiff fetches what one saved point changed.
func loadCommitDiff(hash string) tea.Cmd {
	return func() tea.Msg {
		diff, err := ops.CommitDiff(hash)
		return detailMsg{content: diff, err: err}
	}
}

// save records the given paths as a saved point.
func save(message string, paths []string) tea.Cmd {
	return func() tea.Msg {
		result, err := ops.Save(message, paths)
		if err != nil {
			return doneMsg{err: err}
		}
		note := "Saved " + plural(len(result.Paths), "file", "files") + "."
		if result.Leftover {
			note += " Some changes are still unsaved."
		}
		return doneMsg{note: note}
	}
}

// send uploads saved work.
//
// AllowTerminalPrompt is false without exception here: git would write its
// credential prompt straight to the terminal we're drawing on.
func send() tea.Cmd {
	return func() tea.Msg {
		result, err := ops.Send(ops.SendOptions{AllowTerminalPrompt: false})
		if err != nil {
			return doneMsg{err: err}
		}
		note := "Sent everything online."
		if result.FirstPush {
			note = "Sent — '" + result.Branch + "' now has an online home."
		}
		return doneMsg{note: note}
	}
}

// grab downloads and blends in everyone else's latest work.
func grab() tea.Cmd {
	return func() tea.Msg {
		if err := ops.Grab(); err != nil {
			return doneMsg{err: err}
		}
		return doneMsg{note: "Up to date with everyone's latest work."}
	}
}

// switchBranch moves onto another line of work.
func switchBranch(name string) tea.Cmd {
	return func() tea.Msg {
		if err := ops.Switch(name); err != nil {
			return doneMsg{err: err}
		}
		return doneMsg{note: "Switched to " + name + "."}
	}
}

func plural(n int, one, many string) string {
	word := many
	if n == 1 {
		word = one
	}
	return strconv.Itoa(n) + " " + word
}
