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

	"github.com/ro80t/cacheriff/internal/platform"
	"github.com/ro80t/cacheriff/internal/textwrap"
)

type cargoDriver struct {
	base
}

func NewCargoDriver() Driver {
	// localDir is left unset: cargo has no per-project install
	// directory; dependencies live in CARGO_HOME's shared registry.
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

func (d cargoDriver) CacheDir(_ context.Context) (string, error) {
	return cargoHome()
}

func (d cargoDriver) CacheEntries(ctx context.Context) ([]Entry, error) {
	home, err := cargoHome()
	if err != nil {
		return nil, err
	}

	dirs := make([]namedDir, len(cargoCacheDirs))
	for i, c := range cargoCacheDirs {
		dirs[i] = namedDir{name: c.name, path: filepath.Join(home, c.rel)}
	}
	entries := sizeCacheDirs(ctx, dirs)
	entries = append(entries, cargoToolchainEntries(ctx)...)
	return entries, nil
}

func rustupHome() (string, error) {
	if v := os.Getenv("RUSTUP_HOME"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cargo: resolve home directory: %w", err)
	}
	return filepath.Join(home, ".rustup"), nil
}

// Reads the toolchains dir directly rather than `rustup toolchain
// list`, since it can exist even when rustup isn't on PATH.
func cargoToolchainEntries(ctx context.Context) []Entry {
	home, err := rustupHome()
	if err != nil {
		return nil
	}
	toolchainsDir := filepath.Join(home, "toolchains")
	items, err := os.ReadDir(toolchainsDir)
	if err != nil {
		return nil
	}

	dirs := make([]namedDir, 0, len(items))
	for _, item := range items {
		if item.IsDir() {
			dirs = append(dirs, namedDir{name: item.Name(), path: filepath.Join(toolchainsDir, item.Name())})
		}
	}

	entries := sizeCacheDirs(ctx, dirs)
	for i := range entries {
		entries[i].Kind = KindToolchain
	}
	return entries
}

func (d cargoDriver) GlobalInstallDir(_ context.Context) (string, error) {
	home, err := cargoHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "bin"), nil
}

// cargoInstallListRe matches the "<name> v<version>:" header lines
// printed by `cargo install --list`, e.g. "ripgrep v13.0.0:".
var cargoInstallListRe = regexp.MustCompile(`^(\S+) v(\S+):$`)

func (d cargoDriver) GlobalPackages(ctx context.Context) ([]Entry, error) {
	out, err := exec.CommandContext(ctx, "cargo", "install", "--list").Output()
	if err != nil {
		return nil, fmt.Errorf("cargo install --list: %w", err)
	}

	binDir, err := d.GlobalInstallDir(ctx)
	if err != nil {
		return nil, err
	}

	return parseCargoInstallList(out, binDir), nil
}

// parseCargoInstallList parses `cargo install --list`'s output, e.g.:
//
//	ripgrep v13.0.0:
//	    rg
//	delve v1.27.1:
//	    dlv
//
// returning one Entry per package, sized from its binaries in binDir.
func parseCargoInstallList(out []byte, binDir string) []Entry {
	var entries []Entry
	var name, version string
	var bins []string

	flush := func() {
		if name != "" {
			entries = append(entries, Entry{
				Name:    name,
				Version: version,
				Path:    binDir,
				Kind:    KindGlobalPackage,
				Size:    cargoBinSize(binDir, bins),
			})
		}
		name, version = "", ""
		bins = nil
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, " ") {
			if name != "" {
				bins = append(bins, strings.TrimSpace(line))
			}
			continue
		}
		flush()
		if m := cargoInstallListRe.FindStringSubmatch(line); m != nil {
			name, version = m[1], m[2]
		}
	}
	flush()
	return entries
}

// cargoBinSize returns -1 if none of bins were found. It falls back to
// a ".exe" suffix since `cargo install --list` omits it on Windows.
func cargoBinSize(binDir string, bins []string) int64 {
	var total int64
	var found bool
	for _, b := range bins {
		info, err := os.Stat(filepath.Join(binDir, b))
		if err != nil {
			info, err = os.Stat(filepath.Join(binDir, b+".exe"))
		}
		if err != nil {
			continue
		}
		found = true
		total += info.Size()
	}
	if !found {
		return -1
	}
	return total
}

// Match fields in each "[[package]]" block of a Cargo.lock file, e.g.:
//
//	[[package]]
//	name = "regex"
//	version = "1.10.2"
//	source = "registry+https://github.com/rust-lang/crates.io-index"
var (
	cargoLockPackageNameRe    = regexp.MustCompile(`^name\s*=\s*"(.+)"$`)
	cargoLockPackageVersionRe = regexp.MustCompile(`^version\s*=\s*"(.+)"$`)
	cargoLockPackageSourceRe  = regexp.MustCompile(`^source\s*=\s*".+"$`)
)

// LocalPackages skips workspace members and path dependencies (no
// "source" line, not fetched from a registry), since they aren't
// stored anywhere cacheriff could report on.
func (d cargoDriver) LocalPackages(ctx context.Context, root string) ([]Entry, error) {
	lockPath := filepath.Join(root, "Cargo.lock")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("cargo: read %s: %w", lockPath, err)
	}

	home, err := cargoHome()
	if err != nil {
		return nil, err
	}

	var entries []Entry
	var name, version string
	var hasSource bool
	flush := func() {
		if name != "" && version != "" && hasSource {
			p := cargoRegistrySrcPath(home, name, version)
			size := int64(-1)
			if p != "" {
				if s, err := dirSize(ctx, p); err == nil {
					size = s
				}
			}
			entries = append(entries, Entry{
				Name:    name,
				Version: version,
				Path:    p,
				Kind:    KindLocalPackage,
				Size:    size,
			})
		}
		name, version, hasSource = "", "", false
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case line == "[[package]]":
			flush()
		case cargoLockPackageSourceRe.MatchString(line):
			hasSource = true
		default:
			if m := cargoLockPackageNameRe.FindStringSubmatch(line); m != nil {
				name = m[1]
			} else if m := cargoLockPackageVersionRe.FindStringSubmatch(line); m != nil {
				version = m[1]
			}
		}
	}
	flush()
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("cargo: parse %s: %w", lockPath, err)
	}
	return entries, nil
}

// cargoRegistrySrcPath globs for name@version under registry/src
// rather than replicating cargo's registry-URL hashing scheme.
// Returns "" if no match is found.
func cargoRegistrySrcPath(home, name, version string) string {
	matches, err := filepath.Glob(filepath.Join(home, "registry", "src", "*", name+"-"+version))
	if err != nil || len(matches) == 0 {
		return ""
	}
	return matches[0]
}

func (d cargoDriver) Remove(ctx context.Context, e Entry) error {
	switch e.Kind {
	case KindCache:
		return os.RemoveAll(e.Path)
	case KindGlobalPackage:
		name, err := textwrap.EscapeArg(e.Name)
		if err != nil {
			return fmt.Errorf("cargo: %w", err)
		}
		return d.runCombined(ctx, "uninstall", name)
	case KindToolchain:
		name, err := textwrap.EscapeArg(e.Name)
		if err != nil {
			return fmt.Errorf("cargo: %w", err)
		}
		out, err := exec.CommandContext(ctx, "rustup", "toolchain", "uninstall", name).CombinedOutput()
		if err != nil {
			return fmt.Errorf("rustup toolchain uninstall %s: %w: %s", name, err, strings.TrimSpace(string(out)))
		}
		return nil
	default:
		return d.unsupportedKindErr(e.Kind)
	}
}
