package driver

import (
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

type gemDriver struct {
	base
}

func NewGemDriver() Driver {
	// localDir is left unset: gems install into the shared GEM_HOME
	// rather than into the project, unless the user opts into
	// `bundle install --path`, which isn't the default.
	return gemDriver{base: base{
		id:          "gem",
		name:        "RubyGems",
		binary:      "gem",
		supportedOS: []platform.OS{platform.Windows, platform.MacOS, platform.Linux},
		dirs:        []string{"vendor/bundle"},
	}}
}

func (d gemDriver) gemDir(ctx context.Context) (string, error) {
	return d.runOutput(ctx, "environment", "gemdir")
}

func (d gemDriver) CacheDir(ctx context.Context) (string, error) {
	dir, err := d.gemDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "cache"), nil
}

func (d gemDriver) CacheEntries(ctx context.Context) ([]Entry, error) {
	dir, err := d.CacheDir(ctx)
	if err != nil {
		return nil, err
	}
	return d.singleDirCacheEntries(ctx, dir, "gem cache")
}

func (d gemDriver) GlobalInstallDir(ctx context.Context) (string, error) {
	dir, err := d.gemDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gems"), nil
}

// gemListLineRe matches a line of `gem list --local`, e.g.
// "rake (13.0.6)" or "bundler (2.4.10, 2.3.7)".
var gemListLineRe = regexp.MustCompile(`^(\S+)\s+\((.+)\)$`)

func (d gemDriver) GlobalPackages(ctx context.Context) ([]Entry, error) {
	gemsDir, err := d.GlobalInstallDir(ctx)
	if err != nil {
		return nil, err
	}
	out, err := exec.CommandContext(ctx, "gem", "list", "--local").Output()
	if err != nil {
		return nil, fmt.Errorf("gem list --local: %w", err)
	}
	return parseGemListOutput(string(out), gemsDir), nil
}

// parseGemListOutput skips gems bundled with the Ruby install itself
// (marked "(default: x.y.z)"), since those aren't user-managed and
// `gem uninstall` refuses to remove them.
func parseGemListOutput(out, gemsDir string) []Entry {
	var entries []Entry
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		m := gemListLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name, versions := m[1], m[2]
		if strings.HasPrefix(versions, "default:") {
			continue
		}
		for v := range strings.SplitSeq(versions, ", ") {
			entries = append(entries, Entry{
				Name:    name,
				Version: v,
				Path:    filepath.Join(gemsDir, name+"-"+v),
				Kind:    KindGlobalPackage,
			})
		}
	}
	return entries
}

// LocalPackages always reports nothing: see localDir above.
func (gemDriver) LocalPackages(_ context.Context, _ string) ([]Entry, error) {
	return nil, nil
}

func (d gemDriver) Remove(ctx context.Context, e Entry) error {
	switch e.Kind {
	case KindCache:
		return os.RemoveAll(e.Path)
	case KindGlobalPackage:
		name, err := textwrap.EscapeArg(e.Name)
		if err != nil {
			return fmt.Errorf("gem: %w", err)
		}
		return d.runCombined(ctx, "uninstall", name, "--version", e.Version, "-x")
	default:
		return d.unsupportedKindErr(e.Kind)
	}
}
