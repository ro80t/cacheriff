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

type goDriver struct {
	base
}

// NewGoDriver returns the Driver for the Go toolchain.
func NewGoDriver() Driver {
	return goDriver{base: base{
		id:          "go",
		name:        "Go",
		binary:      "go",
		supportedOS: []platform.OS{platform.Windows, platform.MacOS, platform.Linux},
	}}
}

func goEnv(ctx context.Context, key string) (string, error) {
	out, err := exec.CommandContext(ctx, "go", "env", key).Output()
	if err != nil {
		return "", fmt.Errorf("go env %s: %w", key, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (d goDriver) CacheDir(ctx context.Context) (string, error) {
	return goEnv(ctx, "GOCACHE")
}

// Cache entry names are also used by Remove to pick which `go clean`
// flag applies, since GOCACHE and GOMODCACHE need different ones.
const (
	goBuildCacheName = "Build cache (GOCACHE)"
	goModCacheName   = "Module cache (GOMODCACHE)"
)

func (d goDriver) CacheEntries(ctx context.Context) ([]Entry, error) {
	buildCache, err := goEnv(ctx, "GOCACHE")
	if err != nil {
		return nil, err
	}
	modCache, err := goEnv(ctx, "GOMODCACHE")
	if err != nil {
		return nil, err
	}

	dirs := []struct{ name, path string }{
		{goBuildCacheName, buildCache},
		{goModCacheName, modCache},
	}

	var entries []Entry
	for _, dd := range dirs {
		if dd.path == "" || !pathExists(dd.path) {
			continue
		}
		size, err := dirSize(dd.path)
		if err != nil {
			size = -1
		}
		entries = append(entries, Entry{
			Name: dd.name,
			Path: dd.path,
			Kind: KindCache,
			Size: size,
		})
	}
	return entries, nil
}

func (d goDriver) GlobalInstallDir(ctx context.Context) (string, error) {
	if bin, err := goEnv(ctx, "GOBIN"); err == nil && bin != "" {
		return bin, nil
	}
	gopath, err := goEnv(ctx, "GOPATH")
	if err != nil {
		return "", err
	}
	return filepath.Join(gopath, "bin"), nil
}

// GlobalPackages reports the binaries `go install`ed into GOBIN (or
// GOPATH/bin): go has no single command that lists them, so this
// scans that directory and runs `go version -m` across every file
// there, which reads each binary's embedded build info (module path,
// resolved version) directly - no guessing needed, since the Go
// toolchain stamps this into every module-aware binary it builds.
func (d goDriver) GlobalPackages(ctx context.Context) ([]Entry, error) {
	binDir, err := d.GlobalInstallDir(ctx)
	if err != nil {
		return nil, err
	}
	if !pathExists(binDir) {
		return nil, nil
	}

	dirEntries, err := os.ReadDir(binDir)
	if err != nil {
		return nil, fmt.Errorf("go: read %s: %w", binDir, err)
	}
	var paths []string
	for _, de := range dirEntries {
		if !de.IsDir() {
			paths = append(paths, filepath.Join(binDir, de.Name()))
		}
	}
	if len(paths) == 0 {
		return nil, nil
	}

	// `go version -m` exits non-zero if any one of the given files
	// isn't a Go binary it can read build info from (e.g. stray
	// non-Go junk in the bin dir), but it still prints results for
	// every file that did work, so - like npm ls -g - only the
	// output matters here, not the exit status.
	args := append([]string{"version", "-m"}, paths...)
	out, _ := exec.CommandContext(ctx, "go", args...).Output()
	return parseGoVersionM(out), nil
}

// parseGoVersionM parses `go version -m`'s output across one or more
// binaries, e.g.:
//
//	/path/to/dlv: go1.26.5
//		path	github.com/go-delve/delve/cmd/dlv
//		mod	github.com/go-delve/delve	v1.27.1	h1:...=
//		dep	github.com/cilium/ebpf	v0.11.0	h1:...=
//		build	-buildmode=exe
//
// returning one Entry per binary whose main module (the first "mod"
// line) it could identify.
func parseGoVersionM(out []byte) []Entry {
	var entries []Entry
	var binPath, version string

	flush := func() {
		if binPath == "" || version == "" {
			return
		}
		name := strings.TrimSuffix(filepath.Base(binPath), ".exe")
		size := int64(-1)
		if info, err := os.Stat(binPath); err == nil {
			size = info.Size()
		}
		entries = append(entries, Entry{
			Name:    name,
			Version: version,
			Path:    binPath,
			Kind:    KindGlobalPackage,
			Size:    size,
		})
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "\t") {
			flush()
			binPath, version = "", ""
			if i := strings.LastIndex(line, ": "); i >= 0 {
				binPath = line[:i]
			}
			continue
		}
		if version != "" {
			continue // already have this binary's main module
		}
		fields := strings.Split(strings.TrimPrefix(line, "\t"), "\t")
		if len(fields) >= 3 && fields[0] == "mod" {
			version = fields[2]
		}
	}
	flush()
	return entries
}

// LocalInstallDir reports that Go has no per-project package install
// directory: resolved dependencies are downloaded once into the
// shared GOMODCACHE and referenced from there directly (like cargo's
// shared registry), never copied into the project.
func (goDriver) LocalInstallDir(_ string) (string, bool) {
	return "", false
}

// goModRequire is one entry of `go mod edit -json`'s "Require" list.
type goModRequire struct {
	Path     string
	Version  string
	Indirect bool
}

// LocalPackages reports root's direct (non-indirect) module
// requirements, resolved to their extracted location under
// GOMODCACHE.
func (d goDriver) LocalPackages(ctx context.Context, root string) ([]Entry, error) {
	if !pathExists(filepath.Join(root, "go.mod")) {
		return nil, nil
	}

	modCache, err := goEnv(ctx, "GOMODCACHE")
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, "go", "mod", "edit", "-json")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go mod edit -json: %w", err)
	}

	var parsed struct {
		Require []goModRequire
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("go mod edit -json: parse output: %w", err)
	}

	var entries []Entry
	for _, r := range parsed.Require {
		if r.Indirect {
			continue
		}
		p := filepath.Join(modCache, escapeModulePath(r.Path)+"@"+escapeModulePath(r.Version))
		size, err := dirSize(p)
		if err != nil {
			size = -1
		}
		entries = append(entries, Entry{
			Name:    r.Path,
			Version: r.Version,
			Path:    p,
			Kind:    KindLocalPackage,
			Size:    size,
		})
	}
	return entries, nil
}

// escapeModulePath implements Go's module cache "escaped path"
// encoding (see golang.org/x/mod/module.EscapePath): every uppercase
// letter is replaced with "!" followed by its lowercase form, since
// GOMODCACHE must work on case-insensitive filesystems too. The same
// encoding applies to a module cache entry's version component (e.g.
// ".../github.com/!burnt!sushi/toml@v1.5.0").
func escapeModulePath(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			b.WriteByte('!')
			r += 'a' - 'A'
		}
		b.WriteRune(r)
	}
	return b.String()
}

func (goDriver) Remove(ctx context.Context, e Entry) error {
	switch e.Kind {
	case KindCache:
		var flag string
		switch e.Name {
		case goBuildCacheName:
			flag = "-cache"
		case goModCacheName:
			flag = "-modcache"
		default:
			return fmt.Errorf("go: unknown cache entry %q", e.Name)
		}
		out, err := exec.CommandContext(ctx, "go", "clean", flag).CombinedOutput()
		if err != nil {
			return fmt.Errorf("go clean %s: %w: %s", flag, err, strings.TrimSpace(string(out)))
		}
		return nil
	case KindGlobalPackage:
		// Go has no built-in "uninstall": the documented way to
		// remove a globally installed tool is to delete its binary
		// from GOBIN/GOPATH/bin directly.
		if err := os.Remove(e.Path); err != nil {
			return fmt.Errorf("remove %s: %w", e.Path, err)
		}
		return nil
	default:
		return fmt.Errorf("go: unsupported entry kind %s", e.Kind)
	}
}
