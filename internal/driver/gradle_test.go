package driver

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGradleCacheDirEnvOverride(t *testing.T) {
	home := filepath.Join(t.TempDir(), "gh")
	t.Setenv("GRADLE_USER_HOME", home)
	got, err := gradleDriver{}.CacheDir(context.Background())
	if err != nil || got != filepath.Join(home, "caches") {
		t.Errorf("got %q, %v", got, err)
	}
}

func TestGradleCacheEntries(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GRADLE_USER_HOME", home)
	ctx := context.Background()

	if got, err := (gradleDriver{}).CacheEntries(ctx); err != nil || len(got) != 0 {
		t.Fatalf("empty home: got %v, %v", got, err)
	}

	caches := filepath.Join(home, "caches")
	if err := os.MkdirAll(caches, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(caches, "f"), []byte("12345"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := gradleDriver{}.CacheEntries(ctx)
	if err != nil || len(got) != 1 {
		t.Fatalf("got %v, %v", got, err)
	}
	if got[0].Name != "caches" || got[0].Size != 5 || got[0].Kind != KindCache {
		t.Errorf("unexpected entry %+v", got[0])
	}
}

func TestGradleRemove(t *testing.T) {
	d := NewGradleDriver()
	dir := filepath.Join(t.TempDir(), "caches")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := d.Remove(context.Background(), Entry{Kind: KindCache, Path: dir}); err != nil {
		t.Fatal(err)
	}
	if pathExists(dir) {
		t.Error("cache dir not removed")
	}
	if err := d.Remove(context.Background(), Entry{Kind: KindGlobalPackage}); err == nil {
		t.Error("expected error for unsupported kind")
	}
}
