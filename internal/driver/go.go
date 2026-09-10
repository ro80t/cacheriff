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

func NewGoDriver() Driver {
	// localDir is left unset: Go has no per-project install
	// directory; dependencies live in the shared GOMODCACHE.
	return goDriver{base: base{
		id:          "go",
		name:        "Go",
		binary:      "go",
		supportedOS: []platform.OS{platform.Windows, platform.MacOS, platform.Linux},
	}}
}

func (d goDriver) CacheDir(ctx context.Context) (string, error) {
	return d.runOutput(ctx, "env", "GOCACHE")
}

// Cache entry names are also used by Remove to pick which `go clean`
// flag applies, since GOCACHE and GOMODCACHE need different ones.
const (
	goBuildCacheName = "Build cache (GOCACHE)"
	goModCacheName   = "Module cache (GOMODCACHE)"
)

func (d goDriver) CacheEntries(ctx context.Context) ([]Entry, error) {
	buildCache, err := d.runOutput(ctx, "env", "GOCACHE")
	if err != nil {
		return nil, err
	}
	modCache, err := d.runOutput(ctx, "env", "GOMODCACHE")
	if err != nil {
		return nil, err
	}

	return sizeCacheDirs(ctx, []namedDir{
		{goBuildCacheName, buildCache},
		{goModCacheName, modCache},
	}), nil
}

func (d goDriver) GlobalInstallDir(ctx context.Context) (string, error) {
	if bin, err := d.runOutput(ctx, "env", "GOBIN"); err == nil && bin != "" {
		return bin, nil
	}
	gopath, err := d.runOutput(ctx, "env", "GOPATH")
	if err != nil {
		return "", err
	}
	return filepath.Join(gopath, "bin"), nil
}

// GlobalPackages scans GOBIN (or GOPATH/bin) and runs `go version -m`
// across every file there, since go has no command that lists
// installed binaries directly.
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

	// Same non-zero-exit-but-valid-output caveat as npm ls -g.
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

// goModRequire is one entry of `go mod edit -json`'s "Require" list.
type goModRequire struct {
	Path     string
	Version  string
	Indirect bool
}

func (d goDriver) LocalPackages(ctx context.Context, root string) ([]Entry, error) {
	if !pathExists(filepath.Join(root, "go.mod")) {
		return nil, nil
	}

	modCache, err := d.runOutput(ctx, "env", "GOMODCACHE")
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
		size, err := dirSize(ctx, p)
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
// encoding (golang.org/x/mod/module.EscapePath): each uppercase
// letter becomes "!" + its lowercase form, e.g.
// ".../github.com/!burnt!sushi/toml@v1.5.0".
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

func (d goDriver) Remove(ctx context.Context, e Entry) error {
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
		return d.runCombined(ctx, "clean", flag)
	case KindGlobalPackage:
		// Go has no built-in "uninstall": the documented way to
		// remove a globally installed tool is to delete its binary
		// from GOBIN/GOPATH/bin directly.
		if err := os.Remove(e.Path); err != nil {
			return fmt.Errorf("remove %s: %w", e.Path, err)
		}
		return nil
	default:
		return d.unsupportedKindErr(e.Kind)
	}
}
