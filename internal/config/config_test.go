package config

import (
	"os"
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
