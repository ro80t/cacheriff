// Package config loads cacheriff's user configuration file, following
// the same convention as lazygit: a small YAML file under the user's
// config directory, entirely optional, with a "gui.theme" section for
// overriding colors.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"cacheriff/internal/theme"
)

// Config is the root of cacheriff's config file.
type Config struct {
	Gui GuiConfig `yaml:"gui"`
}

// GuiConfig holds display-related settings.
type GuiConfig struct {
	Theme theme.Override `yaml:"theme"`
}

// Path returns the location cacheriff reads its config file from:
// "<user config dir>/cacheriff/config.yml".
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(dir, "cacheriff", "config.yml"), nil
}

// Load reads the config file, if any. A missing file is not an error —
// it just means "use the defaults", matching lazygit's behavior.
func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}

// Theme resolves the effective color theme: cacheriff's built-in
// defaults with any user overrides from the config file applied on top.
func (c Config) Theme() theme.Theme {
	return c.Gui.Theme.Apply(theme.Default)
}
