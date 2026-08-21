package driver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSplitPackageSpec(t *testing.T) {
	tests := []struct {
		spec        string
		wantName    string
		wantVersion string
	}{
		{"left-pad@1.3.0", "left-pad", "1.3.0"},
		{"@types/node@20.11.5", "@types/node", "20.11.5"},
		{"@scope/pkg@1.0.0-beta.1", "@scope/pkg", "1.0.0-beta.1"},
		{"no-version", "no-version", ""},
		{"@justscope", "@justscope", ""},
	}
	for _, tt := range tests {
		name, version := splitPackageSpec(tt.spec)
		if name != tt.wantName || version != tt.wantVersion {
			t.Errorf("splitPackageSpec(%q) = (%q, %q), want (%q, %q)", tt.spec, name, version, tt.wantName, tt.wantVersion)
		}
	}
}

func TestResolvedPackageVersion(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"x","version":"2.3.4"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := resolvedPackageVersion(dir); got != "2.3.4" {
		t.Errorf("got %q, want 2.3.4", got)
	}
}

func TestResolvedPackageVersionMissing(t *testing.T) {
	if got := resolvedPackageVersion(t.TempDir()); got != "" {
		t.Errorf("got %q, want empty string for a package.json that doesn't exist", got)
	}
}
