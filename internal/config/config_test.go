package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"cacheriff/internal/theme"
)

func TestExampleConfigMatchesDefaultTheme(t *testing.T) {
	data, err := os.ReadFile("../../config.example.yml")
	if err != nil {
		t.Fatalf("read example config: %v", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	got := cfg.Theme()
	want := theme.Theme{
		Name:           "cacheriff-dark", // Name isn't overridable via config
		Primary:        "#FF5FAF",
		ActiveBorder:   "#FF87D7",
		InactiveBorder: "#585858",
		Muted:          "#585858",
		Faint:          "#808080",
		Error:          "#FF5F5F",
		Success:        "#87D787",
	}
	if got != want {
		t.Fatalf("theme mismatch\n got: %+v\nwant: %+v", got, want)
	}

	// A partial override should only touch the fields it sets.
	partial := Config{Gui: GuiConfig{Theme: theme.Override{Primary: "#00FF00"}}}
	merged := partial.Theme()
	if merged.Primary != "#00FF00" {
		t.Fatalf("expected overridden Primary, got %q", merged.Primary)
	}
	if merged.Error != theme.Default.Error {
		t.Fatalf("expected untouched Error to stay default, got %q", merged.Error)
	}
}

func TestPathHonorsConfigFileEnvVar(t *testing.T) {
	want := filepath.Join(t.TempDir(), "custom.yml")
	t.Setenv(FileEnvVar, want)

	got, err := Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if got != want {
		t.Fatalf("Path() = %q, want %q", got, want)
	}
}

func TestLoadReadsFileFromEnvVar(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom.yml")
	if err := os.WriteFile(path, []byte("gui:\n  theme:\n    primaryColor: \"#123456\"\n"), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	t.Setenv(FileEnvVar, path)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.Theme().Primary; got != "#123456" {
		t.Fatalf("Primary = %q, want #123456", got)
	}
}
