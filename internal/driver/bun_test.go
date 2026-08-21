package driver

import "testing"

func TestParseBunPmList(t *testing.T) {
	// Real shape of `bun pm ls -g`'s output.
	out := []byte("C:\\Users\\me\\.bun\\install\\global node_modules (2)\n" +
		"\u251c\u2500\u2500 left-pad@1.3.0\n" +
		"\u2514\u2500\u2500 @types/node@20.11.5\n")

	root, specs, err := parseBunPmList(out)
	if err != nil {
		t.Fatalf("parseBunPmList: %v", err)
	}
	if root != `C:\Users\me\.bun\install\global` {
		t.Errorf("got root %q, want C:\\Users\\me\\.bun\\install\\global", root)
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

func TestParseBunPmListEmpty(t *testing.T) {
	out := []byte("C:\\Users\\me\\.bun\\install\\global node_modules\n")
	root, specs, err := parseBunPmList(out)
	if err != nil {
		t.Fatalf("parseBunPmList: %v", err)
	}
	if root != `C:\Users\me\.bun\install\global` {
		t.Errorf("got root %q", root)
	}
	if specs != nil {
		t.Errorf("got %v, want nil", specs)
	}
}

func TestParseBunPmListNoHeader(t *testing.T) {
	_, _, err := parseBunPmList([]byte("garbage\n"))
	if err == nil {
		t.Error("expected an error when the header line can't be found")
	}
}
