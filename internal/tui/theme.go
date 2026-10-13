// Package tui provides the interactive terminal user interface for gomd.
package tui

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	gansi "charm.land/glamour/v2/ansi"
	"charm.land/lipgloss/v2"

	"github.com/grota/gomd/internal/config"
)

// ─────────────────────────────────────────────
// Themes
// ─────────────────────────────────────────────

// Color is a "#rrggbb" theme color. It keeps the hex string (glamour styles
// need it) and satisfies color.Color, so it can be passed to lipgloss directly.
type Color string

// RGBA implements color.Color.
func (c Color) RGBA() (r, g, b, a uint32) { return lipgloss.Color(string(c)).RGBA() }

// Theme holds color settings for the TUI.
type Theme struct {
	Name       string
	Border     Color
	Selected   Color
	Heading1   Color
	Heading2   Color
	Heading3   Color
	HeadingN   Color
	Background Color
	Foreground Color
	StatusBar  Color
	Highlight  Color
	Code       Color
	Search     Color
	NodeSel    Color // selected node in interactive mode
}

var themes = map[string]Theme{
	"OceanDark": {
		Name: "OceanDark", Border: "#4a6fa5", Selected: "#2d5986",
		Heading1: "#6fb3d2", Heading2: "#59c2a5", Heading3: "#82aaff", HeadingN: "#7f9fbf",
		Background: "#1a2332", Foreground: "#c5d4e8", StatusBar: "#253545",
		Highlight: "#ffd700", Code: "#1e2a3a", Search: "#ff6b6b", NodeSel: "#ff9f43",
	},
	// OceanLight is OceanDark's light companion; it exists in no remote set.
	"OceanLight": {
		Name: "OceanLight", Border: "#7593bb", Selected: "#cddcef",
		Heading1: "#1f5f99", Heading2: "#17735f", Heading3: "#5146b8", HeadingN: "#536781",
		Background: "#f3f6fa", Foreground: "#1e2b3a", StatusBar: "#dde6f0",
		Highlight: "#2f6fb0", Code: "#e6edf5", Search: "#b42318", NodeSel: "#1f5f99",
	},
}

// builtinNames lists the built-in themes, dark first.
var builtinNames = []string{"OceanDark", "OceanLight"}

// ThemeSetting is a parsed ui.theme value: Ghostty's "light:A,dark:B" form,
// or a single name used in both modes.
type ThemeSetting struct{ Light, Dark string }

// ParseThemeSetting parses "light:A,dark:B" (either order) or a single name.
func ParseThemeSetting(s string) (ThemeSetting, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		s = config.Default().UI.Theme
	}
	if !strings.Contains(s, "light:") && !strings.Contains(s, "dark:") {
		return ThemeSetting{Light: s, Dark: s}, nil
	}
	var t ThemeSetting
	for _, part := range strings.Split(s, ",") {
		mode, name, ok := strings.Cut(strings.TrimSpace(part), ":")
		name = strings.TrimSpace(name)
		switch {
		case !ok || name == "":
			return ThemeSetting{}, fmt.Errorf("invalid theme %q: want \"light:A,dark:B\" or a single name", s)
		case mode == "light":
			t.Light = name
		case mode == "dark":
			t.Dark = name
		default:
			return ThemeSetting{}, fmt.Errorf("invalid theme %q: unknown mode %q", s, mode)
		}
	}
	if t.Light == "" || t.Dark == "" {
		return ThemeSetting{}, fmt.Errorf("invalid theme %q: both light: and dark: are required", s)
	}
	return t, nil
}

// String formats the setting the way ParseThemeSetting reads it.
func (t ThemeSetting) String() string {
	if t.Light == t.Dark {
		return t.Light
	}
	return "light:" + t.Light + ",dark:" + t.Dark
}

// For returns the theme name for the given mode.
func (t ThemeSetting) For(dark bool) string {
	if dark {
		return t.Dark
	}
	return t.Light
}

// LoadTheme resolves a theme name, built-in or from the downloaded set, and
// applies the config's colour overrides.
func LoadTheme(name string, ov config.ThemeOverrideConfig) (Theme, error) {
	theme, ok := themes[name]
	if !ok {
		var err error
		if theme, err = loadRemoteTheme(name); err != nil {
			return Theme{}, err
		}
	}
	return applyOverrides(theme, ov), nil
}

// ThemeFor returns the configured theme for the mode. When it cannot be
// loaded (e.g. the Ghostty themes are not downloaded) it returns the Ocean
// built-in for the mode together with the error.
func ThemeFor(cfg config.Config, dark bool) (Theme, error) {
	setting, err := ParseThemeSetting(cfg.UI.Theme)
	if err == nil {
		var t Theme
		if t, err = LoadTheme(setting.For(dark), cfg.UI.ThemeOverride); err == nil {
			return t, nil
		}
	}
	fallback := "OceanLight"
	if dark {
		fallback = "OceanDark"
	}
	return applyOverrides(themes[fallback], cfg.UI.ThemeOverride), err
}

// applyOverrides replaces the theme colours set in [ui.theme_override].
func applyOverrides(theme Theme, ov config.ThemeOverrideConfig) Theme {
	applyOverride := func(dst *Color, src string) {
		if src != "" {
			*dst = Color(src)
		}
	}
	applyOverride(&theme.Border, ov.Border)
	applyOverride(&theme.Selected, ov.Selected)
	applyOverride(&theme.Heading1, ov.Heading1)
	applyOverride(&theme.Heading2, ov.Heading2)
	applyOverride(&theme.Heading3, ov.Heading3)
	applyOverride(&theme.HeadingN, ov.HeadingN)
	applyOverride(&theme.Background, ov.Background)
	applyOverride(&theme.Foreground, ov.Foreground)
	applyOverride(&theme.StatusBar, ov.StatusBar)
	applyOverride(&theme.Highlight, ov.Highlight)
	applyOverride(&theme.Code, ov.Code)
	applyOverride(&theme.Search, ov.Search)
	applyOverride(&theme.NodeSel, ov.NodeSel)
	return theme
}

// luminance is the relative luminance of a "#rrggbb" colour, as Ghostty uses
// to sort themes into dark (< 0.5) and light.
func luminance(hex Color) float64 {
	r, g, b, ok := parseHex(string(hex))
	if !ok {
		return 0
	}
	return (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) / 255
}

// contrast is the WCAG contrast ratio of two "#rrggbb" colours, 1 to 21.
func contrast(a, b Color) float64 {
	rel := func(c Color) float64 {
		r, g, b, ok := parseHex(string(c))
		if !ok {
			return 0
		}
		lin := func(v uint64) float64 {
			s := float64(v) / 255
			if s <= 0.04045 {
				return s / 12.92
			}
			return math.Pow((s+0.055)/1.055, 2.4)
		}
		return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
	}
	la, lb := rel(a), rel(b)
	return (max(la, lb) + 0.05) / (min(la, lb) + 0.05)
}

// textOn returns a readable text colour for a fill painted in bg: the theme
// foreground or background when either reaches WCAG AA (4.5:1), else black or
// white. Ghostty themes such as Primary set palette 8 equal to the foreground.
func (t Theme) textOn(bg Color) Color {
	for _, c := range []Color{t.Foreground, t.Background} {
		if contrast(c, bg) >= 4.5 {
			return c
		}
	}
	if contrast("#000000", bg) > contrast("#ffffff", bg) {
		return "#000000"
	}
	return "#ffffff"
}

// mix blends a towards b by k (0..1). It returns a when either is not "#rrggbb".
func mix(a, b Color, k float64) Color {
	ar, ag, ab, ok1 := parseHex(string(a))
	br, bg, bb, ok2 := parseHex(string(b))
	if !ok1 || !ok2 {
		return a
	}
	m := func(x, y uint64) uint64 { return uint64(math.Round(float64(x) + (float64(y)-float64(x))*k)) }
	return Color(fmt.Sprintf("#%02x%02x%02x", m(ar, br), m(ag, bg), m(ab, bb)))
}

// ensureContrast moves c towards black or white, whichever is further from
// bg, until it reaches ratio against bg. Black or white reach 4.58:1 on any
// background, so every ratio up to that is met.
func ensureContrast(c, bg Color, ratio float64) Color {
	to := Color("#ffffff")
	if contrast("#000000", bg) > contrast(to, bg) {
		to = "#000000"
	}
	for k := 0.0; k < 1; k += 0.02 {
		if m := mix(c, to, k); contrast(m, bg) >= ratio {
			return m
		}
	}
	return to
}

// ─────────────────────────────────────────────
// Glamour style
// ─────────────────────────────────────────────

func boolPtr(b bool) *bool       { return &b }
func stringPtr(s string) *string { return &s }
func uintPtr(u uint) *uint       { return &u }

// StyleConfigFromTheme builds a glamour StyleConfig from a gomd Theme,
// mapping the theme's colors directly into the glamour rendering style.
func StyleConfigFromTheme(theme Theme) gansi.StyleConfig {
	fg := string(theme.Foreground)
	h1 := string(theme.Heading1)
	h2 := string(theme.Heading2)
	h3 := string(theme.Heading3)
	hN := string(theme.HeadingN)
	link := string(theme.Search) // reuse Search color for links
	border := string(theme.Border)

	// Code background: the page background moved towards white (dark themes)
	// or black (light themes) to 1.25:1, the same visible step on every theme.
	// theme.Code is the interactive-mode tint, not a text colour.
	codeBg := string(ensureContrast(theme.Background, theme.Background, 1.25))
	codeFg := string(ensureContrast(theme.Foreground, Color(codeBg), 4.5))

	return gansi.StyleConfig{
		Document: gansi.StyleBlock{
			StylePrimitive: gansi.StylePrimitive{
				Color:       &fg,
				BlockPrefix: "\n",
				BlockSuffix: "\n",
			},
			Margin: uintPtr(2),
		},
		BlockQuote: gansi.StyleBlock{
			Indent:      uintPtr(1),
			IndentToken: stringPtr("│ "),
		},
		List: gansi.StyleList{
			LevelIndent: 2,
		},
		Heading: gansi.StyleBlock{
			StylePrimitive: gansi.StylePrimitive{
				Color:       &hN,
				Bold:        boolPtr(true),
				BlockSuffix: "\n",
			},
		},
		H1: gansi.StyleBlock{
			StylePrimitive: gansi.StylePrimitive{
				Prefix: " ",
				Suffix: " ",
				Color:  &h1,
				Bold:   boolPtr(true),
			},
		},
		H2: gansi.StyleBlock{
			StylePrimitive: gansi.StylePrimitive{
				Prefix: "## ",
				Color:  &h2,
			},
		},
		H3: gansi.StyleBlock{
			StylePrimitive: gansi.StylePrimitive{
				Prefix: "### ",
				Color:  &h3,
			},
		},
		H4: gansi.StyleBlock{
			StylePrimitive: gansi.StylePrimitive{
				Prefix: "#### ",
				Color:  &hN,
			},
		},
		H5: gansi.StyleBlock{
			StylePrimitive: gansi.StylePrimitive{
				Prefix: "##### ",
				Color:  &hN,
			},
		},
		H6: gansi.StyleBlock{
			StylePrimitive: gansi.StylePrimitive{
				Prefix: "###### ",
				Color:  &hN,
				Bold:   boolPtr(false),
			},
		},
		Strikethrough: gansi.StylePrimitive{
			CrossedOut: boolPtr(true),
		},
		Emph: gansi.StylePrimitive{
			Italic: boolPtr(true),
		},
		Strong: gansi.StylePrimitive{
			Bold: boolPtr(true),
		},
		HorizontalRule: gansi.StylePrimitive{
			Color:  &border,
			Format: "\n--------\n",
		},
		Item: gansi.StylePrimitive{
			BlockPrefix: "• ",
		},
		Enumeration: gansi.StylePrimitive{
			BlockPrefix: ". ",
		},
		Task: gansi.StyleTask{
			Ticked:   "[✓] ",
			Unticked: "[ ] ",
		},
		Link: gansi.StylePrimitive{
			Color:     &link,
			Underline: boolPtr(true),
		},
		LinkText: gansi.StylePrimitive{
			Color: &h3,
			Bold:  boolPtr(true),
		},
		Image: gansi.StylePrimitive{
			Color:     &link,
			Underline: boolPtr(true),
		},
		ImageText: gansi.StylePrimitive{
			Color:  &border,
			Format: "{{.text}} 📷",
		},
		Code: gansi.StyleBlock{
			StylePrimitive: gansi.StylePrimitive{
				Prefix:          " ",
				Suffix:          " ",
				Color:           &codeFg,
				BackgroundColor: &codeBg,
			},
		},
		CodeBlock: gansi.StyleCodeBlock{
			StyleBlock: gansi.StyleBlock{
				StylePrimitive: gansi.StylePrimitive{
					Color: &codeFg,
				},
				Margin: uintPtr(2),
			},
			Chroma: chromaForTheme(codeFg, codeBg, isLightColor(codeBg)),
		},
		Table: gansi.StyleTable{},
		DefinitionDescription: gansi.StylePrimitive{
			BlockPrefix: "\n🠶 ",
		},
	}
}

// parseHex parses "#rrggbb" into its components.
func parseHex(hex string) (r, g, b uint64, ok bool) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0, false
	}
	r, err1 := strconv.ParseUint(hex[0:2], 16, 8)
	g, err2 := strconv.ParseUint(hex[2:4], 16, 8)
	b, err3 := strconv.ParseUint(hex[4:6], 16, 8)
	return r, g, b, err1 == nil && err2 == nil && err3 == nil
}

// BGOpen returns the truecolor background escape for a "#rrggbb" color, or "".
func BGOpen(hex string) string {
	r, g, b, ok := parseHex(hex)
	if !ok {
		return ""
	}
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
}

// isLightColor reports whether a "#rrggbb" color is light (simplified sRGB luminance).
func isLightColor(hex string) bool {
	r, g, b, ok := parseHex(hex)
	return ok && 0.299*float64(r)+0.587*float64(g)+0.114*float64(b) > 128
}

// chromaForTheme returns a full Chroma token color config appropriate for light
// or dark backgrounds.  Colors are taken from glamour's built-in styles and
// pushed towards black or white until readable on codeBg: 4.5:1 (WCAG AA),
// 3:1 for comments so they stay dimmer than code.
func chromaForTheme(fg, codeBg string, light bool) *gansi.Chroma {
	readable := func(color string, ratio float64) string {
		if color == "" {
			return ""
		}
		return string(ensureContrast(Color(color), Color(codeBg), ratio))
	}
	dim := func(color string) gansi.StylePrimitive {
		color = readable(color, 3)
		return gansi.StylePrimitive{Color: &color}
	}
	sp := func(color string) gansi.StylePrimitive {
		if color == "" {
			return gansi.StylePrimitive{}
		}
		color = readable(color, 4.5)
		return gansi.StylePrimitive{Color: &color}
	}
	spBg := func(fg, bg string) gansi.StylePrimitive {
		p := gansi.StylePrimitive{}
		if fg != "" {
			p.Color = &fg
		}
		if bg != "" {
			p.BackgroundColor = &bg
		}
		return p
	}
	spStyle := func(color string, bold, italic, underline bool) gansi.StylePrimitive {
		p := gansi.StylePrimitive{}
		if color != "" {
			color = readable(color, 4.5)
			p.Color = &color
		}
		if bold {
			p.Bold = boolPtr(true)
		}
		if italic {
			p.Italic = boolPtr(true)
		}
		if underline {
			p.Underline = boolPtr(true)
		}
		return p
	}

	if light {
		return &gansi.Chroma{
			Text:                sp("#2A2A2A"),
			Error:               spBg("#F1F1F1", "#FF5555"),
			Comment:             dim("#8D8D8D"),
			CommentPreproc:      sp("#FF875F"),
			Keyword:             sp("#279EFC"),
			KeywordReserved:     sp("#FF5FD2"),
			KeywordNamespace:    sp("#FB406F"),
			KeywordType:         sp("#7049C2"),
			Operator:            sp("#FF2626"),
			Punctuation:         sp("#FA7878"),
			NameBuiltin:         sp("#0A1BB1"),
			NameTag:             sp("#581290"),
			NameAttribute:       sp("#8362CB"),
			NameClass:           spStyle("#212121", true, false, true),
			NameConstant:        sp("#581290"),
			NameDecorator:       sp("#A3A322"),
			NameFunction:        sp("#019F57"),
			LiteralNumber:       sp("#22CCAE"),
			LiteralString:       sp("#7E5B38"),
			LiteralStringEscape: sp("#00AEAE"),
			GenericDeleted:      sp("#FD5B5B"),
			GenericEmph:         spStyle("", false, true, false),
			GenericInserted:     sp("#00D787"),
			GenericStrong:       spStyle("", true, false, false),
			GenericSubheading:   dim("#777777"),
			Background:          spBg("", codeBg),
		}
	}

	return &gansi.Chroma{
		Text:                sp(fg),
		Error:               spBg("#F1F1F1", "#F05B5B"),
		Comment:             dim("#676767"),
		CommentPreproc:      sp("#FF875F"),
		Keyword:             sp("#00AAFF"),
		KeywordReserved:     sp("#FF5FD2"),
		KeywordNamespace:    sp("#FF5F87"),
		KeywordType:         sp("#6E6ED8"),
		Operator:            sp("#EF8080"),
		Punctuation:         sp("#E8E8A8"),
		Name:                sp(fg),
		NameBuiltin:         sp("#FF8EC7"),
		NameTag:             sp("#B083EA"),
		NameAttribute:       sp("#7A7AE6"),
		NameClass:           spStyle("#F1F1F1", true, false, true),
		NameDecorator:       sp("#FFFF87"),
		NameFunction:        sp("#00D787"),
		LiteralNumber:       sp("#6EEFC0"),
		LiteralString:       sp("#C69669"),
		LiteralStringEscape: sp("#AFFFD7"),
		GenericDeleted:      sp("#FD5B5B"),
		GenericEmph:         spStyle("", false, true, false),
		GenericInserted:     sp("#00D787"),
		GenericStrong:       spStyle("", true, false, false),
		GenericSubheading:   dim("#777777"),
		Background:          spBg("", codeBg),
	}
}
