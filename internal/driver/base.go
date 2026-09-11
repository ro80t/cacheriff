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

	"github.com/ro80t/cacheriff/internal/platform"
)

type base struct {
	id          string
	name        string
	binary      string // CLI binary on PATH
	supportedOS []platform.OS
	dirs        []string // LocalArtifactDirNames
	localDir    string   // relative dir name (e.g. "node_modules"); "" if none
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

func (b base) LocalInstallDir(root string) (string, bool) {
	if b.localDir == "" {
		return "", false
	}
	return filepath.Join(root, b.localDir), true
}

// runOutput runs the driver's binary with args and returns its
// trimmed stdout, wrapping any failure with the command line.
func (b base) runOutput(ctx context.Context, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, b.binary, args...).Output()
	if err != nil {
		return "", fmt.Errorf("%s: %w", b.cmdLine(args), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// runCombined runs the driver's binary with args, wrapping any
// failure with the command line and combined output.
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

func (b base) unsupportedKindErr(k EntryKind) error {
	return fmt.Errorf("%s: unsupported entry kind %s", b.id, k)
}

// singleDirCacheEntries returns (nil, nil) if dir doesn't exist yet.
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

type namedDir struct {
	name string
	path string
}

// sizeCacheDirs sizes each of dirs concurrently (sizing can be slow
// for large caches) and returns one Entry per directory that exists,
// skipping ones that don't or whose path is empty.
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

// dirSize sums the size of every regular file under path, checking
// ctx on each entry so a large walk can be aborted early.
func dirSize(ctx context.Context, path string) (int64, error) {
	var size int64
	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
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
