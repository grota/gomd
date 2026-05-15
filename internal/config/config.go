// Package config provides configuration management for gomd.
package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Config holds all application configuration.
type Config struct {
	UI       UIConfig       `toml:"ui"`
	Terminal TerminalConfig `toml:"terminal"`
	Images   ImageConfig    `toml:"images"`
	Content  ContentConfig  `toml:"content"`
	Keys     KeysConfig     `toml:"keys"`
}

// UIConfig holds UI-related settings.
type UIConfig struct {
	Theme                string            `toml:"theme"`
	CompactTree          bool              `toml:"compact_tree"`
	Opener               string            `toml:"opener"` // command to open URLs (default: xdg-open on Linux, open on macOS)
	GhosttyThemeDir      string            `toml:"ghostty_theme_directory"`
	SidebarHidden        bool              `toml:"sidebar_hidden"`
	ThemeOverride        ThemeOverrideConfig `toml:"theme_override"`
}

// ThemeOverrideConfig allows overriding individual colors of the active theme.
// Values should be hex color strings (e.g., "#ff0000").
type ThemeOverrideConfig struct {
	Border     string `toml:"border"`
	Selected   string `toml:"selected"`
	Heading1   string `toml:"heading1"`
	Heading2   string `toml:"heading2"`
	Heading3   string `toml:"heading3"`
	HeadingN   string `toml:"headingn"`
	Background string `toml:"background"`
	Foreground string `toml:"foreground"`
	StatusBar  string `toml:"statusbar"`
	Highlight  string `toml:"highlight"`
	Code       string `toml:"code"`
	Search     string `toml:"search"`
	NodeSel    string `toml:"nodesel"`
}

// TerminalConfig holds terminal-related settings.
type TerminalConfig struct {
	ColorMode string `toml:"color_mode"`
}

// ImageConfig holds image rendering settings.
type ImageConfig struct {
	Enabled bool `toml:"enabled"`
}

// ContentConfig holds content display settings.
type ContentConfig struct {
	SyntaxHighlighting bool `toml:"syntax_highlighting"`
}

// KeysConfig holds user-customizable key bindings.
// Each field maps an action name to one or more key strings (comma-separated).
// Key strings use the same format as bubbletea (e.g., "j", "down", "ctrl+d", "tab").
type KeysConfig struct {
	// Shared normal mode
	Quit          string `toml:"quit"`
	Help          string `toml:"help"`
	ThemePicker   string `toml:"theme_picker"`
	ToggleFocus   string `toml:"toggle_focus"`
	ToggleSidebar string `toml:"toggle_sidebar"`
	Reload        string `toml:"reload"`
	Edit          string `toml:"edit"`

	// Sidebar
	SidebarDown   string `toml:"sidebar_down"`
	SidebarUp     string `toml:"sidebar_up"`
	SidebarTop    string `toml:"sidebar_top"`
	SidebarBottom string `toml:"sidebar_bottom"`
	SidebarSearch string `toml:"sidebar_search"`
	NextMatch     string `toml:"next_match"`
	PrevMatch     string `toml:"prev_match"`

	// Content
	ScrollDown       string `toml:"scroll_down"`
	ScrollUp         string `toml:"scroll_up"`
	ScrollHalfDown   string `toml:"scroll_half_down"`
	ScrollHalfUp     string `toml:"scroll_half_up"`
	ContentTop       string `toml:"content_top"`
	ContentBottom    string `toml:"content_bottom"`
	NodeSelect       string `toml:"node_select"`
	ContentSearch    string `toml:"content_search"`
	ContentNextMatch string `toml:"content_next_match"`
	ContentPrevMatch string `toml:"content_prev_match"`
	Jump             string `toml:"jump"`

	// Viewport positioning
	ViewCenter string `toml:"view_center"` // zz: center selection in viewport
	ViewTop    string `toml:"view_top"`    // zt: scroll selection to top of viewport
	ViewBottom string `toml:"view_bottom"` // zb: scroll selection to bottom of viewport

	// Visible area jumps
	JumpHigh string `toml:"jump_high"` // H: top of visible area
	JumpMid  string `toml:"jump_mid"`  // M: middle of visible area
	JumpLow  string `toml:"jump_low"`  // L: bottom of visible area

	// Navigation history
	NavBack    string `toml:"nav_back"`
	NavForward string `toml:"nav_forward"`

	// Node select
	NodeNext string `toml:"node_next"`
	NodePrev string `toml:"node_prev"`
	NodeCopy string `toml:"node_copy"`
	NodeOpen string `toml:"node_open"`
	NodeExit string `toml:"node_exit"`
}

// DefaultKeys returns the default key bindings.
func DefaultKeys() KeysConfig {
	return KeysConfig{
		Quit:             "q,esc",
		Help:             "?",
		ThemePicker:      "T",
		ToggleFocus:      "tab",
		ToggleSidebar:    "w",
		Reload:           "r",
		Edit:             "e",
		SidebarDown:      "j,down",
		SidebarUp:        "k,up",
		SidebarTop:       "gg",
		SidebarBottom:    "G",
		SidebarSearch:    "/",
		NextMatch:        "n",
		PrevMatch:        "N",
		ScrollDown:       "j,down",
		ScrollUp:         "k,up",
		ScrollHalfDown:   "ctrl+d,ctrl+f,pgdown,space",
		ScrollHalfUp:     "ctrl+u,ctrl+b,pgup",
		ContentTop:       "gg",
		ContentBottom:    "G",
		NodeSelect:       "i",
		ContentSearch:    "/",
		ContentNextMatch: "n",
		ContentPrevMatch: "N",
		Jump:             "f",
		ViewCenter:       "zz",
		ViewTop:          "zt",
		ViewBottom:       "zb",
		JumpHigh:         "H",
		JumpMid:          "M",
		JumpLow:          "L",
		NavBack:          "ctrl+o",
		NavForward:       "ctrl+i",
		NodeNext:         "j,down,tab",
		NodePrev:         "k,up,shift+tab",
		NodeCopy:         "y",
		NodeOpen:         "enter",
		NodeExit:         "esc,q,i",
	}
}

// KeyMatches checks if a key string matches any of the configured keys for an action.
func KeyMatches(key string, binding string) bool {
	if binding == "" {
		return false
	}
	for _, k := range splitKeys(binding) {
		if k == key {
			return true
		}
	}
	return false
}

func splitKeys(s string) []string {
	var result []string
	for _, part := range strings.Split(s, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// Default returns a Config with default values.
func Default() Config {
	return Config{
		UI: UIConfig{
			Theme:         "OceanDark",
			CompactTree:   false,
			SidebarHidden: true,
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
		Keys: DefaultKeys(),
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
