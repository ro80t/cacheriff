package driver

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParseYarnListTree(t *testing.T) {
	// Real (NDJSON) shape of `yarn list --depth=0 --json`'s output,
	// with warning lines interspersed the way yarn actually emits them.
	out := []byte(`{"type":"warning","data":"package.json: No license field"}
{"type":"tree","data":{"type":"list","trees":[{"name":"left-pad@1.3.0","children":[],"hint":null,"color":null,"depth":0},{"name":"@types/node@20.11.5","children":[],"hint":null,"color":null,"depth":0}]}}
`)

	specs, err := parseYarnListTree(out)
	if err != nil {
		t.Fatalf("parseYarnListTree: %v", err)
	}
	want := []string{"left-pad@1.3.0", "@types/node@20.11.5"}
	if len(specs) != len(want) {
		t.Fatalf("got %v, want %v", specs, want)
	}
	for i, w := range want {
		if specs[i] != w {
			t.Errorf("specs[%d] = %q, want %q", i, specs[i], w)
		}
	}
}

func TestParseYarnListTreeNoTreeLine(t *testing.T) {
	out := []byte(`{"type":"progressStart","data":{"id":0,"total":0}}
`)
	specs, err := parseYarnListTree(out)
	if err != nil {
		t.Fatalf("parseYarnListTree: %v", err)
	}
	if specs != nil {
		t.Errorf("got %v, want nil", specs)
	}
}

func TestPackagesFromGlobalManifest(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies":{"is-odd":"^3.0.1"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	pkgDir := filepath.Join(dir, "node_modules", "is-odd")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name":"is-odd","version":"3.0.1"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := packagesFromGlobalManifest(context.Background(), dir, KindGlobalPackage)
	if err != nil {
		t.Fatalf("packagesFromGlobalManifest: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1: %+v", len(entries), entries)
	}
	if entries[0].Name != "is-odd" || entries[0].Version != "3.0.1" {
		t.Errorf("got %+v, want name=is-odd version=3.0.1", entries[0])
	}
}

func TestPackagesFromGlobalManifestMissing(t *testing.T) {
	entries, err := packagesFromGlobalManifest(context.Background(), t.TempDir(), KindGlobalPackage)
	if err != nil {
		t.Fatalf("packagesFromGlobalManifest: %v", err)
	}
	if entries != nil {
		t.Errorf("got %+v, want nil when there's no package.json", entries)
	}
}
