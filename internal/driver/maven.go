package driver

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ro80t/cacheriff/internal/platform"
)

type mavenDriver struct {
	base
}

func NewMavenDriver() Driver {
	// Maven has neither a global-install-of-CLI-tools concept (unlike
	// cargo/go/deno) nor a per-project package copy (localDir left
	// unset): dependencies are just cached, by coordinate, in the
	// shared local repository below.
	return mavenDriver{base: base{
		id:          "maven",
		name:        "Maven",
		binary:      "mvn",
		supportedOS: []platform.OS{platform.Windows, platform.MacOS, platform.Linux},
		dirs:        []string{"target"},
	}}
}

type mavenSettings struct {
	LocalRepository string `xml:"localRepository"`
}

// mavenRepoDir honors ~/.m2/settings.xml's <localRepository> override,
// falling back to the default ~/.m2/repository.
func mavenRepoDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("maven: resolve home directory: %w", err)
	}
	if data, err := os.ReadFile(filepath.Join(home, ".m2", "settings.xml")); err == nil {
		var s mavenSettings
		if xml.Unmarshal(data, &s) == nil {
			if repo := strings.TrimSpace(s.LocalRepository); repo != "" {
				return repo, nil
			}
		}
	}
	return filepath.Join(home, ".m2", "repository"), nil
}

func (d mavenDriver) CacheDir(_ context.Context) (string, error) {
	return mavenRepoDir()
}

// CacheEntries reports one entry per top-level group directory (e.g.
// "com", "org", "junit") rather than the whole repository as one blob,
// since it holds every dependency ever downloaded for every project.
func (d mavenDriver) CacheEntries(ctx context.Context) ([]Entry, error) {
	repo, err := mavenRepoDir()
	if err != nil {
		return nil, err
	}
	des, err := os.ReadDir(repo)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("maven: read %s: %w", repo, err)
	}

	dirs := make([]namedDir, 0, len(des))
	for _, de := range des {
		if de.IsDir() {
			dirs = append(dirs, namedDir{name: de.Name(), path: filepath.Join(repo, de.Name())})
		}
	}
	return sizeCacheDirs(ctx, dirs), nil
}

func (d mavenDriver) GlobalInstallDir(_ context.Context) (string, error) {
	return mavenRepoDir()
}

// GlobalPackages always reports nothing: see the note in NewMavenDriver.
func (mavenDriver) GlobalPackages(_ context.Context) ([]Entry, error) {
	return nil, nil
}

// LocalPackages always reports nothing: see the note in NewMavenDriver.
func (mavenDriver) LocalPackages(_ context.Context, _ string) ([]Entry, error) {
	return nil, nil
}

func (d mavenDriver) Remove(_ context.Context, e Entry) error {
	switch e.Kind {
	case KindCache:
		return os.RemoveAll(e.Path)
	default:
		return d.unsupportedKindErr(e.Kind)
	}
}
