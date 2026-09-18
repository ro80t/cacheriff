package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/ro80t/cacheriff/internal/config"
	"github.com/ro80t/cacheriff/internal/ui"
)

// version is overridable at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	root := &cobra.Command{
		Use:   "cacheriff",
		Short: "A lazygit-inspired TUI for cleaning up package manager caches",
		Long: "cacheriff is a lazygit-inspired terminal UI for finding and cleaning up\n" +
			"package manager caches, globally installed packages, and per-project\n" +
			"install artifacts across npm, pnpm, yarn, bun, deno, cargo, and Go.",
		Version:      version,
		SilenceUsage: true,
		RunE:         run,
	}

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "warning: failed to load config, using defaults:", err)
	}
	ui.SetTheme(cfg.Theme())

	p := tea.NewProgram(ui.NewModel(), tea.WithAltScreen())
	_, err = p.Run()
	return err
}
