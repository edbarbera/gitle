package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/edbarbera/gitle/internal/ops"
)

// saveForm is the "which files, and what shall we call it?" prompt, embedded
// in the dashboard rather than run on its own.
//
// huh's Form.Run starts its own Bubble Tea program, which can't be done while
// one is already running, so the form is driven as a sub-model instead: the
// dashboard forwards messages to it and renders its view until it reports
// itself finished.
type saveForm struct {
	form    *huh.Form
	paths   []string // every candidate path, in the order shown
	picked  []string
	message string
	risks   ops.Risks
}

// startSave opens the save prompt, if there's anything to save.
func (m Model) startSave() (tea.Model, tea.Cmd) {
	if len(m.status.Changes) == 0 {
		m.note = "Nothing to save — your work is already up to date."
		m.err = nil
		return m, nil
	}

	f := newSaveForm(m.status.Changes)
	m.form = f
	return m, f.form.Init()
}

func newSaveForm(changes []ops.Change) *saveForm {
	sf := &saveForm{paths: ops.Paths(changes)}

	options := make([]huh.Option[string], len(changes))
	for i, c := range changes {
		// Everything starts ticked, matching the CLI: saving all your work is
		// the common case, and unticking a file is less effort than ticking
		// the rest.
		label := mark(c.Kind) + " " + c.Path
		options[i] = huh.NewOption(label, c.Path).Selected(true)
	}
	sf.picked = append([]string(nil), sf.paths...)

	sf.form = huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Which changes do you want to save?").
				Options(options...).
				Value(&sf.picked),
		),
		huh.NewGroup(
			huh.NewText().
				Title("Describe what you changed").
				Lines(3).
				Value(&sf.message).
				Validate(func(s string) error {
					if s == "" {
						return errEmptyMessage
					}
					return nil
				}),
		),
	).
		WithTheme(huh.ThemeFunc(huh.ThemeCharm)).
		WithShowHelp(true)

	return sf
}

var errEmptyMessage = errNoMessage{}

type errNoMessage struct{}

func (errNoMessage) Error() string { return "a short description is needed to save" }

// updateForm drives the embedded form and acts on the result once it closes.
func (m Model) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	form, cmd := m.form.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form.form = f
	}

	switch m.form.form.State {
	case huh.StateAborted:
		m.form = nil
		m.note = "Nothing was saved."
		return m, nil

	case huh.StateCompleted:
		picked, message := m.form.picked, m.form.message
		m.form = nil

		if len(picked) == 0 {
			m.note = "Nothing selected — nothing was saved."
			return m, nil
		}
		// The same safety rail the CLI applies: refuse silently-risky saves.
		// There's no room to negotiate mid-frame, so flag it and let the user
		// decide what to do next rather than committing a secret for them.
		if risks := ops.ScanRisks(picked); risks.Any() {
			m.err = riskError(risks)
			return m, nil
		}

		m.busy, m.note, m.err = true, "Saving...", nil
		return m, save(message, picked)
	}

	return m, cmd
}

// riskError explains why a save was held back.
func riskError(r ops.Risks) error {
	switch {
	case len(r.Secrets) > 0:
		return &heldBack{"that looks like a private file (" + r.Secrets[0] + ") — save it from the terminal if you're sure"}
	default:
		return &heldBack{"that file is very large (" + r.Large[0].Path + ", " + r.Large[0].Size + ") — save it from the terminal if you're sure"}
	}
}

type heldBack struct{ msg string }

func (h *heldBack) Error() string { return h.msg }

// view renders the form centred on an otherwise empty screen.
func (sf *saveForm) view(width, height int) string {
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, sf.form.View())
}
