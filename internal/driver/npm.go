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

type npmDriver struct {
	base
}

// NewNPMDriver returns the Driver for npm (Node.js packages).
func NewNPMDriver() Driver {
	return npmDriver{base: base{
		id:          "npm",
		name:        "npm",
		binary:      "npm",
		supportedOS: []platform.OS{platform.Windows, platform.MacOS, platform.Linux},
		dirs:        []string{"node_modules"},
	}}
}

func npmConfigGet(ctx context.Context, key string) (string, error) {
	out, err := exec.CommandContext(ctx, "npm", "config", "get", key).Output()
	if err != nil {
		return "", fmt.Errorf("npm config get %s: %w", key, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (d npmDriver) CacheDir(ctx context.Context) (string, error) {
	return npmConfigGet(ctx, "cache")
}

func (d npmDriver) CacheEntries(ctx context.Context) ([]Entry, error) {
	cacheDir, err := d.CacheDir(ctx)
	if err != nil {
		return nil, err
	}
	if !pathExists(cacheDir) {
		return nil, nil
	}
	size, err := dirSize(ctx, cacheDir)
	if err != nil {
		size = -1
	}
	return []Entry{{
		Name: "npm cache",
		Path: cacheDir,
		Kind: KindCache,
		Size: size,
	}}, nil
}

func (d npmDriver) GlobalInstallDir(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "npm", "root", "-g").Output()
	if err != nil {
		return "", fmt.Errorf("npm root -g: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// npmListOutput is the shape of `npm ls -g --depth=0 --json` and
// `npm ls --depth=0 --json`.
type npmListOutput struct {
	Dependencies map[string]struct {
		Version string `json:"version"`
	} `json:"dependencies"`
}

func (d npmDriver) GlobalPackages(ctx context.Context) ([]Entry, error) {
	globalRoot, err := d.GlobalInstallDir(ctx)
	if err != nil {
		return nil, err
	}

	// `npm ls -g` exits non-zero whenever the dependency tree has any
	// problem (e.g. one extraneous/invalid package) even though it
	// still prints valid JSON, so only bail out if the output can't
	// be parsed at all.
	out, _ := exec.CommandContext(ctx, "npm", "ls", "-g", "--depth=0", "--json").Output()
	var parsed npmListOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("npm ls -g --json: parse output: %w", err)
	}
	return npmEntriesFromList(ctx, globalRoot, parsed, KindGlobalPackage), nil
}

func (d npmDriver) LocalInstallDir(root string) (string, bool) {
	return filepath.Join(root, "node_modules"), true
}

func (d npmDriver) LocalPackages(ctx context.Context, root string) ([]Entry, error) {
	dir, _ := d.LocalInstallDir(root)
	if !pathExists(dir) {
		return nil, nil
	}

	// Same non-zero-exit caveat as GlobalPackages above applies here.
	cmd := exec.CommandContext(ctx, "npm", "ls", "--depth=0", "--json")
	cmd.Dir = root
	out, _ := cmd.Output()
	var parsed npmListOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("npm ls --json: parse output: %w", err)
	}
	return npmEntriesFromList(ctx, dir, parsed, KindLocalPackage), nil
}

// npmEntriesFromList turns a parsed `npm ls [-g] --json` result into
// Entries, resolving each package's on-disk path under baseDir (a
// node_modules directory, local or global).
func npmEntriesFromList(ctx context.Context, baseDir string, parsed npmListOutput, kind EntryKind) []Entry {
	entries := make([]Entry, 0, len(parsed.Dependencies))
	for name, meta := range parsed.Dependencies {
		// Scoped package names ("@scope/name") map to a nested
		// "@scope/name" directory under baseDir.
		parts := append([]string{baseDir}, strings.Split(name, "/")...)
		p := filepath.Join(parts...)

		size, err := dirSize(ctx, p)
		if err != nil {
			size = -1
		}
		entries = append(entries, Entry{
			Name:    name,
			Version: meta.Version,
			Path:    p,
			Kind:    kind,
			Size:    size,
		})
	}
	return entries
}

func (npmDriver) Remove(ctx context.Context, e Entry) error {
	switch e.Kind {
	case KindCache:
		out, err := exec.CommandContext(ctx, "npm", "cache", "clean", "--force").CombinedOutput()
		if err != nil {
			return fmt.Errorf("npm cache clean --force: %w: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	case KindGlobalPackage:
		out, err := exec.CommandContext(ctx, "npm", "uninstall", "-g", e.Name).CombinedOutput()
		if err != nil {
			return fmt.Errorf("npm uninstall -g %s: %w: %s", e.Name, err, strings.TrimSpace(string(out)))
		}
		return nil
	default:
		return fmt.Errorf("npm: unsupported entry kind %s", e.Kind)
	}
}
