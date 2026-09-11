package driver

import "testing"

func TestParsePyLauncherOutput(t *testing.T) {
	// Real `py -0p` output, captured on a machine with Python 3.13 on
	// PATH and Python 2.7 installed but not on PATH.
	out := " -V:3.13 *        C:\\Users\\tkata\\AppData\\Local\\Programs\\Python\\Python313\\python.exe\r\n" +
		" -V:2.7           C:\\Python27\\python.exe\r\n"

	got := parsePyLauncherOutput(out)
	want := []string{
		`C:\Users\tkata\AppData\Local\Programs\Python\Python313\Scripts`,
		`C:\Python27\Scripts`,
	}
	if len(got) != len(want) {
		t.Fatalf("got %d dirs, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("dir %d: got %q, want %q", i, got[i], w)
		}
	}
}

func TestParsePyLauncherOutputEmpty(t *testing.T) {
	if got := parsePyLauncherOutput(""); got != nil {
		t.Errorf("got %+v, want nil", got)
	}
}
