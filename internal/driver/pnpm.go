package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"cacheriff/internal/platform"
	"cacheriff/internal/textwrap"
)

type pnpmDriver struct {
	base
}

func NewPnpmDriver() Driver {
	return pnpmDriver{base: base{
		id:          "pnpm",
		name:        "pnpm",
		binary:      "pnpm",
		supportedOS: []platform.OS{platform.Windows, platform.MacOS, platform.Linux},
		dirs:        []string{"node_modules"},
		localDir:    "node_modules",
	}}
}

func (d pnpmDriver) CacheDir(ctx context.Context) (string, error) {
	return d.runOutput(ctx, "store", "path")
}

func (d pnpmDriver) CacheEntries(ctx context.Context) ([]Entry, error) {
	dir, err := d.CacheDir(ctx)
	if err != nil {
		return nil, err
	}
	return d.singleDirCacheEntries(ctx, dir, "pnpm store")
}

func (d pnpmDriver) GlobalInstallDir(ctx context.Context) (string, error) {
	return d.runOutput(ctx, "root", "-g")
}

// pnpmListRoot is one element of `pnpm list [-g] --depth=0 --json`'s
// array output.
type pnpmListRoot struct {
	Dependencies map[string]struct {
		Version string `json:"version"`
		Path    string `json:"path"`
	} `json:"dependencies"`
}

func (d pnpmDriver) GlobalPackages(ctx context.Context) ([]Entry, error) {
	out, err := exec.CommandContext(ctx, "pnpm", "list", "-g", "--depth=0", "--json").Output()
	if err != nil {
		return nil, fmt.Errorf("pnpm list -g --json: %w", err)
	}
	return parsePnpmList(ctx, out, KindGlobalPackage)
}

func (d pnpmDriver) LocalPackages(ctx context.Context, root string) ([]Entry, error) {
	dir, _ := d.LocalInstallDir(root)
	if !pathExists(dir) {
		return nil, nil
	}

	cmd := exec.CommandContext(ctx, "pnpm", "list", "--depth=0", "--json")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pnpm list --json: %w", err)
	}
	return parsePnpmList(ctx, out, KindLocalPackage)
}

func parsePnpmList(ctx context.Context, out []byte, kind EntryKind) ([]Entry, error) {
	var roots []pnpmListRoot
	if err := json.Unmarshal(out, &roots); err != nil {
		return nil, fmt.Errorf("pnpm list --json: parse output: %w", err)
	}

	var entries []Entry
	for _, root := range roots {
		for name, dep := range root.Dependencies {
			size, err := dirSize(ctx, dep.Path)
			if err != nil {
				size = -1
			}
			entries = append(entries, Entry{
				Name:    name,
				Version: dep.Version,
				Path:    dep.Path,
				Kind:    kind,
				Size:    size,
			})
		}
	}
	return entries, nil
}

func (d pnpmDriver) Remove(ctx context.Context, e Entry) error {
	switch e.Kind {
	case KindCache:
		return d.runCombined(ctx, "store", "prune")
	case KindGlobalPackage:
		name, err := textwrap.EscapeArg(e.Name)
		if err != nil {
			return fmt.Errorf("pnpm: %w", err)
		}
		return d.runCombined(ctx, "remove", "-g", name)
	default:
		return d.unsupportedKindErr(e.Kind)
	}
}
