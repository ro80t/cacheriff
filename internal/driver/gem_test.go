package driver

import (
	"path/filepath"
	"testing"
)

func TestParseGemListOutput(t *testing.T) {
	// Real `gem list --local` output (trimmed), including default gems
	// bundled with the Ruby install that must be skipped.
	out := `abbrev (default: 0.1.1)
base64 (default: 0.1.1)
bundler (default: 2.4.10)
debug (1.7.1)
httpclient (2.8.3)
rake (13.0.6)
`
	gemsDir := filepath.Join("C:", "gems")
	entries := parseGemListOutput(out, gemsDir)

	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3 (default gems skipped): %+v", len(entries), entries)
	}
	want := map[string]string{
		"debug":      "1.7.1",
		"httpclient": "2.8.3",
		"rake":       "13.0.6",
	}
	for _, e := range entries {
		v, ok := want[e.Name]
		if !ok {
			t.Errorf("unexpected entry %q", e.Name)
			continue
		}
		if e.Version != v {
			t.Errorf("%s: got version %q, want %q", e.Name, e.Version, v)
		}
		wantPath := filepath.Join(gemsDir, e.Name+"-"+v)
		if e.Path != wantPath {
			t.Errorf("%s: got path %q, want %q", e.Name, e.Path, wantPath)
		}
	}
}

func TestParseGemListOutputMultipleVersions(t *testing.T) {
	out := "bundler (2.4.10, 2.3.7)\n"
	entries := parseGemListOutput(out, "/gems")
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2: %+v", len(entries), entries)
	}
	if entries[0].Version != "2.4.10" || entries[1].Version != "2.3.7" {
		t.Errorf("got versions %q, %q, want 2.4.10, 2.3.7", entries[0].Version, entries[1].Version)
	}
}
