package ui

import (
	"context"
	"fmt"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"cacheriff/internal/driver"
)

// loadDriverDataCmd runs the three independent fetches concurrently,
// since sizing cache entries in particular can be slow.
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

// removePackageCmd runs the driver's own uninstall command rather
// than deleting files directly, so its metadata/lockfiles stay
// consistent.
func removePackageCmd(ctx context.Context, d driver.Driver, e driver.Entry, gen int) tea.Cmd {
	return func() tea.Msg {
		err := d.Remove(ctx, e)
		return packageRemovedMsg{gen: gen, entry: e, err: err}
	}
}
