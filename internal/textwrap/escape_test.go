package textwrap

import "testing"

func TestEscapeArgAllowsOrdinaryPackageNames(t *testing.T) {
	for _, name := range []string{"left-pad", "@scope/name", "serde", "github.com/foo/bar", "left_pad2"} {
		got, err := EscapeArg(name)
		if err != nil {
			t.Errorf("EscapeArg(%q) returned error: %v", name, err)
		}
		if got != name {
			t.Errorf("EscapeArg(%q) = %q, want unchanged", name, got)
		}
	}
}

func TestEscapeArgRejectsEmpty(t *testing.T) {
	if _, err := EscapeArg(""); err == nil {
		t.Error("EscapeArg(\"\") returned nil error, want an error")
	}
}

func TestEscapeArgRejectsFlagLikeNames(t *testing.T) {
	for _, name := range []string{"-g", "--force", "-", "--uninstall"} {
		if _, err := EscapeArg(name); err == nil {
			t.Errorf("EscapeArg(%q) returned nil error, want an error (looks like a flag)", name)
		}
	}
}

func TestEscapeArgRejectsControlCharacters(t *testing.T) {
	for _, name := range []string{"left\x00pad", "left\npad", "left\rpad", "left\x7fpad"} {
		if _, err := EscapeArg(name); err == nil {
			t.Errorf("EscapeArg(%q) returned nil error, want an error (control character)", name)
		}
	}
}
