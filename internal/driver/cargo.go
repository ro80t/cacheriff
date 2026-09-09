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
	"cacheriff/internal/textwrap"
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
		// localDir is left unset: cargo has no per-project package
		// install directory. Resolved dependencies are downloaded once
		// into CARGO_HOME's shared registry (see cargoCacheDirs) and
		// referenced from there directly, rather than being copied into
		// the project like npm's node_modules.
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

// CacheEntries sizes cargoCacheDirs concurrently: registry/src in
// particular can hold thousands of small files, since every
// dependency ever built gets its own extracted source tree, so
// walking them one after another would be slow.
func (d cargoDriver) CacheEntries(ctx context.Context) ([]Entry, error) {
	home, err := cargoHome()
	if err != nil {
		return nil, err
	}

	dirs := make([]namedDir, len(cargoCacheDirs))
	for i, c := range cargoCacheDirs {
		dirs[i] = namedDir{name: c.name, path: filepath.Join(home, c.rel)}
	}
	return sizeCacheDirs(ctx, dirs), nil
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
// returning one Entry per package. The indented lines under each
// header name that package's installed binaries; parseCargoInstallList
// stats them in binDir to size the entry, rather than leaving it
// unmeasured, since (unlike the registry caches) cargo has no separate
// per-package directory to size instead.
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

// cargoBinSize sums the on-disk size of a package's installed binaries
// within binDir. On Windows the binary file carries an additional
// ".exe" suffix that `cargo install --list` doesn't include, so that's
// tried as a fallback. Returns -1 if none of the binaries could be
// found (e.g. the package installed no binaries, or they were removed
// outside of cargo).
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

// cargoLockPackageNameRe, cargoLockPackageVersionRe, and
// cargoLockPackageSourceRe pull the fields LocalPackages needs out of
// each "[[package]]" block in a Cargo.lock file, e.g.:
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

// LocalPackages reports the dependencies pinned in root's Cargo.lock.
// Workspace members and path dependencies (which have no "source"
// line, since they're not fetched from a registry) are skipped, since
// they aren't stored anywhere cacheriff could report on. Each
// dependency's Path points at its extracted source under CARGO_HOME's
// shared registry cache, when it can be found there.
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

// cargoRegistrySrcPath finds the extracted source directory for
// name@version under CARGO_HOME's registry/src cache, without needing
// to replicate cargo's registry-URL hashing scheme. Returns "" if no
// match is found (e.g. the crate hasn't been fetched/extracted yet).
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
	default:
		return d.unsupportedKindErr(e.Kind)
	}
}
