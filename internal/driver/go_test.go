package driver

import "testing"

func TestEscapeModulePath(t *testing.T) {
	tests := []struct{ in, want string }{
		{"github.com/BurntSushi/toml", "github.com/!burnt!sushi/toml"},
		{"github.com/go-delve/delve", "github.com/go-delve/delve"},
		{"v1.27.1", "v1.27.1"},
	}
	for _, tt := range tests {
		if got := escapeModulePath(tt.in); got != tt.want {
			t.Errorf("escapeModulePath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseGoVersionM(t *testing.T) {
	// Real shape of `go version -m` across multiple binaries.
	out := []byte("C:/Users/me/go/bin/dlv.exe: go1.26.5\n" +
		"\tpath\tgithub.com/go-delve/delve/cmd/dlv\n" +
		"\tmod\tgithub.com/go-delve/delve\tv1.27.1\th1:abc=\n" +
		"\tdep\tgithub.com/cilium/ebpf\tv0.11.0\th1:def=\n" +
		"\tbuild\t-buildmode=exe\n" +
		"C:/Users/me/go/bin/go1.24.0.exe: go1.21.2\n" +
		"\tpath\tgolang.org/dl/go1.24.0\n" +
		"\tmod\tgolang.org/dl\tv0.0.0-20250806180942-db69247c9fc7\th1:xyz=\n" +
		"\tbuild\t-buildmode=exe\n")

	entries := parseGoVersionM(out)
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2: %+v", len(entries), entries)
	}

	if entries[0].Name != "dlv" || entries[0].Version != "v1.27.1" {
		t.Errorf("got %+v, want name=dlv version=v1.27.1", entries[0])
	}
	if entries[0].Kind != KindGlobalPackage {
		t.Errorf("got Kind=%v, want KindGlobalPackage", entries[0].Kind)
	}

	if entries[1].Name != "go1.24.0" || entries[1].Version != "v0.0.0-20250806180942-db69247c9fc7" {
		t.Errorf("got %+v, want name=go1.24.0 version=v0.0.0-20250806180942-db69247c9fc7", entries[1])
	}
}

func TestParseGoVersionMSkipsUnreadableBinaries(t *testing.T) {
	// `go version -m` prints its per-file error to stderr, not
	// stdout, but a binary it couldn't read build info from still
	// gets a header line with nothing useful following it.
	out := []byte("C:/Users/me/go/bin/dlv.exe: go1.26.5\n" +
		"\tpath\tgithub.com/go-delve/delve/cmd/dlv\n" +
		"\tmod\tgithub.com/go-delve/delve\tv1.27.1\th1:abc=\n" +
		"C:/Users/me/go/bin/junk.exe: go1.26.5\n")

	entries := parseGoVersionM(out)
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1 (junk.exe has no mod line to report): %+v", len(entries), entries)
	}
	if entries[0].Name != "dlv" {
		t.Errorf("got %+v, want name=dlv", entries[0])
	}
}

func TestParseGoVersionMEmpty(t *testing.T) {
	if entries := parseGoVersionM(nil); entries != nil {
		t.Errorf("got %+v, want nil", entries)
	}
}
