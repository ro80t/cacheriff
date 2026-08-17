package ui

import (
	"github.com/charmbracelet/lipgloss"

	"cacheriff/internal/theme"
)

var (
	colorPrimary   lipgloss.Color
	colorMuted     lipgloss.Color
	colorFaint     lipgloss.Color
	colorFocused   lipgloss.Color
	colorUnfocused lipgloss.Color
	colorError     lipgloss.Color
	colorSuccess   lipgloss.Color

	headerStyle          lipgloss.Style
	headerMetaStyle      lipgloss.Style
	panelTitleStyle      lipgloss.Style
	panelStyle           lipgloss.Style // shared base for bordered panels; callers set Width/Height/BorderForeground
	itemStyle            lipgloss.Style
	selectedItemStyle    lipgloss.Style
	unavailableItemStyle lipgloss.Style
	statusBarStyle       lipgloss.Style
	errorTextStyle       lipgloss.Style
	sectionTitleStyle    lipgloss.Style
)

func init() {
	SetTheme(theme.Default)
}

// SetTheme rebuilds every style in the package from t. Call it once,
// before the program starts, to apply a user-configured color scheme
// (see internal/config).
func SetTheme(t theme.Theme) {
	colorPrimary = lipgloss.Color(t.Primary)
	colorMuted = lipgloss.Color(t.Muted)
	colorFaint = lipgloss.Color(t.Faint)
	colorFocused = lipgloss.Color(t.ActiveBorder)
	colorUnfocused = lipgloss.Color(t.InactiveBorder)
	colorError = lipgloss.Color(t.Error)
	colorSuccess = lipgloss.Color(t.Success)

	headerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorPrimary).
		Padding(0, 1)

	headerMetaStyle = lipgloss.NewStyle().
		Foreground(colorFaint)

	panelTitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorPrimary)

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
}

func borderColor(focused bool) lipgloss.Color {
	if focused {
		return colorFocused
	}
	return colorUnfocused
}
