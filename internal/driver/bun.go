package driver

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"cacheriff/internal/platform"
)

type bunDriver struct {
	base
}

// NewBunDriver returns the Driver for Bun.
func NewBunDriver() Driver {
	return bunDriver{base: base{
		id:          "bun",
		name:        "Bun",
		binary:      "bun",
		supportedOS: []platform.OS{platform.Windows, platform.MacOS, platform.Linux},
		dirs:        []string{"node_modules"},
	}}
}

func bunHome() (string, error) {
	if v := os.Getenv("BUN_INSTALL"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("bun: resolve home directory: %w", err)
	}
	return filepath.Join(home, ".bun"), nil
}

func (d bunDriver) CacheDir(_ context.Context) (string, error) {
	home, err := bunHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "install", "cache"), nil
}

func (d bunDriver) CacheEntries(ctx context.Context) ([]Entry, error) {
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
		Name: "Bun install cache",
		Path: dir,
		Kind: KindCache,
		Size: size,
	}}, nil
}

// bunGlobalListHeaderRe matches the first line of `bun pm ls -g`'s
// output, which reports the resolved global install root, e.g.
// "C:\Users\me node_modules (12)" or, with nothing installed,
// "C:\Users\me node_modules" with no count. This is read from the
// command's own output rather than assumed to be
// "$BUN_INSTALL/install/global", since a bunfig.toml can redirect it
// elsewhere.
var bunGlobalListHeaderRe = regexp.MustCompile(`^(.+) node_modules(?: \(\d+\))?$`)

// bunGlobalListEntryRe matches a top-level package line in `bun pm ls
// -g`'s tree output, e.g. "├── left-pad@1.3.0" or "└── left-pad@1.3.0".
// Nested (transitive) lines are indented further and don't match.
var bunGlobalListEntryRe = regexp.MustCompile(`^(?:├──|└──) (.+)$`)

func (d bunDriver) GlobalInstallDir(ctx context.Context) (string, error) {
	root, _, err := bunGlobalList(ctx)
	return root, err
}

func (d bunDriver) GlobalPackages(ctx context.Context) ([]Entry, error) {
	root, specs, err := bunGlobalList(ctx)
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, spec := range specs {
		name, version := splitPackageSpec(spec)
		p := filepath.Join(root, "node_modules", filepath.FromSlash(name))
		size, err := dirSize(ctx, p)
		if err != nil {
			size = -1
		}
		entries = append(entries, Entry{
			Name:    name,
			Version: version,
			Path:    p,
			Kind:    KindGlobalPackage,
			Size:    size,
		})
	}
	return entries, nil
}

// bunGlobalList runs `bun pm ls -g`, which (unlike most bun pm
// subcommands) prints the tree of directly-installed global packages
// by default without needing a --depth flag, and returns the
// resolved global node_modules root together with each
// "name@version" entry.
func bunGlobalList(ctx context.Context) (string, []string, error) {
	out, err := exec.CommandContext(ctx, "bun", "pm", "ls", "-g").Output()
	if err != nil {
		return "", nil, fmt.Errorf("bun pm ls -g: %w", err)
	}
	return parseBunPmList(out)
}

func parseBunPmList(out []byte) (string, []string, error) {
	var root string
	var specs []string
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if root == "" {
			if m := bunGlobalListHeaderRe.FindStringSubmatch(line); m != nil {
				root = m[1]
				continue
			}
		}
		if m := bunGlobalListEntryRe.FindStringSubmatch(line); m != nil {
			specs = append(specs, m[1])
		}
	}
	if err := scanner.Err(); err != nil {
		return "", nil, fmt.Errorf("parse output: %w", err)
	}
	if root == "" {
		return "", nil, fmt.Errorf("could not find global install path in output")
	}
	return root, specs, nil
}

func (bunDriver) LocalInstallDir(root string) (string, bool) {
	return filepath.Join(root, "node_modules"), true
}

func (d bunDriver) LocalPackages(ctx context.Context, root string) ([]Entry, error) {
	dir, _ := d.LocalInstallDir(root)
	if !pathExists(dir) {
		return nil, nil
	}

	cmd := exec.CommandContext(ctx, "bun", "pm", "ls")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bun pm ls: %w", err)
	}

	var entries []Entry
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		m := bunGlobalListEntryRe.FindStringSubmatch(scanner.Text())
		if m == nil {
			continue
		}
		name, version := splitPackageSpec(m[1])
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
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("bun pm ls: parse output: %w", err)
	}
	return entries, nil
}

func (bunDriver) Remove(ctx context.Context, e Entry) error {
	switch e.Kind {
	case KindCache:
		return os.RemoveAll(e.Path)
	case KindGlobalPackage:
		out, err := exec.CommandContext(ctx, "bun", "remove", "-g", e.Name).CombinedOutput()
		if err != nil {
			return fmt.Errorf("bun remove -g %s: %w: %s", e.Name, err, strings.TrimSpace(string(out)))
		}
		return nil
	default:
		return fmt.Errorf("bun: unsupported entry kind %s", e.Kind)
	}
}
