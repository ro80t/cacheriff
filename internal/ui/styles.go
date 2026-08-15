package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorPrimary   = lipgloss.Color("205")
	colorMuted     = lipgloss.Color("240")
	colorFaint     = lipgloss.Color("244")
	colorFocused   = lipgloss.Color("212")
	colorUnfocused = lipgloss.Color("240")
	colorError     = lipgloss.Color("204")
	colorSuccess   = lipgloss.Color("114")

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			Padding(0, 1)

	headerMetaStyle = lipgloss.NewStyle().
			Foreground(colorFaint)

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	// panelStyle is the shared base for bordered panels; callers set
	// Width/Height and swap the border color based on focus.
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1)

	itemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	selectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPrimary).
				PaddingLeft(0)

	unavailableItemStyle = lipgloss.NewStyle().
				Foreground(colorMuted)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(colorFaint)

	errorTextStyle = lipgloss.NewStyle().
			Foreground(colorError)

	sectionTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Underline(true)
)

func borderColor(focused bool) lipgloss.Color {
	if focused {
		return colorFocused
	}
	return colorUnfocused
}
