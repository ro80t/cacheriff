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

type dotnetDriver struct {
	base
}

func NewDotnetDriver() Driver {
	// localDir is left unset: a project's restored packages are only
	// referenced from the shared global-packages cache below (see
	// LocalPackages), never copied into the project.
	return dotnetDriver{base: base{
		id:          "dotnet",
		name:        "NuGet",
		binary:      "dotnet",
		supportedOS: []platform.OS{platform.Windows, platform.MacOS, platform.Linux},
		dirs:        []string{"bin", "obj"},
	}}
}

func dotnetNugetLocals(ctx context.Context) (map[string]string, error) {
	out, err := exec.CommandContext(ctx, "dotnet", "nuget", "locals", "all", "--list").Output()
	if err != nil {
		return nil, fmt.Errorf("dotnet nuget locals all --list: %w", err)
	}
	return parseDotnetNugetLocals(string(out)), nil
}

// parseDotnetNugetLocals parses `dotnet nuget locals all --list`'s
// "name: path" lines into a name->path map (http-cache, global-packages,
// temp, plugins-cache).
func parseDotnetNugetLocals(out string) map[string]string {
	locals := make(map[string]string)
	for line := range strings.SplitSeq(out, "\n") {
		name, path, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		locals[strings.TrimSpace(name)] = strings.TrimSpace(path)
	}
	return locals
}

func (d dotnetDriver) CacheDir(ctx context.Context) (string, error) {
	locals, err := dotnetNugetLocals(ctx)
	if err != nil {
		return "", err
	}
	return locals["global-packages"], nil
}

func (d dotnetDriver) CacheEntries(ctx context.Context) ([]Entry, error) {
	locals, err := dotnetNugetLocals(ctx)
	if err != nil {
		return nil, err
	}
	dirs := make([]namedDir, 0, len(locals))
	for name, path := range locals {
		dirs = append(dirs, namedDir{name: name, path: path})
	}
	return sizeCacheDirs(ctx, dirs), nil
}

func (d dotnetDriver) GlobalInstallDir(_ context.Context) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("dotnet: resolve home directory: %w", err)
	}
	return filepath.Join(home, ".dotnet", "tools"), nil
}

type dotnetToolListOutput struct {
	Data []struct {
		PackageID string `json:"packageId"`
		Version   string `json:"version"`
	} `json:"data"`
}

func (d dotnetDriver) GlobalPackages(ctx context.Context) ([]Entry, error) {
	toolsDir, err := d.GlobalInstallDir(ctx)
	if err != nil {
		return nil, err
	}
	out, err := exec.CommandContext(ctx, "dotnet", "tool", "list", "-g", "--format", "json").Output()
	if err != nil {
		return nil, fmt.Errorf("dotnet tool list -g --format json: %w", err)
	}
	var parsed dotnetToolListOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("dotnet tool list -g --format json: parse output: %w", err)
	}

	entries := make([]Entry, 0, len(parsed.Data))
	for _, p := range parsed.Data {
		// The tool's actual content lives under .store/<id>/<version>;
		// the entry in toolsDir itself is just a small shim.
		path := filepath.Join(toolsDir, ".store", strings.ToLower(p.PackageID), p.Version)
		size, err := dirSize(ctx, path)
		if err != nil {
			size = -1
		}
		entries = append(entries, Entry{
			Name:    p.PackageID,
			Version: p.Version,
			Path:    path,
			Kind:    KindGlobalPackage,
			Size:    size,
		})
	}
	return entries, nil
}

// dotnetProjectAssets is the subset of obj/project.assets.json (written
// by `dotnet restore`) naming each resolved package's relative path
// under the global-packages cache.
type dotnetProjectAssets struct {
	Libraries map[string]struct {
		Type string `json:"type"`
		Path string `json:"path"`
	} `json:"libraries"`
}

func (d dotnetDriver) LocalPackages(ctx context.Context, root string) ([]Entry, error) {
	assetsPath := filepath.Join(root, "obj", "project.assets.json")
	data, err := os.ReadFile(assetsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("dotnet: read %s: %w", assetsPath, err)
	}

	var assets dotnetProjectAssets
	if err := json.Unmarshal(data, &assets); err != nil {
		return nil, fmt.Errorf("dotnet: parse %s: %w", assetsPath, err)
	}

	packagesDir, err := d.CacheDir(ctx)
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, 0, len(assets.Libraries))
	for key, lib := range assets.Libraries {
		if lib.Type != "package" {
			continue // a project-to-project reference, not a package
		}
		name, version, ok := strings.Cut(key, "/")
		if !ok {
			continue
		}
		path := filepath.Join(packagesDir, lib.Path)
		size, err := dirSize(ctx, path)
		if err != nil {
			size = -1
		}
		entries = append(entries, Entry{
			Name:    name,
			Version: version,
			Path:    path,
			Kind:    KindLocalPackage,
			Size:    size,
		})
	}
	return entries, nil
}

func (d dotnetDriver) Remove(ctx context.Context, e Entry) error {
	switch e.Kind {
	case KindCache:
		return d.runCombined(ctx, "nuget", "locals", e.Name, "--clear")
	case KindGlobalPackage:
		name, err := textwrap.EscapeArg(e.Name)
		if err != nil {
			return fmt.Errorf("dotnet: %w", err)
		}
		return d.runCombined(ctx, "tool", "uninstall", "-g", name)
	default:
		return d.unsupportedKindErr(e.Kind)
	}
}
