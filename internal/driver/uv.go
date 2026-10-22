package driver

import (
	"context"
	"os"

	"github.com/ro80t/cacheriff/internal/platform"
)

type uvDriver struct {
	base
}

func NewUvDriver() Driver {
	return uvDriver{base: base{
		id:          "uv",
		name:        "uv",
		binary:      "uv",
		supportedOS: []platform.OS{platform.Windows, platform.MacOS, platform.Linux},
		dirs:        []string{".venv"},
		localDir:    ".venv",
	}}
}

func (d uvDriver) CacheDir(ctx context.Context) (string, error) {
	return d.runOutput(ctx, "cache", "dir")
}

func (d uvDriver) CacheEntries(ctx context.Context) ([]Entry, error) {
	dir, err := d.CacheDir(ctx)
	if err != nil {
		return nil, err
	}
	return d.singleDirCacheEntries(ctx, dir, "uv cache")
}

func (d uvDriver) GlobalInstallDir(ctx context.Context) (string, error) {
	return d.runOutput(ctx, "tool", "dir")
}

func (uvDriver) GlobalPackages(_ context.Context) ([]Entry, error) { return nil, nil }

func (uvDriver) LocalPackages(_ context.Context, _ string) ([]Entry, error) { return nil, nil }

func (d uvDriver) Remove(_ context.Context, e Entry) error {
	if e.Kind != KindCache {
		return d.unsupportedKindErr(e.Kind)
	}
	return os.RemoveAll(e.Path)
}
