package driver

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestUvRemove(t *testing.T) {
	d := NewUvDriver()
	dir := filepath.Join(t.TempDir(), "cache")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := d.Remove(context.Background(), Entry{Kind: KindCache, Path: dir}); err != nil {
		t.Fatal(err)
	}
	if pathExists(dir) {
		t.Error("cache dir not removed")
	}
	if err := d.Remove(context.Background(), Entry{Kind: KindLocalPackage}); err == nil {
		t.Error("expected error for unsupported kind")
	}
}

func TestUvLocalInstallDir(t *testing.T) {
	got, ok := NewUvDriver().LocalInstallDir("proj")
	if !ok || got != filepath.Join("proj", ".venv") {
		t.Errorf("got %q, %v", got, ok)
	}
}
