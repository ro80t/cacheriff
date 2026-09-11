package driver

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/ro80t/cacheriff/internal/platform"
	"github.com/ro80t/cacheriff/internal/textwrap"
)

type pipDriver struct {
	base
}

func NewPipDriver() Driver {
	// localDir is left unset: pip installs into whichever Python
	// environment is active (global or a venv anywhere), never into a
	// fixed directory under the project root the way npm does.
	return pipDriver{base: base{
		id:          "pip",
		name:        "pip",
		binary:      "pip3",
		supportedOS: []platform.OS{platform.Windows, platform.MacOS, platform.Linux},
		dirs:        []string{"venv", ".venv"},
	}}
}

func (pipDriver) Available() bool {
	return len(discoverPipInstallations()) > 0
}

// pipInstallation is one pip executable, found by scanning every PATH
// entry (unlike exec.LookPath, which stops at the first match) plus,
// on Windows, the py launcher (pip_pylauncher.go).
type pipInstallation struct {
	bin string
}

func discoverPipInstallations() []pipInstallation {
	var installs []pipInstallation
	seenDirs := make(map[string]bool)
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		addPipInstallation(&installs, seenDirs, dir)
	}
	if platform.IsWindows() {
		for _, dir := range discoverPyLauncherPythonDirs() {
			addPipInstallation(&installs, seenDirs, dir)
		}
	}
	return installs
}

func discoverPipInstallationsIn(pathEnv string) []pipInstallation {
	var installs []pipInstallation
	seenDirs := make(map[string]bool)
	for _, dir := range filepath.SplitList(pathEnv) {
		addPipInstallation(&installs, seenDirs, dir)
	}
	return installs
}

// dir is skipped if already seen: PATH and the py launcher can point
// at the same directory.
func addPipInstallation(installs *[]pipInstallation, seenDirs map[string]bool, dir string) {
	if dir == "" {
		return
	}
	dir = filepath.Clean(dir)
	key := dir
	if platform.IsWindows() {
		key = strings.ToLower(dir)
	}
	if seenDirs[key] {
		return
	}

	names := []string{"pip3", "pip"}
	if platform.IsWindows() {
		names = []string{"pip3.exe", "pip.exe"}
	}
	for _, name := range names {
		p := filepath.Join(dir, name)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			*installs = append(*installs, pipInstallation{bin: p})
			seenDirs[key] = true
			return
		}
	}
}

// pip --version reports ".../site-packages/pip", one level below where
// every installed package actually lives.
var pipVersionRe = regexp.MustCompile(`^pip \S+ from (.+) \(python (\S+)\)`)

type pipInfo struct {
	siteDir string
	version string // Python version, e.g. "3.13"
}

func fetchPipInfo(ctx context.Context, bin string) (pipInfo, error) {
	out, err := exec.CommandContext(ctx, bin, "--version").Output()
	if err != nil {
		return pipInfo{}, fmt.Errorf("%s --version: %w", bin, err)
	}
	m := pipVersionRe.FindStringSubmatch(strings.TrimSpace(string(out)))
	if m == nil {
		return pipInfo{}, fmt.Errorf("%s --version: unexpected output: %s", bin, out)
	}
	return pipInfo{siteDir: filepath.Dir(m[1]), version: m[2]}, nil
}

func fetchPipCacheDir(ctx context.Context, bin string) (string, error) {
	out, err := exec.CommandContext(ctx, bin, "cache", "dir").Output()
	if err != nil {
		return "", fmt.Errorf("%s cache dir: %w", bin, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (d pipDriver) CacheDir(ctx context.Context) (string, error) {
	installs := discoverPipInstallations()
	if len(installs) == 0 {
		return "", fmt.Errorf("pip: no installation found on PATH")
	}
	return fetchPipCacheDir(ctx, installs[0].bin)
}

// A broken installation is skipped rather than failing the whole scan.
func (d pipDriver) CacheEntries(ctx context.Context) ([]Entry, error) {
	var entries []Entry
	for _, inst := range discoverPipInstallations() {
		info, err := fetchPipInfo(ctx, inst.bin)
		if err != nil {
			continue
		}
		dir, err := fetchPipCacheDir(ctx, inst.bin)
		if err != nil {
			continue
		}
		es, err := d.singleDirCacheEntries(ctx, dir, "pip cache (Python "+info.version+")")
		if err != nil {
			continue
		}
		entries = append(entries, es...)
	}
	return entries, nil
}

func (d pipDriver) GlobalInstallDir(ctx context.Context) (string, error) {
	installs := discoverPipInstallations()
	if len(installs) == 0 {
		return "", fmt.Errorf("pip: no installation found on PATH")
	}
	info, err := fetchPipInfo(ctx, installs[0].bin)
	if err != nil {
		return "", err
	}
	return info.siteDir, nil
}

type pipPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Each Entry's Path is its own site-packages dir, so Remove can find
// the right pip to uninstall with.
func (d pipDriver) GlobalPackages(ctx context.Context) ([]Entry, error) {
	var entries []Entry
	for _, inst := range discoverPipInstallations() {
		info, err := fetchPipInfo(ctx, inst.bin)
		if err != nil {
			continue
		}
		out, err := exec.CommandContext(ctx, inst.bin, "list", "--format=json").Output()
		if err != nil {
			continue
		}
		var parsed []pipPackage
		if err := json.Unmarshal(out, &parsed); err != nil {
			continue
		}
		for _, p := range parsed {
			entries = append(entries, Entry{
				Name:    p.Name,
				Version: p.Version,
				Path:    info.siteDir,
				Kind:    KindGlobalPackage,
				Size:    pipPackageSize(info.siteDir, p.Name, p.Version),
			})
		}
	}
	return entries, nil
}

// -, _, and . are equivalent in a distribution name (PEP 503).
var pipNameSepRe = regexp.MustCompile(`[-_.]+`)

func pipNormalizeName(s string) string {
	return pipNameSepRe.ReplaceAllString(strings.ToLower(s), "-")
}

// Sums the RECORD file pip itself reads to uninstall a package, rather
// than guessing its installed module directory name (e.g. PyYAML
// installs as "yaml").
func pipPackageSize(siteDir, name, version string) int64 {
	matches, err := filepath.Glob(filepath.Join(siteDir, "*-"+version+".dist-info"))
	if err != nil {
		return -1
	}
	target := pipNormalizeName(name)
	for _, m := range matches {
		base := strings.TrimSuffix(filepath.Base(m), ".dist-info")
		n := strings.TrimSuffix(base, "-"+version)
		if pipNormalizeName(n) != target {
			continue
		}
		size, err := pipRecordSize(filepath.Join(m, "RECORD"))
		if err != nil {
			return -1
		}
		return size
	}
	return -1
}

// RECORD lines are "path,hash,size"; hash/size are empty for RECORD's
// own entry.
func pipRecordSize(path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1

	var total int64
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		if len(rec) < 3 || rec[2] == "" {
			continue
		}
		n, err := strconv.ParseInt(rec[2], 10, 64)
		if err != nil {
			continue
		}
		total += n
	}
	return total, nil
}

// LocalPackages always reports nothing: pip has no per-project install
// directory the way npm has node_modules -- it installs into whatever
// Python environment (global or a venv) is active regardless of the
// project root.
func (pipDriver) LocalPackages(_ context.Context, _ string) ([]Entry, error) {
	return nil, nil
}

// pipBinForSiteDir re-resolves which installation owns a site-packages
// directory, since Entry carries no field naming its source pip.
func pipBinForSiteDir(ctx context.Context, siteDir string) (string, error) {
	for _, inst := range discoverPipInstallations() {
		info, err := fetchPipInfo(ctx, inst.bin)
		if err != nil {
			continue
		}
		if info.siteDir == siteDir {
			return inst.bin, nil
		}
	}
	return "", fmt.Errorf("pip: no installation found for %s", siteDir)
}

func (d pipDriver) Remove(ctx context.Context, e Entry) error {
	switch e.Kind {
	case KindCache:
		return os.RemoveAll(e.Path)
	case KindGlobalPackage:
		name, err := textwrap.EscapeArg(e.Name)
		if err != nil {
			return fmt.Errorf("pip: %w", err)
		}
		bin, err := pipBinForSiteDir(ctx, e.Path)
		if err != nil {
			return err
		}
		out, err := exec.CommandContext(ctx, bin, "uninstall", "-y", name).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s uninstall -y %s: %w: %s", bin, name, err, strings.TrimSpace(string(out)))
		}
		return nil
	default:
		return d.unsupportedKindErr(e.Kind)
	}
}
