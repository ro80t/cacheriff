package ui

import (
	"context"
	"fmt"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"cacheriff/internal/driver"
)

// loadDriverDataCmd fetches a driver's cache entries and its globally
// and locally installed packages in the background, tagging the
// result with gen so the caller can discard it if the user has since
// moved on. root is the project directory LocalPackages is evaluated
// against (typically cacheriff's working directory). The three
// fetches are independent (and cache entries in particular can be
// slow to size, e.g. a large package-manager cache holding many
// files), so they run concurrently rather than one after the other.
func loadDriverDataCmd(ctx context.Context, d driver.Driver, root string, gen int) tea.Cmd {
	return func() tea.Msg {
		var cache, global, local []driver.Entry
		var cacheErr, globalErr, localErr error

		var wg sync.WaitGroup
		wg.Add(3)
		go func() {
			defer wg.Done()
			cache, cacheErr = d.CacheEntries(ctx)
		}()
		go func() {
			defer wg.Done()
			global, globalErr = d.GlobalPackages(ctx)
		}()
		go func() {
			defer wg.Done()
			local, localErr = d.LocalPackages(ctx, root)
		}()
		wg.Wait()

		if cacheErr != nil {
			return driverDataMsg{gen: gen, err: fmt.Errorf("cache entries: %w", cacheErr)}
		}
		if globalErr != nil {
			return driverDataMsg{gen: gen, cache: cache, err: fmt.Errorf("global packages: %w", globalErr)}
		}
		if localErr != nil {
			return driverDataMsg{gen: gen, cache: cache, global: global, err: fmt.Errorf("local packages: %w", localErr)}
		}
		return driverDataMsg{gen: gen, cache: cache, global: global, local: local}
	}
}
