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

type cargoDriver struct {
	base
}

// NewCargoDriver returns the Driver for Rust's cargo/crates.io toolchain.
func NewCargoDriver() Driver {
	return cargoDriver{base: base{
		id:          "cargo",
		name:        "Cargo",
		binary:      "cargo",
		supportedOS: []platform.OS{platform.Windows, platform.MacOS, platform.Linux},
		dirs:        []string{"target"},
	}}
}

func cargoHome() (string, error) {
	if v := os.Getenv("CARGO_HOME"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cargo: resolve home directory: %w", err)
	}
	return filepath.Join(home, ".cargo"), nil
}

// cargoCacheDirs describes the subdirectories of CARGO_HOME that hold
// downloaded/derived data rather than user configuration.
var cargoCacheDirs = []struct {
	name string
	rel  string
}{
	{"Registry cache (.crate archives)", filepath.Join("registry", "cache")},
	{"Registry source (extracted crates)", filepath.Join("registry", "src")},
	{"Registry index", filepath.Join("registry", "index")},
	{"Git checkouts", filepath.Join("git", "checkouts")},
	{"Git database", filepath.Join("git", "db")},
}

func (d cargoDriver) CacheEntries(_ context.Context) ([]Entry, error) {
	home, err := cargoHome()
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, c := range cargoCacheDirs {
		p := filepath.Join(home, c.rel)
		if !pathExists(p) {
			continue
		}
		size, err := dirSize(p)
		if err != nil {
			size = -1
		}
		entries = append(entries, Entry{
			Name: c.name,
			Path: p,
			Kind: KindCache,
			Size: size,
		})
	}
	return entries, nil
}

// cargoInstallListRe matches the "<name> v<version>:" header lines
// printed by `cargo install --list`, e.g. "ripgrep v13.0.0:".
var cargoInstallListRe = regexp.MustCompile(`^(\S+) v(\S+):$`)

func (d cargoDriver) GlobalPackages(ctx context.Context) ([]Entry, error) {
	out, err := exec.CommandContext(ctx, "cargo", "install", "--list").Output()
	if err != nil {
		return nil, fmt.Errorf("cargo install --list: %w", err)
	}

	home, err := cargoHome()
	if err != nil {
		return nil, err
	}
	binDir := filepath.Join(home, "bin")

	var entries []Entry
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, " ") {
			continue // indented lines list installed binaries, not packages
		}
		m := cargoInstallListRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		entries = append(entries, Entry{
			Name:    m[1],
			Version: m[2],
			Path:    binDir,
			Kind:    KindGlobalPackage,
			Size:    -1,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("cargo install --list: parse output: %w", err)
	}
	return entries, nil
}

func (cargoDriver) Remove(ctx context.Context, e Entry) error {
	switch e.Kind {
	case KindCache:
		return os.RemoveAll(e.Path)
	case KindGlobalPackage:
		out, err := exec.CommandContext(ctx, "cargo", "uninstall", e.Name).CombinedOutput()
		if err != nil {
			return fmt.Errorf("cargo uninstall %s: %w: %s", e.Name, err, strings.TrimSpace(string(out)))
		}
		return nil
	default:
		return fmt.Errorf("cargo: unsupported entry kind %s", e.Kind)
	}
}
