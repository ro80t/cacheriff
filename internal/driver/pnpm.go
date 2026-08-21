package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"cacheriff/internal/platform"
)

type pnpmDriver struct {
	base
}

// NewPnpmDriver returns the Driver for pnpm.
func NewPnpmDriver() Driver {
	return pnpmDriver{base: base{
		id:          "pnpm",
		name:        "pnpm",
		binary:      "pnpm",
		supportedOS: []platform.OS{platform.Windows, platform.MacOS, platform.Linux},
		dirs:        []string{"node_modules"},
	}}
}

func (d pnpmDriver) CacheDir(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "pnpm", "store", "path").Output()
	if err != nil {
		return "", fmt.Errorf("pnpm store path: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (d pnpmDriver) CacheEntries(ctx context.Context) ([]Entry, error) {
	dir, err := d.CacheDir(ctx)
	if err != nil {
		return nil, err
	}
	if !pathExists(dir) {
		return nil, nil
	}
	size, err := dirSize(ctx, dir)
	if err != nil {
		size = -1
	}
	return []Entry{{
		Name: "pnpm store",
		Path: dir,
		Kind: KindCache,
		Size: size,
	}}, nil
}

func (d pnpmDriver) GlobalInstallDir(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "pnpm", "root", "-g").Output()
	if err != nil {
		return "", fmt.Errorf("pnpm root -g: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// pnpmListRoot is the shape of one element of `pnpm list [-g]
// --depth=0 --json`'s array output: a project (or, for -g, the
// single global "project") together with its direct dependencies.
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

func (pnpmDriver) LocalInstallDir(root string) (string, bool) {
	return filepath.Join(root, "node_modules"), true
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

func (pnpmDriver) Remove(ctx context.Context, e Entry) error {
	switch e.Kind {
	case KindCache:
		out, err := exec.CommandContext(ctx, "pnpm", "store", "prune").CombinedOutput()
		if err != nil {
			return fmt.Errorf("pnpm store prune: %w: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	case KindGlobalPackage:
		out, err := exec.CommandContext(ctx, "pnpm", "remove", "-g", e.Name).CombinedOutput()
		if err != nil {
			return fmt.Errorf("pnpm remove -g %s: %w: %s", e.Name, err, strings.TrimSpace(string(out)))
		}
		return nil
	default:
		return fmt.Errorf("pnpm: unsupported entry kind %s", e.Kind)
	}
}
