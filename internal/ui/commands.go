package ui

import (
	"context"
	"fmt"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"cacheriff/internal/driver"
)

// loadDriverDataCmd fetches a driver's cache entries and globally
// installed packages in the background, tagging the result with gen
// so the caller can discard it if the user has since moved on. The
// two fetches are independent (and cache entries in particular can be
// slow to size, e.g. a large package-manager cache holding many
// files), so they run concurrently rather than one after the other.
func loadDriverDataCmd(ctx context.Context, d driver.Driver, gen int) tea.Cmd {
	return func() tea.Msg {
		var cache, packages []driver.Entry
		var cacheErr, packagesErr error

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			cache, cacheErr = d.CacheEntries(ctx)
		}()
		go func() {
			defer wg.Done()
			packages, packagesErr = d.GlobalPackages(ctx)
		}()
		wg.Wait()

		if cacheErr != nil {
			return driverDataMsg{gen: gen, err: fmt.Errorf("cache entries: %w", cacheErr)}
		}
		if packagesErr != nil {
			return driverDataMsg{gen: gen, cache: cache, err: fmt.Errorf("global packages: %w", packagesErr)}
		}
		return driverDataMsg{gen: gen, cache: cache, packages: packages}
	}
}
