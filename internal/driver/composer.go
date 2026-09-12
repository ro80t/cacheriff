package driver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/ro80t/cacheriff/internal/platform"
	"github.com/ro80t/cacheriff/internal/textwrap"
)

type composerDriver struct {
	base
}

func NewComposerDriver() Driver {
	return composerDriver{base: base{
		id:          "composer",
		name:        "Composer",
		binary:      "composer",
		supportedOS: []platform.OS{platform.Windows, platform.MacOS, platform.Linux},
		dirs:        []string{"vendor"},
		localDir:    "vendor",
	}}
}

func (d composerDriver) CacheDir(ctx context.Context) (string, error) {
	return d.runOutput(ctx, "config", "-g", "cache-dir")
}

func (d composerDriver) CacheEntries(ctx context.Context) ([]Entry, error) {
	dir, err := d.CacheDir(ctx)
	if err != nil {
		return nil, err
	}
	return d.singleDirCacheEntries(ctx, dir, "composer cache")
}

func (d composerDriver) GlobalInstallDir(ctx context.Context) (string, error) {
	home, err := d.runOutput(ctx, "config", "-g", "home")
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "vendor"), nil
}

// composerShowOutput is the shape of both `composer show --format=json`
// and `composer global show --format=json`.
type composerShowOutput struct {
	Installed []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"installed"`
}

func (d composerDriver) GlobalPackages(ctx context.Context) ([]Entry, error) {
	vendorDir, err := d.GlobalInstallDir(ctx)
	if err != nil {
		return nil, err
	}
	out, err := exec.CommandContext(ctx, "composer", "global", "show", "--format=json").Output()
	if err != nil {
		return nil, fmt.Errorf("composer global show --format=json: %w", err)
	}
	parsed, err := parseComposerShowOutput(out)
	if err != nil {
		return nil, fmt.Errorf("composer global show --format=json: parse output: %w", err)
	}
	return composerEntriesFromShow(ctx, vendorDir, parsed, KindGlobalPackage), nil
}

func (d composerDriver) LocalPackages(ctx context.Context, root string) ([]Entry, error) {
	vendorDir, _ := d.LocalInstallDir(root)
	if !pathExists(vendorDir) {
		return nil, nil
	}
	cmd := exec.CommandContext(ctx, "composer", "show", "--format=json")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("composer show --format=json: %w", err)
	}
	parsed, err := parseComposerShowOutput(out)
	if err != nil {
		return nil, fmt.Errorf("composer show --format=json: parse output: %w", err)
	}
	return composerEntriesFromShow(ctx, vendorDir, parsed, KindLocalPackage), nil
}

// parseComposerShowOutput handles a genuine CLI quirk: `composer show
// --format=json` prints a bare "[]" instead of {"installed": []} when
// there are zero packages, rather than the object shape it uses
// otherwise.
func parseComposerShowOutput(out []byte) (composerShowOutput, error) {
	if trimmed := bytes.TrimSpace(out); len(trimmed) > 0 && trimmed[0] == '[' {
		return composerShowOutput{}, nil
	}
	var parsed composerShowOutput
	err := json.Unmarshal(out, &parsed)
	return parsed, err
}

func composerEntriesFromShow(ctx context.Context, vendorDir string, parsed composerShowOutput, kind EntryKind) []Entry {
	entries := make([]Entry, 0, len(parsed.Installed))
	for _, p := range parsed.Installed {
		// p.Name is "vendor/package"; Join handles the "/" fine on
		// every OS.
		path := filepath.Join(vendorDir, p.Name)
		size, err := dirSize(ctx, path)
		if err != nil {
			size = -1
		}
		entries = append(entries, Entry{
			Name:    p.Name,
			Version: p.Version,
			Path:    path,
			Kind:    kind,
			Size:    size,
		})
	}
	return entries
}

func (d composerDriver) Remove(ctx context.Context, e Entry) error {
	switch e.Kind {
	case KindCache:
		return d.runCombined(ctx, "clear-cache")
	case KindGlobalPackage:
		name, err := textwrap.EscapeArg(e.Name)
		if err != nil {
			return fmt.Errorf("composer: %w", err)
		}
		return d.runCombined(ctx, "global", "remove", name)
	default:
		return d.unsupportedKindErr(e.Kind)
	}
}
