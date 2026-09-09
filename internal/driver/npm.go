package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"cacheriff/internal/platform"
	"cacheriff/internal/textwrap"
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
		localDir:    "node_modules",
	}}
}

func (d npmDriver) CacheDir(ctx context.Context) (string, error) {
	return d.runOutput(ctx, "config", "get", "cache")
}

func (d npmDriver) CacheEntries(ctx context.Context) ([]Entry, error) {
	dir, err := d.CacheDir(ctx)
	if err != nil {
		return nil, err
	}
	return d.singleDirCacheEntries(ctx, dir, "npm cache")
}

func (d npmDriver) GlobalInstallDir(ctx context.Context) (string, error) {
	return d.runOutput(ctx, "root", "-g")
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

func (d npmDriver) Remove(ctx context.Context, e Entry) error {
	switch e.Kind {
	case KindCache:
		return d.runCombined(ctx, "cache", "clean", "--force")
	case KindGlobalPackage:
		name, err := textwrap.EscapeArg(e.Name)
		if err != nil {
			return fmt.Errorf("npm: %w", err)
		}
		return d.runCombined(ctx, "uninstall", "-g", name)
	default:
		return d.unsupportedKindErr(e.Kind)
	}
}
