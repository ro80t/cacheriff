package ui

import (
	"context"
	"fmt"
	"os"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ro80t/cacheriff/internal/driver"
)

// loadDriverDataCmd runs the independent fetches concurrently, since
// sizing cache entries in particular can be slow.
func loadDriverDataCmd(ctx context.Context, d driver.Driver, root string, gen int) tea.Cmd {
	return func() tea.Msg {
		var cache, global, local []driver.Entry
		var cacheErr, globalErr, localErr error
		var globalDir string

		var wg sync.WaitGroup
		wg.Add(4)
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
		go func() {
			defer wg.Done()
			globalDir, _ = d.GlobalInstallDir(ctx)
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

		paths := installPathEntries(d, root, globalDir)
		return driverDataMsg{gen: gen, cache: append(paths, cache...), global: global, local: local}
	}
}

// installPathEntries surfaces where this driver installs packages as
// display-only Entries alongside the cache list (Size -1, same as any
// other not-applicable size). The local path is included only if it
// actually exists: unlike the cache/global dirs, LocalInstallDir
// reports a path unconditionally even for a project that has never
// installed anything there.
func installPathEntries(d driver.Driver, root, globalDir string) []driver.Entry {
	var entries []driver.Entry
	if globalDir != "" {
		entries = append(entries, driver.Entry{Name: "Global install", Path: globalDir, Size: -1})
	}
	if localDir, ok := d.LocalInstallDir(root); ok {
		if info, err := os.Stat(localDir); err == nil && info.IsDir() {
			entries = append(entries, driver.Entry{Name: "Local install", Path: localDir, Size: -1})
		}
	}
	return entries
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
