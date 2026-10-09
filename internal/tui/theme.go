// Package tui provides the interactive terminal user interface for gomd.
package tui

import (
	"fmt"
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
	"Nord": {
		Name: "Nord", Border: "#4c566a", Selected: "#3b4252",
		Heading1: "#88c0d0", Heading2: "#81a1c1", Heading3: "#5e81ac", HeadingN: "#616e88",
		Background: "#2e3440", Foreground: "#d8dee9", StatusBar: "#3b4252",
		Highlight: "#ebcb8b", Code: "#272c36", Search: "#bf616a", NodeSel: "#a3be8c",
	},
	"Dracula": {
		Name: "Dracula", Border: "#6272a4", Selected: "#44475a",
		Heading1: "#bd93f9", Heading2: "#ff79c6", Heading3: "#8be9fd", HeadingN: "#6272a4",
		Background: "#282a36", Foreground: "#f8f8f2", StatusBar: "#44475a",
		Highlight: "#f1fa8c", Code: "#21222c", Search: "#ff5555", NodeSel: "#50fa7b",
	},
	"Gruvbox": {
		Name: "Gruvbox", Border: "#504945", Selected: "#3c3836",
		Heading1: "#fabd2f", Heading2: "#b8bb26", Heading3: "#83a598", HeadingN: "#928374",
		Background: "#282828", Foreground: "#ebdbb2", StatusBar: "#3c3836",
		Highlight: "#fe8019", Code: "#1d2021", Search: "#fb4934", NodeSel: "#8ec07c",
	},
	"TokyoNight": {
		Name: "TokyoNight", Border: "#3b4261", Selected: "#283457",
		Heading1: "#7aa2f7", Heading2: "#7dcfff", Heading3: "#bb9af7", HeadingN: "#565f89",
		Background: "#1a1b26", Foreground: "#c0caf5", StatusBar: "#1f2335",
		Highlight: "#e0af68", Code: "#16161e", Search: "#f7768e", NodeSel: "#9ece6a",
	},
}

// themeNames lists the built-in themes in theme-picker order.
var themeNames = []string{"OceanDark", "Nord", "Dracula", "Gruvbox", "TokyoNight"}

// ResolveTheme returns the active theme, applying ghostty loading and config overrides.
// It first checks built-in themes, then tries loading from the ghostty theme directory,
// and finally applies any color overrides from the config.
func ResolveTheme(cfg config.Config) Theme {
	theme, ok := themes[cfg.UI.Theme]
	if !ok {
		theme = themes["OceanDark"]
		if cfg.UI.GhosttyThemeDir != "" {
			if t, err := LoadGhosttyTheme(cfg.UI.GhosttyThemeDir, cfg.UI.Theme); err == nil {
				theme = t
			}
		}
	}

	// Apply per-color overrides from config.
	ov := cfg.UI.ThemeOverride
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
	code := string(theme.Code)
	bg := string(theme.Background)
	link := string(theme.Search) // reuse Search color for links
	border := string(theme.Border)

	// Pick a code block background: slightly offset from the main background.
	codeBg := "#373737"
	if isLightColor(bg) {
		codeBg = "#e8e8e8"
	}

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
				Color:           &code,
				BackgroundColor: &codeBg,
			},
		},
		CodeBlock: gansi.StyleCodeBlock{
			StyleBlock: gansi.StyleBlock{
				StylePrimitive: gansi.StylePrimitive{
					Color: &fg,
				},
				Margin: uintPtr(2),
			},
			Chroma: chromaForTheme(fg, codeBg, isLightColor(bg)),
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
// or dark backgrounds.  Colors are taken from glamour's built-in styles.
func chromaForTheme(fg, codeBg string, light bool) *gansi.Chroma {
	sp := func(color string) gansi.StylePrimitive {
		if color == "" {
			return gansi.StylePrimitive{}
		}
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
			Comment:             sp("#8D8D8D"),
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
			GenericSubheading:   sp("#777777"),
			Background:          spBg("", codeBg),
		}
	}

	return &gansi.Chroma{
		Text:                sp(fg),
		Error:               spBg("#F1F1F1", "#F05B5B"),
		Comment:             sp("#676767"),
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
		GenericSubheading:   sp("#777777"),
		Background:          spBg("", codeBg),
	}
}
