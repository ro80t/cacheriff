package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"cacheriff/internal/driver"
	"cacheriff/internal/platform"
	"cacheriff/internal/textwrap"
)

// panelLayout holds the pixel budget computed for the current
// terminal size, shared by applyLayout (which sizes the viewport)
// and View (which renders the panels).
type panelLayout struct {
	sidebarOuterWidth int
	mainOuterWidth    int

	// Passed to panelStyle.Width(), which already accounts for its own
	// padding, so these only need to exclude the border (2 cols).
	sidebarBoxWidth int
	mainBoxWidth    int

	// Usable text columns once both border and padding are excluded.
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

	// panelStyle has no vertical padding, so heights only subtract the
	// border (2); widths subtract border + horizontal padding (4).
	return panelLayout{
		sidebarOuterWidth:    sidebarOuterWidth,
		mainOuterWidth:       mainOuterWidth,
		sidebarBoxWidth:      maxInt(1, sidebarOuterWidth-2),
		mainBoxWidth:         maxInt(1, mainOuterWidth-2),
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

	if len(m.drivers) == 0 {
		b.WriteString(unavailableItemStyle.Render("No supported package managers for this OS."))
		return b.String()
	}

	for i, it := range m.drivers {
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

	switch {
	case m.removing:
		return fmt.Sprintf("%s Uninstalling %s...", m.spinner.View(), m.pendingRemove.Name)
	case m.state == loadInProgress:
		return fmt.Sprintf("%s Scanning caches and packages...", m.spinner.View())
	case m.state == loadError:
		return errorTextStyle.Render("Error: " + m.loadErr.Error())
	case m.state == loadDone:
		return m.renderEntries()
	default:
		return ""
	}
}

// cacheRowLines is shared by renderEntries and packageCursorLineOffset
// so the two stay in lockstep.
func cacheRowLines(e driver.Entry, contentWidth int) []string {
	prefix := fmt.Sprintf("  %-40s %10s  ", e.Name, formatBytes(e.Size))
	return textwrap.ContentLine(prefix+e.Path, contentWidth, lipgloss.Width(prefix))
}

func cacheSectionLineCount(cache []driver.Entry, contentWidth int) int {
	lines := 1 // "Caches (N)" title
	if len(cache) == 0 {
		lines++
	}
	for _, e := range cache {
		lines += len(cacheRowLines(e, contentWidth))
	}
	return lines
}

// packageRowLines is shared by renderEntries and packageCursorLineOffset
// so the two stay in lockstep.
func packageRowLines(e driver.Entry, contentWidth int, selected bool) []string {
	marker := "  "
	if selected {
		marker = "❯ "
	}
	line := fmt.Sprintf("%s%-30s v%-14s %10s", marker, e.Name, e.Version, formatBytes(e.Size))
	return textwrap.ContentLine(line, contentWidth, 4)
}

func (m Model) packageCursorLineOffset(contentWidth int) int {
	lines := cacheSectionLineCount(m.cache, contentWidth)
	lines += 2 // blank line + scope tabs line

	entries := m.currentEntries()
	if len(entries) == 0 {
		lines++
	}
	for i, e := range entries {
		if i == m.packageCursor {
			return lines
		}
		lines += len(packageRowLines(e, contentWidth, false))
	}
	return lines
}

func (m Model) renderEntries() string {
	contentWidth := m.computeLayout().mainContentWidth

	var b strings.Builder

	b.WriteString(sectionTitleStyle.Render(fmt.Sprintf("Caches (%d)", len(m.cache))))
	b.WriteString("\n")
	if len(m.cache) == 0 {
		b.WriteString(unavailableItemStyle.Render("  none found"))
		b.WriteString("\n")
	}
	for _, e := range m.cache {
		for _, chunk := range cacheRowLines(e, contentWidth) {
			b.WriteString(chunk)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(m.renderScopeTabs())
	b.WriteString("\n")

	if m.removeErr != nil {
		b.WriteString(errorTextStyle.Render("  Uninstall failed: " + m.removeErr.Error()))
		b.WriteString("\n")
	}

	entries := m.currentEntries()
	if len(entries) == 0 {
		b.WriteString(unavailableItemStyle.Render("  none found"))
		b.WriteString("\n")
	}
	for i, e := range entries {
		selected := m.scope == scopeGlobal && i == m.packageCursor
		style := lipgloss.NewStyle()
		if selected {
			style = selectedItemStyle
		}
		for _, chunk := range packageRowLines(e, contentWidth, selected) {
			b.WriteString(style.Render(chunk))
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (m Model) renderScopeTabs() string {
	global := fmt.Sprintf("Global packages (%d)", len(m.globalPackages))
	local := fmt.Sprintf("Local packages (%d)", len(m.localPackages))

	if m.scope == scopeGlobal {
		global = selectedItemStyle.Render("[ " + global + " ]")
		local = unavailableItemStyle.Render(local)
	} else {
		global = unavailableItemStyle.Render(global)
		local = selectedItemStyle.Render("[ " + local + " ]")
	}
	return global + "   " + local
}

func (m Model) renderConfirmModal() string {
	e := m.pendingRemove
	title := lipgloss.NewStyle().Bold(true).Foreground(colorError).Render("Uninstall package?")
	detail := fmt.Sprintf("%s v%s", e.Name, e.Version)
	warning := "This runs the package manager's own uninstall\ncommand and cannot be undone."
	hint := unavailableItemStyle.Render("[y] confirm    [n / esc] cancel")

	content := lipgloss.JoinVertical(lipgloss.Left, title, "", detail, "", warning, "", hint)
	return panelStyle.
		BorderForeground(colorError).
		Padding(1, 2).
		Render(content)
}

func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}
	l := m.computeLayout()

	sidebar := panelStyle.
		Width(l.sidebarBoxWidth).
		Height(l.sidebarContentHeight).
		BorderForeground(borderColor(m.focus == focusSidebar)).
		Render(m.renderSidebar())

	mainBody := lipgloss.JoinVertical(lipgloss.Left,
		panelTitleStyle.Render(m.mainTitle()),
		"",
		m.viewport.View(),
	)
	main := panelStyle.
		Width(l.mainBoxWidth).
		Height(l.mainContentHeight).
		BorderForeground(borderColor(m.focus == focusMain)).
		Render(mainBody)

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)

	if m.confirmRemove {
		body = lipgloss.Place(m.width, lipgloss.Height(body), lipgloss.Center, lipgloss.Center, m.renderConfirmModal())
	}

	return lipgloss.JoinVertical(lipgloss.Left, m.renderHeader(), body, m.renderFooter())
}
