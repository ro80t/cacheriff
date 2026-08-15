package ui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"cacheriff/internal/driver"
)

// loadDriverDataCmd fetches a driver's cache entries and globally
// installed packages in the background, tagging the result with gen
// so the caller can discard it if the user has since moved on.
func loadDriverDataCmd(ctx context.Context, d driver.Driver, gen int) tea.Cmd {
	return func() tea.Msg {
		cache, err := d.CacheEntries(ctx)
		if err != nil {
			return driverDataMsg{gen: gen, err: fmt.Errorf("cache entries: %w", err)}
		}

		packages, err := d.GlobalPackages(ctx)
		if err != nil {
			return driverDataMsg{gen: gen, cache: cache, err: fmt.Errorf("global packages: %w", err)}
		}

		return driverDataMsg{gen: gen, cache: cache, packages: packages}
	}
}
