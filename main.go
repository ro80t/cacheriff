package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"cacheriff/internal/ui"
)

func main() {
	p := tea.NewProgram(ui.NewModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error running program:", err)
		os.Exit(1)
	}
}
