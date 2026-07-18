package tui

import "charm.land/bubbles/v2/key"

// keyMap declares every binding once, so the help overlay stays in step with
// what the dashboard actually does.
type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Next    key.Binding
	Prev    key.Binding
	Enter   key.Binding
	Save    key.Binding
	Send    key.Binding
	Grab    key.Binding
	Refresh key.Binding
	Help    key.Binding
	Quit    key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Next: key.NewBinding(
		key.WithKeys("tab", "right", "l"),
		key.WithHelp("tab", "next panel"),
	),
	Prev: key.NewBinding(
		key.WithKeys("shift+tab", "left", "h"),
		key.WithHelp("shift+tab", "previous panel"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "open/switch"),
	),
	Save: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "save"),
	),
	Send: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "send online"),
	),
	Grab: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "grab latest"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c", "esc"),
		key.WithHelp("q", "quit"),
	),
}

// ShortHelp is the single footer line.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Save, k.Send, k.Grab, k.Next, k.Refresh, k.Help, k.Quit}
}

// FullHelp is the expanded overlay, grouped into columns.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Next, k.Prev},
		{k.Enter, k.Save, k.Send, k.Grab},
		{k.Refresh, k.Help, k.Quit},
	}
}
