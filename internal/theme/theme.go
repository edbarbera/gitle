// Package theme is the single source of colour and style for gitle. Every
// styled string in the CLI and the TUI comes from here, so changing the look
// means editing one file.
package theme

import (
	"charm.land/lipgloss/v2"
)

// Palette. These are the basic ANSI colours gitle has always used, kept as
// numbers rather than hex so they inherit whatever the user's terminal theme
// defines — a green that looks right in Solarized and in the default macOS
// profile alike.
var (
	Green  = lipgloss.Color("2")
	Red    = lipgloss.Color("1")
	Yellow = lipgloss.Color("3")
	Cyan   = lipgloss.Color("6")
	Grey   = lipgloss.Color("8")
)

// Line styles used by the plain-text printers in internal/ui.
var (
	Success = lipgloss.NewStyle().Foreground(Green)
	Info    = lipgloss.NewStyle().Foreground(Cyan)
	Warn    = lipgloss.NewStyle().Foreground(Yellow)
	Error   = lipgloss.NewStyle().Foreground(Red)
	Hint    = lipgloss.NewStyle().Faint(true)
	Bold    = lipgloss.NewStyle().Bold(true)
	Ask     = lipgloss.NewStyle().Foreground(Yellow)
)

// Colour-only styles, exposed so callers can tint arbitrary fragments.
var (
	GreenText  = lipgloss.NewStyle().Foreground(Green)
	YellowText = lipgloss.NewStyle().Foreground(Yellow)
	RedText    = lipgloss.NewStyle().Foreground(Red)
	CyanText   = lipgloss.NewStyle().Foreground(Cyan)
	DimText    = lipgloss.NewStyle().Faint(true)
)

// Dashboard styles.
var (
	// PaneActive and PaneInactive border the focused and unfocused panes. The
	// border is always drawn so pane widths don't shift when focus moves;
	// only its colour changes.
	PaneActive = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Cyan).
			Padding(0, 1)

	PaneInactive = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Grey).
			Padding(0, 1)

	// PaneTitle heads each pane; the active one is brightened by the model.
	PaneTitle       = lipgloss.NewStyle().Bold(true).Foreground(Grey)
	PaneTitleActive = lipgloss.NewStyle().Bold(true).Foreground(Cyan)

	// Header is the top bar: repo name, branch, ahead/behind.
	Header = lipgloss.NewStyle().Bold(true).Foreground(Cyan)

	// SelectedRow marks the cursor line inside a pane list.
	SelectedRow = lipgloss.NewStyle().Foreground(Cyan).Bold(true)

	// StatusBar is the footer strip of key hints.
	StatusBar = lipgloss.NewStyle().Faint(true)

	// Banner styles the ASCII art on `gitle start`.
	Banner = lipgloss.NewStyle().Foreground(Cyan)
)
