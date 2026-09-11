package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ro80t/cacheriff/internal/platform"
	"github.com/ro80t/cacheriff/internal/textwrap"
)

type nixDriver struct {
	base
}

func NewNixDriver() Driver {
	// localDir is left unset: Nix resolves dependencies into the
	// shared, content-addressed /nix/store rather than copying them
	// into the project, the same as cargo, Go, and Deno.
	return nixDriver{base: base{
		id:   "nix",
		name: "Nix",
		// nix-env, unlike the unified `nix` CLI, needs no
		// experimental-features flag enabled.
		binary:      "nix-env",
		supportedOS: []platform.OS{platform.MacOS, platform.Linux},
		dirs:        []string{"result"},
	}}
}

const nixStoreGCEntryName = "Nix store (garbage collection)"

func nixCacheDir() (string, error) {
	if v := os.Getenv("XDG_CACHE_HOME"); v != "" {
		return filepath.Join(v, "nix"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("nix: resolve home directory: %w", err)
	}
	return filepath.Join(home, ".cache", "nix"), nil
}

func nixStoreDir() string {
	if v := os.Getenv("NIX_STORE_DIR"); v != "" {
		return v
	}
	return "/nix/store"
}

func (d nixDriver) CacheDir(_ context.Context) (string, error) {
	return nixCacheDir()
}

// The GC entry's size is unknown without a full dead-path scan, and it
// must be reclaimed via nix-collect-garbage rather than deleted
// directly -- /nix/store can't be edited by hand.
func (d nixDriver) CacheEntries(ctx context.Context) ([]Entry, error) {
	dir, err := nixCacheDir()
	if err != nil {
		return nil, err
	}
	entries, err := d.singleDirCacheEntries(ctx, dir, "Download/eval cache")
	if err != nil {
		return nil, err
	}
	entries = append(entries, Entry{
		Name: nixStoreGCEntryName,
		Path: nixStoreDir(),
		Kind: KindCache,
		Size: -1,
	})
	return entries, nil
}

func nixProfileDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("nix: resolve home directory: %w", err)
	}
	return filepath.Join(home, ".nix-profile"), nil
}

func (d nixDriver) GlobalInstallDir(_ context.Context) (string, error) {
	return nixProfileDir()
}

type nixEnvPackage struct {
	Pname   string `json:"pname"`
	Version string `json:"version"`
}

// Size is left unknown: nix-env -q --json doesn't reliably include
// each package's store output path (NixOS/nix#5877).
func (d nixDriver) GlobalPackages(ctx context.Context) ([]Entry, error) {
	profileDir, err := d.GlobalInstallDir(ctx)
	if err != nil {
		return nil, err
	}

	out, err := exec.CommandContext(ctx, d.binary, "-q", "--json").Output()
	if err != nil {
		return nil, fmt.Errorf("nix-env -q --json: %w", err)
	}
	var parsed map[string]nixEnvPackage
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("nix-env -q --json: parse output: %w", err)
	}

	entries := make([]Entry, 0, len(parsed))
	for _, pkg := range parsed {
		entries = append(entries, Entry{
			Name:    pkg.Pname,
			Version: pkg.Version,
			Path:    profileDir,
			Kind:    KindGlobalPackage,
			Size:    -1,
		})
	}
	return entries, nil
}

// LocalPackages always reports nothing: flake.lock pins input
// revisions, but built outputs live in the shared /nix/store (already
// covered by CacheEntries), never copied into the project.
func (nixDriver) LocalPackages(_ context.Context, _ string) ([]Entry, error) {
	return nil, nil
}

func (d nixDriver) Remove(ctx context.Context, e Entry) error {
	switch e.Kind {
	case KindCache:
		if e.Name == nixStoreGCEntryName {
			out, err := exec.CommandContext(ctx, "nix-collect-garbage", "-d").CombinedOutput()
			if err != nil {
				return fmt.Errorf("nix-collect-garbage -d: %w: %s", err, strings.TrimSpace(string(out)))
			}
			return nil
		}
		return os.RemoveAll(e.Path)
	case KindGlobalPackage:
		name, err := textwrap.EscapeArg(e.Name)
		if err != nil {
			return fmt.Errorf("nix: %w", err)
		}
		return d.runCombined(ctx, "-e", name)
	default:
		return d.unsupportedKindErr(e.Kind)
	}
}
