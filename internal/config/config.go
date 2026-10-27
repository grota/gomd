// Package config provides configuration management for gomd.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Config holds all application configuration.
type Config struct {
	UI     UIConfig    `toml:"ui"`
	Images ImageConfig `toml:"images"`
	Keys   KeysConfig  `toml:"keys"`
}

// UIConfig holds UI-related settings.
type UIConfig struct {
	Theme         string              `toml:"theme"`
	CompactTree   bool                `toml:"compact_tree"`
	Opener        string              `toml:"opener"` // command to open URLs, may include arguments (default: xdg-open on Linux, open on macOS)
	SidebarHidden bool                `toml:"sidebar_hidden"`
	ContentBorder string              `toml:"content_border"` // ContentBorderAlways or ContentBorderWithSidebar
	ThemeOverride ThemeOverrideConfig `toml:"theme_override"`
}

// Values of ui.content_border.
const (
	ContentBorderAlways      = "always"       // border the content pane in every layout
	ContentBorderWithSidebar = "with_sidebar" // border it only while the sidebar is shown
)

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

// ImageConfig holds image rendering settings.
type ImageConfig struct {
	Enabled bool `toml:"enabled"`
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
	NodeMode string `toml:"node_mode"` // cycle sub-mode (all/code/inline/links)
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
		NodeMode:         "m",
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

// IsSequencePrefix reports whether key starts a two-key sequence such as "gg"
// in any of the bindings, so the caller should wait for the next key.
// shortcut: a sequence is any 2-character entry except f1-f9; extend this if
// longer sequences are ever needed.
func IsSequencePrefix(key string, bindings ...string) bool {
	for _, b := range bindings {
		for _, e := range splitKeys(b) {
			isFn := e[0] == 'f' && len(e) == 2 && e[1] >= '0' && e[1] <= '9'
			if len(e) == 2 && !isFn && e[:1] == key {
				return true
			}
		}
	}
	return false
}

// FirstKey returns the first key of a binding, for short on-screen hints.
func FirstKey(binding string) string {
	if keys := splitKeys(binding); len(keys) > 0 {
		return keys[0]
	}
	return ""
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
			Theme:         "light:OceanLight,dark:OceanDark",
			CompactTree:   false,
			SidebarHidden: true,
			ContentBorder: ContentBorderAlways,
		},
		Images: ImageConfig{
			Enabled: true,
		},
		Keys: DefaultKeys(),
	}
}

// Load reads the configuration file, returning defaults if it is missing.
// A malformed file yields defaults plus an error describing the problem.
func Load() (Config, error) {
	cfg := Default()
	path := configPath()
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Default(), fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// SaveTheme persists ui.theme into the config file, editing only that line so
// the user's comments, ordering and other keys survive.
func SaveTheme(name string) error {
	path := configPath()
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	var probe map[string]any
	if err := toml.Unmarshal(data, &probe); err != nil {
		return fmt.Errorf("config %s is invalid, not saving: %w", path, err)
	}
	out := setUITheme(string(data), name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(out), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

var tomlTableRe = regexp.MustCompile(`^\s*\[`)
var uiThemeRe = regexp.MustCompile(`^(\s*theme\s*=\s*).*$`)

// setUITheme sets `theme` inside the [ui] table of TOML text.
// shortcut: line-based, ignores dotted keys like `ui.theme = ...`; switch to a
// comment-preserving TOML editor if the config grows more complex.
func setUITheme(content, name string) string {
	line := fmt.Sprintf("theme = %q", name)
	lines := strings.Split(content, "\n")
	uiAt := -1
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if tomlTableRe.MatchString(l) {
			if uiAt >= 0 {
				break
			}
			if t == "[ui]" {
				uiAt = i
			}
			continue
		}
		if uiAt >= 0 && uiThemeRe.MatchString(l) {
			lines[i] = uiThemeRe.ReplaceAllString(l, "${1}") + fmt.Sprintf("%q", name)
			return strings.Join(lines, "\n")
		}
	}
	if uiAt >= 0 {
		lines = append(lines[:uiAt+1], append([]string{line}, lines[uiAt+1:]...)...)
		return strings.Join(lines, "\n")
	}
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if content != "" {
		content += "\n"
	}
	return content + "[ui]\n" + line + "\n"
}

func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "gomd", "config.toml")
}
