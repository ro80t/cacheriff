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

type npmDriver struct{}

// NewNPMDriver returns the Driver for npm (Node.js packages).
func NewNPMDriver() Driver { return npmDriver{} }

func (npmDriver) ID() string   { return "npm" }
func (npmDriver) Name() string { return "npm" }

func (npmDriver) Available() bool {
	_, err := exec.LookPath("npm")
	return err == nil
}

func (npmDriver) SupportedOS() []platform.OS {
	return []platform.OS{platform.Windows, platform.MacOS, platform.Linux}
}

func (npmDriver) LocalArtifactDirNames() []string {
	return []string{"node_modules"}
}

func npmConfigGet(ctx context.Context, key string) (string, error) {
	out, err := exec.CommandContext(ctx, "npm", "config", "get", key).Output()
	if err != nil {
		return "", fmt.Errorf("npm config get %s: %w", key, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (d npmDriver) CacheEntries(ctx context.Context) ([]Entry, error) {
	cacheDir, err := npmConfigGet(ctx, "cache")
	if err != nil {
		return nil, err
	}
	if !pathExists(cacheDir) {
		return nil, nil
	}
	size, err := dirSize(cacheDir)
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

// npmListOutput is the shape of `npm ls -g --depth=0 --json`.
type npmListOutput struct {
	Dependencies map[string]struct {
		Version string `json:"version"`
	} `json:"dependencies"`
}

func (d npmDriver) GlobalPackages(ctx context.Context) ([]Entry, error) {
	rootOut, err := exec.CommandContext(ctx, "npm", "root", "-g").Output()
	if err != nil {
		return nil, fmt.Errorf("npm root -g: %w", err)
	}
	globalRoot := strings.TrimSpace(string(rootOut))

	// `npm ls -g` exits non-zero whenever the dependency tree has any
	// problem (e.g. one extraneous/invalid package) even though it
	// still prints valid JSON, so only bail out if the output can't
	// be parsed at all.
	out, _ := exec.CommandContext(ctx, "npm", "ls", "-g", "--depth=0", "--json").Output()
	var parsed npmListOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("npm ls -g --json: parse output: %w", err)
	}

	entries := make([]Entry, 0, len(parsed.Dependencies))
	for name, meta := range parsed.Dependencies {
		// Scoped package names ("@scope/name") map to a nested
		// "@scope/name" directory under the global root.
		parts := append([]string{globalRoot}, strings.Split(name, "/")...)
		p := filepath.Join(parts...)

		size, err := dirSize(p)
		if err != nil {
			size = -1
		}
		entries = append(entries, Entry{
			Name:    name,
			Version: meta.Version,
			Path:    p,
			Kind:    KindGlobalPackage,
			Size:    size,
		})
	}
	return entries, nil
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
