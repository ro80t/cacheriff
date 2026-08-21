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
	"strings"

	"cacheriff/internal/platform"
)

type yarnDriver struct {
	base
}

// NewYarnDriver returns the Driver for Yarn Classic (v1). Yarn
// Berry (v2+) uses a different cache layout and dropped the concept
// of global installs in favor of `yarn dlx`, so this driver targets
// the still-widely-used v1 line.
func NewYarnDriver() Driver {
	return yarnDriver{base: base{
		id:          "yarn",
		name:        "Yarn",
		binary:      "yarn",
		supportedOS: []platform.OS{platform.Windows, platform.MacOS, platform.Linux},
		dirs:        []string{"node_modules"},
	}}
}

func (d yarnDriver) CacheDir(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "yarn", "cache", "dir").Output()
	if err != nil {
		return "", fmt.Errorf("yarn cache dir: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (d yarnDriver) CacheEntries(ctx context.Context) ([]Entry, error) {
	dir, err := d.CacheDir(ctx)
	if err != nil {
		return nil, err
	}
	if !pathExists(dir) {
		return nil, nil
	}
	size, err := dirSize(dir)
	if err != nil {
		size = -1
	}
	return []Entry{{
		Name: "Yarn cache",
		Path: dir,
		Kind: KindCache,
		Size: size,
	}}, nil
}

func (d yarnDriver) yarnGlobalDir(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "yarn", "global", "dir").Output()
	if err != nil {
		return "", fmt.Errorf("yarn global dir: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (d yarnDriver) GlobalInstallDir(ctx context.Context) (string, error) {
	dir, err := d.yarnGlobalDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "node_modules"), nil
}

// GlobalPackages reads the global package.json's declared
// dependencies directly rather than shelling out to `yarn global list
// --json`: as of Yarn Classic 1.22, that command's JSON output only
// carries progress events, never the actual package list.
func (d yarnDriver) GlobalPackages(ctx context.Context) ([]Entry, error) {
	globalDir, err := d.yarnGlobalDir(ctx)
	if err != nil {
		return nil, err
	}
	return packagesFromGlobalManifest(globalDir, KindGlobalPackage)
}

func packagesFromGlobalManifest(globalDir string, kind EntryKind) ([]Entry, error) {
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
		size, err := dirSize(p)
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

func (yarnDriver) LocalInstallDir(root string) (string, bool) {
	return filepath.Join(root, "node_modules"), true
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
		size, err := dirSize(p)
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

// yarnListTreeLine is the one line, out of `yarn list --json`'s
// newline-delimited output, that actually carries the dependency
// tree; the rest are progress/warning events this driver ignores.
type yarnListTreeLine struct {
	Type string `json:"type"`
	Data struct {
		Trees []struct {
			Name string `json:"name"`
		} `json:"trees"`
	} `json:"data"`
}

// parseYarnListTree scans the NDJSON output of `yarn list --json`
// for its "tree" line and returns each entry's "name@version" spec.
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

func (yarnDriver) Remove(ctx context.Context, e Entry) error {
	switch e.Kind {
	case KindCache:
		out, err := exec.CommandContext(ctx, "yarn", "cache", "clean").CombinedOutput()
		if err != nil {
			return fmt.Errorf("yarn cache clean: %w: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	case KindGlobalPackage:
		out, err := exec.CommandContext(ctx, "yarn", "global", "remove", e.Name).CombinedOutput()
		if err != nil {
			return fmt.Errorf("yarn global remove %s: %w: %s", e.Name, err, strings.TrimSpace(string(out)))
		}
		return nil
	default:
		return fmt.Errorf("yarn: unsupported entry kind %s", e.Kind)
	}
}
