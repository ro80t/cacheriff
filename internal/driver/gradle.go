package driver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ro80t/cacheriff/internal/platform"
)

type gradleDriver struct {
	base
}

func NewGradleDriver() Driver {
	// Like Maven, Gradle keeps dependencies only in a shared cache
	// (localDir unset) and has no global-install concept.
	return gradleDriver{base: base{
		id:          "gradle",
		name:        "Gradle",
		binary:      "gradle",
		supportedOS: []platform.OS{platform.Windows, platform.MacOS, platform.Linux},
		dirs:        []string{".gradle", "build"},
	}}
}

func gradleHome() (string, error) {
	if h := os.Getenv("GRADLE_USER_HOME"); h != "" {
		return h, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("gradle: resolve home directory: %w", err)
	}
	return filepath.Join(home, ".gradle"), nil
}

func (gradleDriver) CacheDir(_ context.Context) (string, error) {
	h, err := gradleHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, "caches"), nil
}

func (gradleDriver) CacheEntries(ctx context.Context) ([]Entry, error) {
	h, err := gradleHome()
	if err != nil {
		return nil, err
	}
	return sizeCacheDirs(ctx, []namedDir{
		{"caches", filepath.Join(h, "caches")},
		{"wrapper/dists", filepath.Join(h, "wrapper", "dists")},
	}), nil
}

func (gradleDriver) GlobalInstallDir(_ context.Context) (string, error) {
	return gradleHome()
}

func (gradleDriver) GlobalPackages(_ context.Context) ([]Entry, error) { return nil, nil }

func (gradleDriver) LocalPackages(_ context.Context, _ string) ([]Entry, error) { return nil, nil }

func (d gradleDriver) Remove(_ context.Context, e Entry) error {
	if e.Kind != KindCache {
		return d.unsupportedKindErr(e.Kind)
	}
	return os.RemoveAll(e.Path)
}
