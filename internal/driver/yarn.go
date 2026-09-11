package driver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ro80t/cacheriff/internal/platform"
	"github.com/ro80t/cacheriff/internal/textwrap"
)

type yarnDriver struct {
	base
}

// NewYarnDriver targets Yarn Classic (v1); Yarn Berry (v2+) uses a
// different cache layout and dropped global installs for `yarn dlx`.
func NewYarnDriver() Driver {
	return yarnDriver{base: base{
		id:          "yarn",
		name:        "Yarn",
		binary:      "yarn",
		supportedOS: []platform.OS{platform.Windows, platform.MacOS, platform.Linux},
		dirs:        []string{"node_modules"},
		localDir:    "node_modules",
	}}
}

func (d yarnDriver) CacheDir(ctx context.Context) (string, error) {
	return d.runOutput(ctx, "cache", "dir")
}

func (d yarnDriver) CacheEntries(ctx context.Context) ([]Entry, error) {
	dir, err := d.CacheDir(ctx)
	if err != nil {
		return nil, err
	}
	return d.singleDirCacheEntries(ctx, dir, "Yarn cache")
}

func (d yarnDriver) yarnGlobalDir(ctx context.Context) (string, error) {
	return d.runOutput(ctx, "global", "dir")
}

func (d yarnDriver) GlobalInstallDir(ctx context.Context) (string, error) {
	dir, err := d.yarnGlobalDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "node_modules"), nil
}

// GlobalPackages reads the global package.json directly: `yarn global
// list --json` (as of 1.22) only emits progress events, never the list.
func (d yarnDriver) GlobalPackages(ctx context.Context) ([]Entry, error) {
	globalDir, err := d.yarnGlobalDir(ctx)
	if err != nil {
		return nil, err
	}
	return packagesFromGlobalManifest(ctx, globalDir, KindGlobalPackage)
}

func packagesFromGlobalManifest(ctx context.Context, globalDir string, kind EntryKind) ([]Entry, error) {
	manifestPath := filepath.Join(globalDir, "package.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", manifestPath, err)
	}

	var manifest struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse %s: %w", manifestPath, err)
	}

	nodeModules := filepath.Join(globalDir, "node_modules")
	var entries []Entry
	for name := range manifest.Dependencies {
		p := filepath.Join(nodeModules, filepath.FromSlash(name))
		size, err := dirSize(ctx, p)
		if err != nil {
			size = -1
		}
		entries = append(entries, Entry{
			Name:    name,
			Version: resolvedPackageVersion(p),
			Path:    p,
			Kind:    kind,
			Size:    size,
		})
	}
	return entries, nil
}

func (d yarnDriver) LocalPackages(ctx context.Context, root string) ([]Entry, error) {
	dir, _ := d.LocalInstallDir(root)
	if !pathExists(dir) {
		return nil, nil
	}

	cmd := exec.CommandContext(ctx, "yarn", "list", "--depth=0", "--json")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("yarn list --json: %w", err)
	}
	specs, err := parseYarnListTree(out)
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, spec := range specs {
		name, version := splitPackageSpec(spec)
		p := filepath.Join(dir, filepath.FromSlash(name))
		size, err := dirSize(ctx, p)
		if err != nil {
			size = -1
		}
		entries = append(entries, Entry{
			Name:    name,
			Version: version,
			Path:    p,
			Kind:    KindLocalPackage,
			Size:    size,
		})
	}
	return entries, nil
}

// yarnListTreeLine is the "tree" line of `yarn list --json`'s NDJSON
// output; other lines are progress/warning events, ignored.
type yarnListTreeLine struct {
	Type string `json:"type"`
	Data struct {
		Trees []struct {
			Name string `json:"name"`
		} `json:"trees"`
	} `json:"data"`
}

func parseYarnListTree(out []byte) ([]string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		var line yarnListTreeLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue // not every line is JSON (e.g. deprecation warnings)
		}
		if line.Type != "tree" {
			continue
		}
		specs := make([]string, 0, len(line.Data.Trees))
		for _, t := range line.Data.Trees {
			specs = append(specs, t.Name)
		}
		return specs, nil
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("yarn list --json: parse output: %w", err)
	}
	return nil, nil
}

func (d yarnDriver) Remove(ctx context.Context, e Entry) error {
	switch e.Kind {
	case KindCache:
		return d.runCombined(ctx, "cache", "clean")
	case KindGlobalPackage:
		name, err := textwrap.EscapeArg(e.Name)
		if err != nil {
			return fmt.Errorf("yarn: %w", err)
		}
		return d.runCombined(ctx, "global", "remove", name)
	default:
		return d.unsupportedKindErr(e.Kind)
	}
}
