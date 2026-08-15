package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"cacheriff/internal/platform"
)

// panelLayout holds the pixel budget computed for the current
// terminal size, shared by applyLayout (which sizes the viewport)
// and View (which renders the panels).
type panelLayout struct {
	sidebarOuterWidth int
	mainOuterWidth    int

	sidebarContentWidth  int
	sidebarContentHeight int
	mainContentWidth     int
	mainContentHeight    int
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (m Model) computeLayout() panelLayout {
	footerLines := strings.Count(m.renderFooter(), "\n") + 1
	const headerLines = 1

	bodyHeight := m.height - headerLines - footerLines
	if bodyHeight < 5 {
		bodyHeight = 5
	}

	sidebarOuterWidth := clamp(m.width*3/10, 24, 44)
	if sidebarOuterWidth > m.width-16 {
		sidebarOuterWidth = clamp(m.width-16, 10, sidebarOuterWidth)
	}
	mainOuterWidth := m.width - sidebarOuterWidth
	if mainOuterWidth < 10 {
		mainOuterWidth = 10
	}

	// Each panel spends 2 columns/rows on its border and 2
	// columns on left/right padding (panelStyle uses Padding(0, 1)).
	return panelLayout{
		sidebarOuterWidth:    sidebarOuterWidth,
		mainOuterWidth:       mainOuterWidth,
		sidebarContentWidth:  maxInt(1, sidebarOuterWidth-4),
		sidebarContentHeight: maxInt(1, bodyHeight-2),
		mainContentWidth:     maxInt(1, mainOuterWidth-4),
		mainContentHeight:    maxInt(1, bodyHeight-2),
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// applyLayout resizes the viewport to fit the current terminal size
// and re-renders its content so long lines wrap/scroll correctly.
func (m *Model) applyLayout() {
	if !m.ready {
		return
	}
	l := m.computeLayout()
	// The main panel shows a title line, a blank line, then the
	// scrollable viewport.
	m.viewport.Width = l.mainContentWidth
	m.viewport.Height = maxInt(1, l.mainContentHeight-2)
	m.viewport.SetContent(m.renderMainContent())
}

func (m Model) renderHeader() string {
	left := headerStyle.Render("cacheriff")
	meta := headerMetaStyle.Render(fmt.Sprintf("OS: %s", platform.Current()))

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(meta)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + meta
}

func (m Model) renderFooter() string {
	return statusBarStyle.Render(m.help.View(m.keys))
}

func (m Model) renderSidebar() string {
	var b strings.Builder
	b.WriteString(panelTitleStyle.Render("Package Managers"))
	b.WriteString("\n\n")

	if len(m.items) == 0 {
		b.WriteString(unavailableItemStyle.Render("No supported package managers for this OS."))
		return b.String()
	}

	for i, it := range m.items {
		marker := "  "
		if i == m.cursor {
			marker = "❯ "
		}
		status := "✓"
		if !it.available {
			status = "✗"
		}
		line := fmt.Sprintf("%s%s %s", marker, status, it.driver.Name())

		switch {
		case i == m.activeIdx:
			line = selectedItemStyle.Render(line)
		case !it.available:
			line = unavailableItemStyle.Render(line)
		case i == m.cursor:
			line = lipgloss.NewStyle().Foreground(colorFocused).Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) mainTitle() string {
	item, ok := m.activeItem()
	if !ok {
		return "Details"
	}
	return item.driver.Name()
}

// renderMainContent builds the (unbounded-height) content shown in
// the main panel's viewport for the current selection/load state.
func (m Model) renderMainContent() string {
	item, ok := m.activeItem()
	if !ok {
		return unavailableItemStyle.Render("Select a package manager on the left and press enter.")
	}

	if !item.available {
		return errorTextStyle.Render(fmt.Sprintf(
			"%s was not found on this machine.\nInstall it and restart cacheriff to inspect its caches and packages.",
			item.driver.Name(),
		))
	}

	switch m.state {
	case loadInProgress:
		return fmt.Sprintf("%s Scanning caches and global packages...", m.spinner.View())
	case loadError:
		return errorTextStyle.Render("Error: " + m.loadErr.Error())
	case loadDone:
		return m.renderEntries()
	default:
		return ""
	}
}

func (m Model) renderEntries() string {
	var b strings.Builder

	b.WriteString(sectionTitleStyle.Render(fmt.Sprintf("Caches (%d)", len(m.cache))))
	b.WriteString("\n")
	if len(m.cache) == 0 {
		b.WriteString(unavailableItemStyle.Render("  none found"))
		b.WriteString("\n")
	}
	for _, e := range m.cache {
		b.WriteString(fmt.Sprintf("  %-40s %10s  %s\n", e.Name, formatBytes(e.Size), e.Path))
	}

	b.WriteString("\n")
	b.WriteString(sectionTitleStyle.Render(fmt.Sprintf("Global packages (%d)", len(m.packages))))
	b.WriteString("\n")
	if len(m.packages) == 0 {
		b.WriteString(unavailableItemStyle.Render("  none found"))
		b.WriteString("\n")
	}
	for _, e := range m.packages {
		b.WriteString(fmt.Sprintf("  %-30s v%-14s %10s\n", e.Name, e.Version, formatBytes(e.Size)))
	}

	return b.String()
}

func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}
	l := m.computeLayout()

	sidebar := panelStyle.
		Width(l.sidebarContentWidth).
		Height(l.sidebarContentHeight).
		BorderForeground(borderColor(m.focus == focusSidebar)).
		Render(m.renderSidebar())

	mainBody := lipgloss.JoinVertical(lipgloss.Left,
		panelTitleStyle.Render(m.mainTitle()),
		"",
		m.viewport.View(),
	)
	main := panelStyle.
		Width(l.mainContentWidth).
		Height(l.mainContentHeight).
		BorderForeground(borderColor(m.focus == focusMain)).
		Render(mainBody)

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)

	return lipgloss.JoinVertical(lipgloss.Left, m.renderHeader(), body, m.renderFooter())
}
