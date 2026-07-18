package tui

import (
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/edbarbera/gitle/internal/ops"
	"github.com/edbarbera/gitle/internal/theme"
)

// Layout constants. The left column holds the three lists; the rest goes to
// the detail preview.
const (
	minWidth     = 60
	minHeight    = 16
	leftColWidth = 34
	headerHeight = 1
	footerHeight = 1

	// paneChrome is the rows a pane spends on itself before any content: two
	// border lines plus its title. Getting this wrong makes the left column
	// and the detail pane disagree about where the bottom of the screen is.
	paneChrome = 3
)

// layout recomputes the sizes that depend on the terminal, and is called
// whenever the window changes or the help overlay is toggled.
func (m *Model) layout() {
	if m.width < minWidth || m.height < minHeight {
		return
	}
	detailWidth := m.width - leftColWidth - 4
	m.detail.SetWidth(max(20, detailWidth))
	m.detail.SetHeight(max(3, m.bodyHeight()-paneChrome))
}

// bodyHeight is the number of rows the panes get, once the header, footer and
// help line have taken theirs.
func (m Model) bodyHeight() int {
	helpHeight := lipgloss.Height(m.help.View(keys))
	return m.height - headerHeight - footerHeight - helpHeight
}

func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	// The alt screen keeps the dashboard out of the scrollback, so quitting
	// leaves the terminal exactly as it was found.
	v.AltScreen = true
	return v
}

func (m Model) render() string {
	if !m.ready {
		return "Loading..."
	}
	if m.width < minWidth || m.height < minHeight {
		return "This window is a little too small for the gitle dashboard.\n" +
			"Make it bigger, or press q to leave."
	}

	// A form takes over the screen entirely: half a dashboard behind a prompt
	// is harder to read than the prompt on its own.
	if m.form != nil {
		return m.form.view(m.width, m.height)
	}

	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.leftColumn(),
		m.detailPane(),
	)

	return strings.Join([]string{
		m.header(),
		body,
		m.footer(),
		m.help.View(keys),
	}, "\n")
}

// header is the top bar: what project, which line of work, how it compares.
func (m Model) header() string {
	parts := []string{"📦 " + theme.Header.Render(orDash(m.status.Name))}
	if m.status.Branch != "" {
		parts = append(parts, theme.DimText.Render("on")+" "+theme.CyanText.Render(m.status.Branch))
	}
	if ab := m.status.VsUpstream; ab != nil && !ab.InSync() {
		var bits []string
		if ab.Ahead > 0 {
			bits = append(bits, theme.GreenText.Render(strconv.Itoa(ab.Ahead)+" to send"))
		}
		if ab.Behind > 0 {
			bits = append(bits, theme.YellowText.Render(strconv.Itoa(ab.Behind)+" to grab"))
		}
		parts = append(parts, strings.Join(bits, theme.DimText.Render(", ")))
	} else if !m.status.HasUpstream {
		parts = append(parts, theme.DimText.Render("not online yet"))
	}
	if m.status.Conflicted() {
		parts = append(parts, theme.RedText.Render("conflicts to fix"))
	}
	return strings.Join(parts, theme.DimText.Render(" · "))
}

// footer shows the spinner while busy, then whatever happened last.
func (m Model) footer() string {
	switch {
	case m.err != nil:
		return theme.Error.Render("✗ " + ops.FirstLine(m.err.Error()))
	case m.busy:
		return m.spin.View() + " " + theme.DimText.Render(m.note)
	case m.note != "":
		return theme.Success.Render("✓ " + m.note)
	default:
		return theme.DimText.Render(m.hint())
	}
}

// hint nudges towards the most useful next step, the way the CLI does.
func (m Model) hint() string {
	switch {
	case m.status.Conflicted():
		return "Conflicts need fixing — run gitle fix-conflicts."
	case len(m.status.Changes) > 0:
		return "Press s to save your changes."
	case m.status.VsUpstream != nil && m.status.VsUpstream.Ahead > 0:
		return "Press p to send your saved work online."
	case m.status.VsUpstream != nil && m.status.VsUpstream.Behind > 0:
		return "Press g to grab the latest work."
	default:
		return "Everything is saved and up to date."
	}
}

// leftColumn stacks the three lists, sharing the available height between them.
func (m Model) leftColumn() string {
	// Each pane's own chrome comes off the top first; what's left is content
	// rows to divide up. Splitting before subtracting would make the column
	// three title-rows taller than the detail pane beside it.
	rows := m.bodyHeight() - int(paneCount)*paneChrome

	// Changes gets the most room: it's what people look at most often. The
	// last pane takes the remainder so rounding never loses a row.
	heights := [paneCount]int{
		rows / 2,
		rows / 4,
		rows - rows/2 - rows/4,
	}

	var rendered []string
	for p := pane(0); p < paneCount; p++ {
		rendered = append(rendered, m.renderPane(p, max(1, heights[p])))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rendered...)
}

// renderPane draws one list, scrolled so the cursor stays visible.
func (m Model) renderPane(p pane, height int) string {
	rows := m.paneRows(p)
	cursor := m.cursor[p]
	focused := m.focus == p

	title := p.title()
	if n := len(rows); n > 0 {
		title += " (" + strconv.Itoa(n) + ")"
	}
	titleStyle := theme.PaneTitle
	boxStyle := theme.PaneInactive
	if focused {
		titleStyle = theme.PaneTitleActive
		boxStyle = theme.PaneActive
	}

	visible := make([]string, 0, height)
	if len(rows) == 0 {
		visible = append(visible, theme.DimText.Render(emptyMessage(p)))
	} else {
		// Scroll the window so the cursor is always on screen, keeping it
		// roughly centred once the list is longer than the pane.
		start := 0
		if cursor >= height {
			start = cursor - height + 1
		}
		for i := start; i < len(rows) && len(visible) < height; i++ {
			line := truncate(rows[i], leftColWidth-4)
			if i == cursor && focused {
				line = theme.SelectedRow.Render("❯ " + line)
			} else {
				line = "  " + line
			}
			visible = append(visible, line)
		}
	}

	// Pad to the full height so the panes below don't shift as lists change.
	for len(visible) < height {
		visible = append(visible, "")
	}

	content := titleStyle.Render(title) + "\n" + strings.Join(visible, "\n")
	return boxStyle.Width(leftColWidth).Render(content)
}

func (m Model) paneRows(p pane) []string {
	switch p {
	case paneChanges:
		rows := make([]string, len(m.status.Changes))
		for i, c := range m.status.Changes {
			rows[i] = changeStyle(c.Kind).Render(mark(c.Kind)) + " " + c.Path
		}
		return rows

	case paneBranches:
		rows := make([]string, len(m.branches))
		for i, b := range m.branches {
			name := strings.TrimPrefix(b.Name, "remotes/")
			switch {
			case b.Current:
				rows[i] = theme.GreenText.Render("● ") + name
			case b.Remote:
				rows[i] = theme.DimText.Render("☁ " + name)
			default:
				rows[i] = "  " + name
			}
		}
		return rows

	default:
		rows := make([]string, len(m.history))
		for i, s := range m.history {
			rows[i] = theme.DimText.Render(s.Hash) + " " + s.Subject
		}
		return rows
	}
}

func emptyMessage(p pane) string {
	switch p {
	case paneChanges:
		return "Nothing changed since your last save."
	case paneBranches:
		return "No lines of work yet."
	default:
		return "Nothing saved yet."
	}
}

// detailPane is the preview on the right.
func (m Model) detailPane() string {
	title := theme.PaneTitle.Render("Details")
	body := m.detail.View()
	if strings.TrimSpace(body) == "" {
		body = theme.DimText.Render("Nothing to preview.")
	}

	// Height here is the pane's total rendered height, border included, so
	// it matches the stacked column beside it exactly.
	return theme.PaneInactive.
		Width(m.width - leftColWidth - 4).
		Height(max(1, m.bodyHeight())).
		Render(title + "\n" + body)
}

// branchDetail describes the highlighted line of work in words, since there's
// no diff to show for one.
func branchDetail(branches []ops.Branch, cursor int) string {
	if cursor < 0 || cursor >= len(branches) {
		return ""
	}
	b := branches[cursor]
	name := strings.TrimPrefix(b.Name, "remotes/")

	var sb strings.Builder
	sb.WriteString(theme.Bold.Render(name) + "\n\n")
	switch {
	case b.Current:
		sb.WriteString("This is where you are now.\n")
	case b.Remote:
		sb.WriteString("This line of work lives online.\n")
		sb.WriteString(theme.DimText.Render("Grab it before you can work on it here.\n"))
	default:
		sb.WriteString("A separate line of work on this computer.\n")
		sb.WriteString(theme.DimText.Render("Press enter to switch to it.\n"))
	}
	return sb.String()
}

func changeStyle(k ops.ChangeKind) lipgloss.Style {
	switch k {
	case ops.ChangeNew:
		return theme.GreenText
	case ops.ChangeRemoved:
		return theme.RedText
	default:
		return theme.YellowText
	}
}

// mark is the single character standing in for a kind of change.
func mark(k ops.ChangeKind) string {
	switch k {
	case ops.ChangeNew:
		return "+"
	case ops.ChangeRemoved:
		return "-"
	default:
		return "~"
	}
}

// truncate shortens s to width display cells, ending in an ellipsis. Long file
// paths are cut from the left, since the file name matters more than the
// folders above it.
func truncate(s string, width int) string {
	if width <= 1 || lipgloss.Width(s) <= width {
		return s
	}
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	return "…" + string(runes[len(runes)-width+1:])
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
