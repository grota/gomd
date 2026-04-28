// Package config provides configuration management for gomd.
package config

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// Config holds all application configuration.
type Config struct {
	UI       UIConfig       `toml:"ui"`
	Terminal TerminalConfig `toml:"terminal"`
	Images   ImageConfig    `toml:"images"`
	Content  ContentConfig  `toml:"content"`
}

// UIConfig holds UI-related settings.
type UIConfig struct {
	Theme       string `toml:"theme"`
	CompactTree bool   `toml:"compact_tree"`
}

// TerminalConfig holds terminal-related settings.
type TerminalConfig struct {
	ColorMode           string `toml:"color_mode"`
	WarnedTerminalApp   bool   `toml:"warned_terminal_app"`
}

// ImageConfig holds image rendering settings.
type ImageConfig struct {
	Enabled bool `toml:"enabled"`
}

// ContentConfig holds content display settings.
type ContentConfig struct {
	SyntaxHighlighting bool `toml:"syntax_highlighting"`
}

// Default returns a Config with default values.
func Default() Config {
	return Config{
		UI: UIConfig{
			Theme:       "OceanDark",
			CompactTree: false,
		},
		Terminal: TerminalConfig{
			ColorMode: "auto",
		},
		Images: ImageConfig{
			Enabled: true,
		},
		Content: ContentConfig{
			SyntaxHighlighting: true,
		},
	}
}

// Load reads the configuration file, returning defaults if not found.
func Load() Config {
	cfg := Default()

	path := configPath()
	if path == "" {
		return cfg
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg
	}

	return cfg
}

// Save writes the configuration to disk.
func (c *Config) Save() error {
	path := configPath()
	if path == "" {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := toml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// IsCompactTree returns whether the compact tree style is enabled.
func (c *Config) IsCompactTree() bool {
	return c.UI.CompactTree
}

func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "gomd", "config.toml")
}
