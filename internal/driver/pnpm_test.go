package driver

import "testing"

func TestParsePnpmList(t *testing.T) {
	// Real shape of `pnpm list -g --depth=0 --json`'s output.
	data := []byte(`[
		{
			"path": "C:\\Users\\me\\AppData\\Local\\pnpm\\global\\5",
			"private": false,
			"dependencies": {
				"is-odd": {
					"from": "is-odd",
					"version": "3.0.1",
					"resolved": "https://registry.npmjs.org/is-odd/-/is-odd-3.0.1.tgz",
					"path": "C:\\Users\\me\\AppData\\Local\\pnpm\\global\\5\\.pnpm\\is-odd@3.0.1\\node_modules\\is-odd"
				}
			}
		}
	]`)

	entries, err := parsePnpmList(data, KindGlobalPackage)
	if err != nil {
		t.Fatalf("parsePnpmList: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1: %+v", len(entries), entries)
	}
	e := entries[0]
	if e.Name != "is-odd" || e.Version != "3.0.1" {
		t.Errorf("got name=%q version=%q, want is-odd/3.0.1", e.Name, e.Version)
	}
	if e.Kind != KindGlobalPackage {
		t.Errorf("got Kind=%v, want KindGlobalPackage", e.Kind)
	}
	if e.Path == "" {
		t.Error("got empty Path")
	}
}

func TestParsePnpmListNoDependencies(t *testing.T) {
	// pnpm omits the "dependencies" key entirely when nothing is
	// installed, rather than emitting an empty object.
	data := []byte(`[{"path": "C:\\Users\\me\\AppData\\Local\\pnpm\\global\\5", "private": false}]`)

	entries, err := parsePnpmList(data, KindGlobalPackage)
	if err != nil {
		t.Fatalf("parsePnpmList: %v", err)
	}
	if entries != nil {
		t.Errorf("got %+v, want nil", entries)
	}
}
