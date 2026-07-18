package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/edbarbera/gitle/internal/ops"
)

// The dashboard is driven here as a plain state machine: feed it the messages
// Bubble Tea would deliver and inspect what it renders. That needs no
// terminal, so the tests are deterministic and run anywhere.

// plain renders the model and strips styling, so assertions match on words
// rather than escape codes.
func plain(m tea.Model) string {
	return ansi.Strip(m.View().Content)
}

// press sends a keypress the way the runtime would. Special keys carry a
// named code and no text, which is what key.Matches compares against.
func press(t *testing.T, m tea.Model, k string) tea.Model {
	t.Helper()
	var msg tea.KeyPressMsg
	switch k {
	case "tab":
		msg = tea.KeyPressMsg{Code: tea.KeyTab}
	case "enter":
		msg = tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		msg = tea.KeyPressMsg{Code: tea.KeyEscape}
	default:
		msg = tea.KeyPressMsg{Code: rune(k[0]), Text: k}
	}
	next, _ := m.Update(msg)
	return next
}

// ready returns a sized model loaded with the given repo state.
func ready(t *testing.T, msg loadedMsg) tea.Model {
	t.Helper()
	var m tea.Model = New()
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(msg)
	return m
}

// sample is a repo mid-flight: unsaved work, two branches, some history.
func sample() loadedMsg {
	return loadedMsg{
		status: ops.Status{
			Name:        "myproject",
			Branch:      "main",
			MainBranch:  "main",
			HasCommits:  true,
			HasRemote:   true,
			HasUpstream: true,
			VsUpstream:  &ops.AheadBehind{Ahead: 2},
			Changes: []ops.Change{
				{Path: "README.md", Kind: ops.ChangeModified},
				{Path: "src/new.go", Kind: ops.ChangeNew},
				{Path: "old.txt", Kind: ops.ChangeRemoved},
			},
		},
		branches: []ops.Branch{
			{Name: "main", Current: true},
			{Name: "feature-login"},
			{Name: "remotes/origin/main", Remote: true},
		},
		history: []ops.SavedPoint{
			{Hash: "abc1234", Subject: "add search", Author: "Tester", When: "2 hours ago"},
			{Hash: "def5678", Subject: "first version", Author: "Tester", When: "a day ago"},
		},
	}
}

func TestRendersRepoState(t *testing.T) {
	view := plain(ready(t, sample()))

	for _, want := range []string{
		"myproject",     // header: project name
		"main",          // header: branch
		"2 to send",     // header: ahead of upstream
		"Changes (3)",   // pane titles carry counts
		"README.md",     // a changed file
		"src/new.go",    // a new file
		"Lines of work", // branch pane
		"feature-login",
		"History",
		"add search",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("view is missing %q\n---\n%s", want, view)
		}
	}
}

func TestEmptyRepoRendersGuidance(t *testing.T) {
	view := plain(ready(t, loadedMsg{
		status: ops.Status{Name: "fresh", Branch: "main"},
	}))

	if !strings.Contains(view, "Nothing changed since your") {
		t.Errorf("expected the empty-changes message\n---\n%s", view)
	}
	if !strings.Contains(view, "Nothing saved yet.") {
		t.Errorf("expected the empty-history message\n---\n%s", view)
	}
	if !strings.Contains(view, "not online yet") {
		t.Errorf("expected the header to say the repo isn't online\n---\n%s", view)
	}
}

func TestFooterHintFollowsState(t *testing.T) {
	cases := []struct {
		name string
		msg  loadedMsg
		want string
	}{
		{
			name: "unsaved changes",
			msg: loadedMsg{status: ops.Status{
				Changes: []ops.Change{{Path: "a.txt", Kind: ops.ChangeModified}},
			}},
			want: "Press s to save",
		},
		{
			name: "work to send",
			msg: loadedMsg{status: ops.Status{
				HasUpstream: true,
				VsUpstream:  &ops.AheadBehind{Ahead: 1},
			}},
			want: "Press p to send",
		},
		{
			name: "work to grab",
			msg: loadedMsg{status: ops.Status{
				HasUpstream: true,
				VsUpstream:  &ops.AheadBehind{Behind: 3},
			}},
			want: "Press g to grab",
		},
		{
			name: "all clear",
			msg: loadedMsg{status: ops.Status{
				HasUpstream: true,
				VsUpstream:  &ops.AheadBehind{},
			}},
			want: "Everything is saved and up to date",
		},
		{
			name: "conflicts win over everything",
			msg: loadedMsg{status: ops.Status{
				Op:      1, // any in-progress operation
				Changes: []ops.Change{{Path: "a.txt", Kind: ops.ChangeModified}},
			}},
			want: "Conflicts need fixing",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if view := plain(ready(t, tc.msg)); !strings.Contains(view, tc.want) {
				t.Errorf("footer should say %q\n---\n%s", tc.want, view)
			}
		})
	}
}

func TestFocusCyclesBetweenPanes(t *testing.T) {
	m := ready(t, sample())

	if got := m.(Model).focus; got != paneChanges {
		t.Errorf("should start on the changes pane, got %v", got)
	}
	m = press(t, m, "tab")
	if got := m.(Model).focus; got != paneBranches {
		t.Errorf("tab should move to branches, got %v", got)
	}
	m = press(t, m, "tab")
	if got := m.(Model).focus; got != paneHistory {
		t.Errorf("tab should move to history, got %v", got)
	}
	// Wrapping round is what makes tab-only navigation workable.
	m = press(t, m, "tab")
	if got := m.(Model).focus; got != paneChanges {
		t.Errorf("tab should wrap back to changes, got %v", got)
	}
}

func TestCursorStopsAtListEnds(t *testing.T) {
	m := ready(t, sample())

	// Up at the top must not run off into negative indices.
	m = press(t, m, "k")
	if got := m.(Model).cursor[paneChanges]; got != 0 {
		t.Errorf("cursor went above the first row: %d", got)
	}

	// Down past the last row must stop on it. sample() has 3 changes.
	for range 10 {
		m = press(t, m, "j")
	}
	if got := m.(Model).cursor[paneChanges]; got != 2 {
		t.Errorf("cursor should stop on the last of 3 changes, got %d", got)
	}
}

// TestCursorSurvivesShrinkingList covers the refresh-after-save case: the
// cursor sat on row 2, then saving emptied the list.
func TestCursorSurvivesShrinkingList(t *testing.T) {
	m := ready(t, sample())
	m = press(t, m, "j")
	m = press(t, m, "j")
	if got := m.(Model).cursor[paneChanges]; got != 2 {
		t.Fatalf("setup: cursor should be on row 2, got %d", got)
	}

	// Everything got saved; the changes list is now empty.
	m, _ = m.Update(loadedMsg{status: ops.Status{Name: "myproject", HasCommits: true}})

	if got := m.(Model).cursor[paneChanges]; got != 0 {
		t.Errorf("cursor should have been pulled back to 0, got %d", got)
	}
	// Rendering must not panic or index past the end.
	if view := plain(m); !strings.Contains(view, "Nothing changed") {
		t.Errorf("expected the empty state\n---\n%s", view)
	}
}

func TestQuitKeys(t *testing.T) {
	for _, k := range []string{"q", "esc", "ctrl+c"} {
		t.Run(k, func(t *testing.T) {
			m := ready(t, sample())
			var msg tea.KeyPressMsg
			switch k {
			case "q":
				msg = tea.KeyPressMsg{Code: 'q', Text: "q"}
			case "esc":
				msg = tea.KeyPressMsg{Code: tea.KeyEscape}
			case "ctrl+c":
				msg = tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
			}
			_, cmd := m.Update(msg)
			if cmd == nil {
				t.Fatalf("%s should produce a command", k)
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Errorf("%s should quit, got %T", k, cmd())
			}
		})
	}
}

func TestSaveWithNothingToSaveDoesNotOpenForm(t *testing.T) {
	m := ready(t, loadedMsg{status: ops.Status{Name: "clean", HasCommits: true}})
	m = press(t, m, "s")

	if m.(Model).form != nil {
		t.Errorf("form should not open when there's nothing to save")
	}
	if view := plain(m); !strings.Contains(view, "Nothing to save") {
		t.Errorf("expected a note explaining why\n---\n%s", view)
	}
}

func TestSaveOpensFormAndTakesOverScreen(t *testing.T) {
	m := ready(t, sample())
	m = press(t, m, "s")

	if m.(Model).form == nil {
		t.Fatal("pressing s should open the save form")
	}
	view := plain(m)
	if !strings.Contains(view, "Which changes do you want to save?") {
		t.Errorf("form should ask which files\n---\n%s", view)
	}
	// While the form is up the dashboard behind it must not also be drawn.
	if strings.Contains(view, "Lines of work") {
		t.Errorf("dashboard should be hidden behind the form\n---\n%s", view)
	}
}

// TestFormSwallowsDashboardKeys guards the trap where a keystroke meant for a
// text field also triggers a dashboard action — typing "s" in a commit
// message must not kick off a second save.
func TestFormSwallowsDashboardKeys(t *testing.T) {
	m := ready(t, sample())
	m = press(t, m, "s")
	before := m.(Model).form

	m = press(t, m, "g") // would be "grab" on the dashboard
	if m.(Model).busy {
		t.Errorf("a keypress inside the form triggered a dashboard action")
	}
	if m.(Model).form != before {
		t.Errorf("form should still be open")
	}
}

func TestErrorsAreShownOnOneLine(t *testing.T) {
	m := ready(t, sample())
	m, _ = m.Update(doneMsg{err: errMultiline{}})

	view := plain(m)
	if !strings.Contains(view, "first line of trouble") {
		t.Errorf("expected the error in the footer\n---\n%s", view)
	}
	// git errors run to a dozen lines; the footer has room for one.
	if strings.Contains(view, "second line") {
		t.Errorf("footer should show only the first line\n---\n%s", view)
	}
}

type errMultiline struct{}

func (errMultiline) Error() string { return "first line of trouble\nsecond line\nthird line" }

func TestSmallWindowExplainsItself(t *testing.T) {
	var m tea.Model = New()
	m, _ = m.Update(tea.WindowSizeMsg{Width: 30, Height: 8})
	m, _ = m.Update(sample())

	if view := plain(m); !strings.Contains(view, "too small") {
		t.Errorf("a cramped window should say so rather than render garbage\n---\n%s", view)
	}
}

func TestRemoteBranchCannotBeSwitchedTo(t *testing.T) {
	m := ready(t, sample())
	m = press(t, m, "tab") // focus branches
	// Move to the remote branch, which is third in the list.
	m = press(t, m, "j")
	m = press(t, m, "j")

	b, ok := m.(Model).selectedBranch()
	if !ok || !b.Remote {
		t.Fatalf("setup: expected to be on the remote branch, got %+v", b)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.(Model).busy {
		t.Errorf("should not attempt to switch to a remote-tracking branch")
	}
	if view := plain(m); !strings.Contains(view, "lives online") {
		t.Errorf("expected an explanation\n---\n%s", view)
	}
}

// TestColumnsAlign guards the layout arithmetic. The left stack and the detail
// pane are built independently, so it's easy to make one a few rows taller
// than the other — which looks broken at a glance.
func TestColumnsAlign(t *testing.T) {
	for _, size := range []struct{ w, h int }{
		{120, 40}, {80, 24}, {200, 60}, {60, 16}, {100, 31},
	} {
		var m tea.Model = New()
		m, _ = m.Update(tea.WindowSizeMsg{Width: size.w, Height: size.h})
		m, _ = m.Update(sample())

		dash := m.(Model)
		left := lipgloss.Height(dash.leftColumn())
		right := lipgloss.Height(dash.detailPane())
		if left != right {
			t.Errorf("%dx%d: left column is %d rows, detail pane is %d — they must match",
				size.w, size.h, left, right)
		}

		// The whole frame must also fit the window, or the terminal scrolls
		// and the alt screen jumps.
		if got := lipgloss.Height(dash.render()); got > size.h {
			t.Errorf("%dx%d: rendered %d rows, which overflows the window", size.w, size.h, got)
		}
	}
}
