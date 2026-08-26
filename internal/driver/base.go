package driver

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"cacheriff/internal/platform"
)

type base struct {
	id          string
	name        string
	binary      string // CLI binary looked up on PATH for Available(), and run by runOutput/runCombined
	supportedOS []platform.OS
	dirs        []string // LocalArtifactDirNames
	localDir    string   // relative dir name (e.g. "node_modules") LocalInstallDir resolves under a project root; "" if this package manager has no per-project install directory
}

func (b base) ID() string   { return b.id }
func (b base) Name() string { return b.name }

func (b base) Available() bool {
	_, err := exec.LookPath(b.binary)
	return err == nil
}

func (b base) SupportedOS() []platform.OS {
	return b.supportedOS
}

func (b base) LocalArtifactDirNames() []string {
	return b.dirs
}

// LocalInstallDir reports root/localDir, or ("", false) if localDir
// is unset: some package managers (cargo, Go, Deno) have no
// per-project install directory because resolved dependencies live
// only in a shared, machine-wide cache instead.
func (b base) LocalInstallDir(root string) (string, bool) {
	if b.localDir == "" {
		return "", false
	}
	return filepath.Join(root, b.localDir), true
}

// runOutput runs the driver's own binary with args and returns its
// trimmed stdout, wrapping any failure with the command line for
// context. Used for the "ask the tool where its state lives" queries
// that recur across drivers (e.g. `npm config get cache`, `go env
// GOMODCACHE`).
func (b base) runOutput(ctx context.Context, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, b.binary, args...).Output()
	if err != nil {
		return "", fmt.Errorf("%s: %w", b.cmdLine(args), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// runCombined runs the driver's own binary with args, returning its
// combined output alongside any failure for context. Used by Remove
// implementations that shell out to the package manager's own
// removal command (e.g. `npm cache clean --force`, `pnpm remove -g`).
func (b base) runCombined(ctx context.Context, args ...string) error {
	out, err := exec.CommandContext(ctx, b.binary, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w: %s", b.cmdLine(args), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (b base) cmdLine(args []string) string {
	return strings.Join(append([]string{b.binary}, args...), " ")
}

// unsupportedKindErr builds the standard error Remove returns for an
// Entry.Kind it doesn't know how to handle.
func (b base) unsupportedKindErr(k EntryKind) error {
	return fmt.Errorf("%s: unsupported entry kind %s", b.id, k)
}

// singleDirCacheEntries builds the one-entry CacheEntries result for
// a driver whose entire cache lives under a single directory, sizing
// it if present. Returns (nil, nil) if dir doesn't exist yet (the
// package manager has never been used, or its cache was cleared).
func (base) singleDirCacheEntries(ctx context.Context, dir, label string) ([]Entry, error) {
	if !pathExists(dir) {
		return nil, nil
	}
	size, err := dirSize(ctx, dir)
	if err != nil {
		size = -1
	}
	return []Entry{{
		Name: label,
		Path: dir,
		Kind: KindCache,
		Size: size,
	}}, nil
}

// namedDir pairs a human-readable label with the path of one cache
// subdirectory a driver wants sizeCacheDirs to report on.
type namedDir struct {
	name string
	path string
}

// sizeCacheDirs sizes each of dirs concurrently and returns one Entry
// per directory that exists, silently skipping ones that don't (e.g.
// a cache subdirectory the tool hasn't populated yet) or whose path
// is empty (not reported by the tool at all). Used by drivers whose
// cache spans several named subdirectories (cargo's registry/git
// caches, Go's build/module caches, Deno's DENO_DIR subdirectories),
// each of which can hold enough files that sizing them one after
// another would be slow.
func sizeCacheDirs(ctx context.Context, dirs []namedDir) []Entry {
	entries := make([]Entry, len(dirs))
	present := make([]bool, len(dirs))
	var wg sync.WaitGroup
	for i, d := range dirs {
		if d.path == "" || !pathExists(d.path) {
			continue
		}
		present[i] = true
		wg.Add(1)
		go func(i int, name, path string) {
			defer wg.Done()
			size, err := dirSize(ctx, path)
			if err != nil {
				size = -1
			}
			entries[i] = Entry{
				Name: name,
				Path: path,
				Kind: KindCache,
				Size: size,
			}
		}(i, d.name, d.path)
	}
	wg.Wait()

	result := make([]Entry, 0, len(entries))
	for i, e := range entries {
		if present[i] {
			result = append(result, e)
		}
	}
	return result
}

// dirSize walks path and sums the size of every regular file under
// it. Some caches (e.g. cargo's registry/src, npm/bun/pnpm/yarn's
// content-addressable stores) hold enough files that a full walk can
// take a long time, so this checks ctx on every entry and aborts as
// soon as it's canceled or its deadline passes, rather than running
// to completion regardless of the caller's timeout.
func dirSize(ctx context.Context, path string) (int64, error) {
	var size int64
	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			// Skip entries we can't stat (permissions, races) rather
			// than failing the whole walk.
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		size += info.Size()
		return nil
	})
	return size, err
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
