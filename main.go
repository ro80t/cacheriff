package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"cacheriff/internal/config"
	"cacheriff/internal/ui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "warning: failed to load config, using defaults:", err)
	}
	ui.SetTheme(cfg.Theme())

	p := tea.NewProgram(ui.NewModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error running program:", err)
		os.Exit(1)
	}
}
