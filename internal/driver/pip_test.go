package driver

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ro80t/cacheriff/internal/platform"
)

func TestPipVersionRe(t *testing.T) {
	tests := []struct {
		name     string
		out      string
		wantDir  string
		wantVers string
	}{
		{
			name:     "windows",
			out:      `pip 25.1.1 from C:\Users\tkata\AppData\Local\Programs\Python\Python313\Lib\site-packages\pip (python 3.13)`,
			wantDir:  `C:\Users\tkata\AppData\Local\Programs\Python\Python313\Lib\site-packages\pip`,
			wantVers: "3.13",
		},
		{
			name:     "linux",
			out:      "pip 23.3.1 from /usr/lib/python3/dist-packages/pip (python 3.11)",
			wantDir:  "/usr/lib/python3/dist-packages/pip",
			wantVers: "3.11",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := pipVersionRe.FindStringSubmatch(tt.out)
			if m == nil {
				t.Fatalf("no match for %q", tt.out)
			}
			if m[1] != tt.wantDir {
				t.Errorf("dir: got %q, want %q", m[1], tt.wantDir)
			}
			if m[2] != tt.wantVers {
				t.Errorf("version: got %q, want %q", m[2], tt.wantVers)
			}
		})
	}
}

func TestDiscoverPipInstallationsIn(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	dir3 := t.TempDir() // no pip here

	primary, secondary := "pip3", "pip"
	if platform.IsWindows() {
		primary, secondary = "pip3.exe", "pip.exe"
	}

	// dir1 has both names: the higher-priority one should win, and the
	// directory should only be reported once.
	mustCreate(t, filepath.Join(dir1, primary))
	mustCreate(t, filepath.Join(dir1, secondary))
	// dir2 has only the lower-priority name.
	mustCreate(t, filepath.Join(dir2, secondary))

	pathEnv := strings.Join([]string{dir1, dir2, dir1, dir3}, string(os.PathListSeparator))
	got := discoverPipInstallationsIn(pathEnv)

	want := []string{filepath.Join(dir1, primary), filepath.Join(dir2, secondary)}
	if len(got) != len(want) {
		t.Fatalf("got %d installs, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].bin != w {
			t.Errorf("install %d: got %q, want %q", i, got[i].bin, w)
		}
	}
}

func mustCreate(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, nil, 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestPipListJSON(t *testing.T) {
	// Real `pip list --format=json` output.
	out := `[{"name": "pip", "version": "25.1.1"}]`
	var parsed []pipPackage
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(parsed) != 1 || parsed[0].Name != "pip" || parsed[0].Version != "25.1.1" {
		t.Errorf("got %+v, want [{pip 25.1.1}]", parsed)
	}
}

func TestPipNormalizeName(t *testing.T) {
	tests := map[string]string{
		"PyYAML":            "pyyaml",
		"typing_extensions": "typing-extensions",
		"zope.interface":    "zope-interface",
		"requests":          "requests",
	}
	for in, want := range tests {
		if got := pipNormalizeName(in); got != want {
			t.Errorf("pipNormalizeName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPipPackageSize(t *testing.T) {
	dir := t.TempDir()
	distInfo := filepath.Join(dir, "PyYAML-6.0.1.dist-info")
	if err := os.MkdirAll(distInfo, 0o755); err != nil {
		t.Fatal(err)
	}

	f, err := os.Create(filepath.Join(distInfo, "RECORD"))
	if err != nil {
		t.Fatal(err)
	}
	w := csv.NewWriter(f)
	rows := [][]string{
		{"yaml/__init__.py", "sha256=abc", "1234"},
		{"yaml/scanner.py", "sha256=def", "5678"},
		{"PyYAML-6.0.1.dist-info/RECORD", "", ""},
	}
	for _, r := range rows {
		if err := w.Write(r); err != nil {
			t.Fatal(err)
		}
	}
	w.Flush()
	f.Close()

	got := pipPackageSize(dir, "PyYAML", "6.0.1")
	if want := int64(1234 + 5678); got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func TestPipPackageSizeNoMatch(t *testing.T) {
	dir := t.TempDir()
	if got := pipPackageSize(dir, "nonexistent", "1.0.0"); got != -1 {
		t.Errorf("got %d, want -1", got)
	}
}
