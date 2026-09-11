package driver

import (
	"encoding/json"
	"testing"
)

func TestNixEnvJSONParsing(t *testing.T) {
	// Real shape captured from NixOS/nix#5877 (an object keyed by the
	// installed derivation name, values carrying separately-split
	// pname/version since NixOS/nix#4463).
	out := []byte(`{
		"hello-2.12.1": {
			"pname": "hello",
			"version": "2.12.1",
			"system": "x86_64-linux",
			"meta": {"available": true, "broken": false}
		},
		"ripgrep-14.1.0": {
			"pname": "ripgrep",
			"version": "14.1.0",
			"system": "x86_64-linux",
			"meta": {"available": true, "broken": false}
		}
	}`)

	var parsed map[string]nixEnvPackage
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(parsed) != 2 {
		t.Fatalf("got %d packages, want 2: %+v", len(parsed), parsed)
	}
	if got := parsed["hello-2.12.1"]; got.Pname != "hello" || got.Version != "2.12.1" {
		t.Errorf("got %+v, want {hello 2.12.1}", got)
	}
	if got := parsed["ripgrep-14.1.0"]; got.Pname != "ripgrep" || got.Version != "14.1.0" {
		t.Errorf("got %+v, want {ripgrep 14.1.0}", got)
	}
}

func TestNixStoreDirDefault(t *testing.T) {
	t.Setenv("NIX_STORE_DIR", "")
	if got := nixStoreDir(); got != "/nix/store" {
		t.Errorf("got %q, want /nix/store", got)
	}
}

func TestNixStoreDirOverride(t *testing.T) {
	t.Setenv("NIX_STORE_DIR", "/custom/store")
	if got := nixStoreDir(); got != "/custom/store" {
		t.Errorf("got %q, want /custom/store", got)
	}
}
