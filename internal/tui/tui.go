// Package tui provides the interactive terminal user interface for gomd.
package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/atotto/clipboard"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	gansi "charm.land/glamour/v2/ansi"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/fsnotify/fsnotify"

	"github.com/grota/gomd/internal/config"
	"github.com/grota/gomd/internal/parser"
)

// ─────────────────────────────────────────────
// Themes
// ─────────────────────────────────────────────

// Theme holds color settings for the TUI.
type Theme struct {
	Name       string
	Border     lipgloss.Color
	Selected   lipgloss.Color
	Heading1   lipgloss.Color
	Heading2   lipgloss.Color
	Heading3   lipgloss.Color
	HeadingN   lipgloss.Color
	Background lipgloss.Color
	Foreground lipgloss.Color
	StatusBar  lipgloss.Color
	Highlight  lipgloss.Color
	Code       lipgloss.Color
	Search     lipgloss.Color
	NodeSel    lipgloss.Color // selected node in interactive mode
}

// backtickRe matches `code` in markdown heading text (glamour strips backticks).
var backtickRe = regexp.MustCompile("`([^`]*)`")

// mdLinkRe matches [text](url) in markdown heading text (glamour shows only text).
var mdLinkRe = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)

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

// GetTheme returns a built-in theme by name, falling back to "OceanDark".
func GetTheme(name string) Theme {
	if t, ok := themes[name]; ok {
		return t
	}
	return themes["OceanDark"]
}

// ResolveTheme returns the active theme, applying ghostty loading and config overrides.
// It first checks built-in themes, then tries loading from the ghostty theme directory,
// and finally applies any color overrides from the config.
func ResolveTheme(cfg config.Config) Theme {
	name := cfg.UI.Theme
	var theme Theme

	if t, ok := themes[name]; ok {
		theme = t
	} else if cfg.UI.GhosttyThemeDir != "" {
		if t, err := LoadGhosttyTheme(cfg.UI.GhosttyThemeDir, name); err == nil {
			theme = t
		} else {
			theme = themes["OceanDark"]
		}
	} else {
		theme = themes["OceanDark"]
	}

	// Apply per-color overrides from config.
	ov := cfg.UI.ThemeOverride
	applyOverride := func(dst *lipgloss.Color, src string) {
		if src != "" {
			*dst = lipgloss.Color(src)
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
// Focus & Mode
// ─────────────────────────────────────────────

// FocusPane tracks which pane is active.
type FocusPane int

const (
	FocusSidebar FocusPane = iota
	FocusContent
)

// AppMode represents the current application mode.
type AppMode int

// contentMatch records the position of a search match in rendered content.
type contentMatch struct {
	line      int // rendered line index
	colStart  int // display column start
	colEnd    int // display column end
	byteStart int // byte offset in stripped line
	byteEnd   int // byte offset end
}

const (
	ModeNormal        AppMode = iota
	ModeSearch                // sidebar heading search (/)
	ModeContentSearch         // content pane text search (/)
	ModeHelp
	ModeThemePicker
	ModeNodeSelect // interactive node selection in content pane
	ModeJump       // EasyMotion-style jump labels
)

const (
	nodeKindCount = 4 // number of node sub-modes (code, inline, link, all)
	tabStopWidth  = 8 // tab expansion width
)

// ─────────────────────────────────────────────
// CodeNode — a selectable node inside a section
// ─────────────────────────────────────────────

type nodeKind int

const (
	nodeCodeBlock nodeKind = iota
	nodeInlineCode
	nodeLink
	nodeAll
)

type codeNode struct {
	kind      nodeKind
	lang      string
	content   string // raw code without fence lines / backticks; for links: URL
	display   string // for links: the visible text glamour renders (e.g. link text, "Image: alt")
	startLine int    // 0-based line index in sectionLines
	endLine   int    // inclusive (== startLine for inline)
	inline    bool   // true for backtick inline code spans and links
	colStart  int    // byte offset of opening backtick/bracket in the line (inline only)
	colEnd    int    // byte offset just past closing backtick/paren (inline only)
}

// nodeRenderLoc records where a codeNode appears in the glamour-rendered output.
type nodeRenderLoc struct {
	firstLine int // index into renderedLines; -1 = not found
	lastLine  int // inclusive end line (== firstLine for inline)
	// For inline nodes: display-column offsets for highlighting (used by highlightSpanInLine).
	spanCol    int // display column of span start (-1 if not found)
	spanColEnd int // display column just past span end
	// Byte offsets within the stripped line — used internally by mapNodesToRenderedLines
	// for the searchFromCol cursor (strings.Index returns byte offsets).
	spanColByte    int
	spanColEndByte int
}

// ─────────────────────────────────────────────
// App state
// ─────────────────────────────────────────────

type App struct {
	doc      *parser.Document
	filename string
	filepath string
	cfg      config.Config
	theme    Theme

	width  int
	height int

	// Sidebar
	sidebarHidden bool
	focus         FocusPane
	outlineOffset int // first visible entry index (0 = root "(Document)" entry)
	selectedIdx   int // -1 = root/whole-document, 0..N-1 = heading index

	// Content — shows only the section of the selected heading
	sectionLines  []string // raw lines of the current section (used for node extraction)
	renderedLines []string // glamour-rendered lines of the current section
	contentOffset int      // scroll offset (into renderedLines in normal mode, sectionLines in node-select)

	// glamour renderer — rebuilt when content width or theme changes
	glamourRenderer  interface{ Render(string) (string, error) }
	glamourWidth     int    // innerW for which renderer was built
	glamourStyleName string // gomd theme name for which renderer was built

	// rendered lines cache — rebuilt when section or width changes
	renderedLinesWidth int // innerW for which renderedLines was built
	renderedLinesIdx   int // selectedIdx for which renderedLines was built
	// nodeRenderInfo[i] describes where codeNodes[i] appears in renderedLines.
	// Rebuilt whenever renderedLines is rebuilt.
	nodeRenderInfo []nodeRenderLoc

	// Search (sidebar heading search)
	mode          AppMode
	searchQuery   string
	searchMatches []int
	searchIdx     int

	// Content search
	contentSearchQuery   string
	contentSearchMatches []contentMatch // all matches in rendered lines
	contentSearchIdx     int            // current match index

	// Interactive node selection
	codeNodes   []codeNode // all selectable nodes in current section
	nodeSelIdx  int        // which node is highlighted (within filtered list)
	nodeSubMode nodeKind   // current sub-mode filter
	copyMsg     string     // transient "Copied!" feedback

	// File watcher
	watcher *fsnotify.Watcher

	// Navigation history
	navHistory []int // stack of previous selectedIdx values
	navFuture  []int // forward stack

	// Jump (EasyMotion) state
	jumpLabels map[string]int // label -> node index in codeNodes
	jumpInput  string         // characters typed so far

	// Pending key prefix for multi-key sequences (gg, zz, zt, zb)
	pendingKey string

	// Status
	statusMsg string
}

// ─────────────────────────────────────────────
// Construction
// ─────────────────────────────────────────────

func NewApp(doc *parser.Document, filename, filePath string, cfg config.Config) *App {
	a := &App{
		doc:           doc,
		filename:      filename,
		filepath:      filePath,
		cfg:           cfg,
		theme:         ResolveTheme(cfg),
		focus:         FocusContent,
		selectedIdx:   -1, // start on the root (Document) node
		sidebarHidden: cfg.UI.SidebarHidden,
	}
	a.rebuildSection()
	return a
}

// ─────────────────────────────────────────────
// Section helpers
// ─────────────────────────────────────────────

// rebuildSection recomputes sectionLines and codeNodes for the selected heading.

func boolPtr(b bool) *bool       { return &b }
func stringPtr(s string) *string { return &s }
func uintPtr(u uint) *uint       { return &u }

// StyleConfigFromTheme builds a glamour StyleConfig from a gomd Theme,
// mapping the theme's colors directly into the glamour rendering style.
// This replaces the old approach of picking a built-in glamour style.
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
			Format: "{{.text}} 🖼️",
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

// BGOpen returns the ANSI escape sequence to set a truecolor background from
// a hex color string like "#ffffff".  Exported for use by the render-mode
// post-processor.
func BGOpen(hex string) string {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return ""
	}
	r, err1 := strconv.ParseUint(hex[0:2], 16, 8)
	g, err2 := strconv.ParseUint(hex[2:4], 16, 8)
	b, err3 := strconv.ParseUint(hex[4:6], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return ""
	}
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
}

// isLightColor returns true if a hex color string (e.g., "#f7f7f7") is
// perceptually light (luminance > 0.5).
func isLightColor(hex string) bool {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return false
	}
	r, err1 := strconv.ParseUint(hex[0:2], 16, 8)
	g, err2 := strconv.ParseUint(hex[2:4], 16, 8)
	b, err3 := strconv.ParseUint(hex[4:6], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return false
	}
	// Relative luminance (simplified sRGB)
	lum := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
	return lum > 128
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

// ensureGlamourRenderer returns a cached glamour renderer for the given inner width
// and current theme, rebuilding it only when those change.
func (a *App) ensureGlamourRenderer(innerW int) interface{ Render(string) (string, error) } {
	themeName := a.theme.Name
	if a.glamourRenderer != nil && a.glamourWidth == innerW && a.glamourStyleName == themeName {
		return a.glamourRenderer
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(StyleConfigFromTheme(a.theme)),
		glamour.WithWordWrap(innerW),
		glamour.WithChromaFormatter("terminal16m"),
	)
	if err != nil {
		return nil
	}
	a.glamourRenderer = r
	a.glamourWidth = innerW
	a.glamourStyleName = themeName
	return r
}

// expandTabs replaces tab characters in s with spaces, advancing to the next
// 8-column tab stop. ANSI escape sequences are skipped (zero-width) so column
// counting stays accurate even in styled output.
func expandTabs(s string) string {
	if !strings.ContainsRune(s, '\t') {
		return s
	}
	var buf strings.Builder
	col := 0
	inEsc := false
	for _, r := range s {
		switch {
		case inEsc:
			buf.WriteRune(r)
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEsc = false
			}
		case r == '\x1b':
			inEsc = true
			buf.WriteRune(r)
		case r == '\t':
			spaces := tabStopWidth - (col % tabStopWidth)
			for i := 0; i < spaces; i++ {
				buf.WriteByte(' ')
			}
			col += spaces
		default:
			buf.WriteRune(r)
			col++
		}
	}
	return buf.String()
}

// imgTagRe matches <img ... /> or <img ...> HTML tags.
var imgTagRe = regexp.MustCompile(`(?i)<img\s[^>]*>`)

// aTagRe matches <a href="...">text</a> HTML tags.
var aTagRe = regexp.MustCompile(`(?i)<a\s[^>]*href\s*=\s*["']([^"']*)["'][^>]*>(.*?)</a>`)

// PreProcessMarkdown converts HTML image and anchor tags into their markdown
// equivalents so that glamour renders them properly. This must be called
// before passing content to glamour.
func PreProcessMarkdown(md string) string {
	// Replace <img ... src="url" alt="text" ... /> with ![text](url)
	md = imgTagRe.ReplaceAllStringFunc(md, func(tag string) string {
		src := extractHTMLAttr(tag, "src")
		if src == "" {
			return tag
		}
		alt := extractHTMLAttr(tag, "alt")
		return "![" + alt + "](" + src + ")"
	})

	// Replace <a href="url">text</a> with [text](url)
	md = aTagRe.ReplaceAllString(md, "[$2]($1)")

	return md
}

// PostProcessLinks inserts a 🔗 glyph between link text and URL in
// glamour-rendered output. Glamour renders [text](url) as:
//
//	OSC8(url) STYLED_TEXT RESET OSC8_RESET FG_STYLE SPACE RESET LINK_STYLE OSC8(url) URL ...
//
// We find the OSC8 reset after the text and insert " 🔗" before the space.
var linkGlyphRe = regexp.MustCompile(`(\x1b\]8;;\x07)((?:\x1b\[[0-9;]*m)*)([ \t])((?:\x1b\[[0-9;]*m)*\x1b\]8;id=)`)

// PostProcessLinks adds a link glyph between the text and URL portions of
// rendered links. It should be called on the full rendered output string.
func PostProcessLinks(s string) string {
	return linkGlyphRe.ReplaceAllString(s, "${1} 🔗${2}${3}${4}")
}

// renderGlamour renders a markdown string through glamour and returns the output
// split into lines, with a trailing empty line stripped.
func (a *App) renderGlamour(markdown string, innerW int) []string {
	r := a.ensureGlamourRenderer(innerW)
	if r == nil {
		return strings.Split(markdown, "\n")
	}
	markdown = PreProcessMarkdown(markdown)
	out, err := r.Render(markdown)
	if err != nil {
		return strings.Split(markdown, "\n")
	}
	out = PostProcessLinks(out)

	// glamour always ends with a newline; split and drop the trailing empty entry
	lines := strings.Split(out, "\n")
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	// Expand any tab characters left in the glamour output so that
	// lipgloss.Width() and ansi.Truncate() measure the correct column widths.
	for i, l := range lines {
		lines[i] = expandTabs(l)
	}
	return lines
}

func (a *App) rebuildSection() {
	a.contentOffset = 0
	a.nodeSelIdx = 0
	a.codeNodes = nil
	a.nodeRenderInfo = nil
	a.renderedLines = nil

	// selectedIdx == -1 means the root (Document) node: show entire file.
	//
	// Node extraction (extractCodeNodes → extractLinkNodes) runs on the raw
	// markdown lines, *before* PreProcessMarkdown converts HTML tags to
	// markdown equivalents. This is intentional: the extracted colStart/colEnd
	// byte offsets must match the raw source so that highlightSpanInLine can
	// highlight the correct column range in the rendered output. Running
	// extraction after pre-processing would shift those offsets because
	// PreProcessMarkdown changes line content and length (e.g. a long <img>
	// tag becomes a shorter ![alt](url)). The trade-off is that extraction
	// must handle both markdown and HTML syntax for links/images.
	if a.selectedIdx < 0 || len(a.doc.Headings) == 0 {
		content := strings.TrimRight(a.doc.Content, "\n")
		a.sectionLines = strings.Split(content, "\n")
		a.codeNodes = extractCodeNodes(a.sectionLines)
		return
	}

	if a.selectedIdx >= len(a.doc.Headings) {
		a.selectedIdx = len(a.doc.Headings) - 1
	}

	h := a.doc.Headings[a.selectedIdx]

	// Byte range for section
	start := h.Offset
	end := len(a.doc.Content)
	for i := a.selectedIdx + 1; i < len(a.doc.Headings); i++ {
		if a.doc.Headings[i].Level <= h.Level {
			end = a.doc.Headings[i].Offset
			break
		}
	}

	section := a.doc.Content[start:end]
	section = strings.TrimRight(section, "\n")
	a.sectionLines = strings.Split(section, "\n")
	a.codeNodes = extractCodeNodes(a.sectionLines)
}

// extractCodeNodes finds all fenced code blocks and inline backtick spans in lines.
func extractCodeNodes(lines []string) []codeNode {
	var nodes []codeNode
	inBlock := false
	var fence, lang string
	var bodyLines []string
	var startLine int

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inBlock {
			if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
				inBlock = true
				fence = trimmed[:3]
				lang = strings.TrimSpace(trimmed[3:])
				bodyLines = nil
				startLine = i
			} else {
				// Scan for inline backtick spans and links (including on heading lines)
				nodes = append(nodes, extractInlineNodes(line, i)...)
				nodes = append(nodes, extractLinkNodes(line, i)...)
			}
		} else {
			if strings.HasPrefix(trimmed, fence) {
				nodes = append(nodes, codeNode{
					kind:      nodeCodeBlock,
					lang:      lang,
					content:   strings.Join(bodyLines, "\n"),
					startLine: startLine,
					endLine:   i,
				})
				inBlock = false
				fence = ""
				lang = ""
				bodyLines = nil
			} else {
				bodyLines = append(bodyLines, line)
			}
		}
	}
	return nodes
}

// extractInlineNodes finds all `code` spans in a single line and returns them as codeNodes.
func extractInlineNodes(line string, lineIdx int) []codeNode {
	var nodes []codeNode
	i := 0
	for i < len(line) {
		if line[i] != '`' {
			i++
			continue
		}
		// Count opening backticks
		j := i
		for j < len(line) && line[j] == '`' {
			j++
		}
		tick := line[i:j] // one or more backticks
		// Find matching closing sequence
		k := j
		for k < len(line) {
			closeStart := strings.Index(line[k:], tick)
			if closeStart == -1 {
				break
			}
			closeStart += k
			// Make sure it's not longer than tick (e.g. `` inside ``` context)
			// Check that the char before or after isn't another backtick
			closeEnd := closeStart + len(tick)
			// If surrounded by more backticks, skip
			if closeStart > 0 && line[closeStart-1] == '`' {
				k = closeStart + 1
				continue
			}
			if closeEnd < len(line) && line[closeEnd] == '`' {
				k = closeEnd
				continue
			}
			// Found a valid span
			inner := line[j:closeStart]
			// Trim single leading/trailing space per CommonMark
			if len(inner) > 2 && inner[0] == ' ' && inner[len(inner)-1] == ' ' {
				inner = inner[1 : len(inner)-1]
			}
			nodes = append(nodes, codeNode{
				kind:      nodeInlineCode,
				lang:      "",
				content:   inner,
				startLine: lineIdx,
				endLine:   lineIdx,
				inline:    true,
				colStart:  i,
				colEnd:    closeEnd,
			})
			i = closeEnd
			goto nextChar
		}
		i = j
	nextChar:
	}
	return nodes
}

// extractLinkNodes finds markdown links [text](url) and autolinks <url> in a line.
func extractLinkNodes(line string, lineIdx int) []codeNode {
	var nodes []codeNode
	i := 0
	for i < len(line) {
		// Check for ![alt](url) image link or [text](url) regular link
		if line[i] == '!' && i+1 < len(line) && line[i+1] == '[' {
			// Image link ![alt](url)
			j := i + 2
			depth := 1
			for j < len(line) && depth > 0 {
				if line[j] == '[' {
					depth++
				} else if line[j] == ']' {
					depth--
				}
				j++
			}
			if depth == 0 && j < len(line) && line[j] == '(' {
				k := j + 1
				for k < len(line) && line[k] != ')' {
					k++
				}
				if k < len(line) {
					url := line[j+1 : k]
					alt := line[i+2 : j-1]
					// Glamour renders ![alt](url) as "alt 🖼️ path" (custom format)
					display := alt + " 🖼️"
					if alt == "" {
						display = url
					}
					nodes = append(nodes, codeNode{
						kind:      nodeLink,
						lang:      "link",
						content:   url,
						display:   display,
						startLine: lineIdx,
						endLine:   lineIdx,
						inline:    true,
						colStart:  i,
						colEnd:    k + 1,
					})
					i = k + 1
					continue
				}
			}
			i = j
		} else if line[i] == '[' {
			// Regular link [text](url)
			j := i + 1
			depth := 1
			for j < len(line) && depth > 0 {
				if line[j] == '[' {
					depth++
				} else if line[j] == ']' {
					depth--
				}
				j++
			}
			if depth == 0 && j < len(line) && line[j] == '(' {
				k := j + 1
				for k < len(line) && line[k] != ')' {
					k++
				}
				if k < len(line) {
					url := line[j+1 : k]
					text := line[i+1 : j-1]
					// Glamour renders [text](url) as "text" (with OSC8 hyperlink)
					nodes = append(nodes, codeNode{
						kind:      nodeLink,
						lang:      "link",
						content:   url,
						display:   text,
						startLine: lineIdx,
						endLine:   lineIdx,
						inline:    true,
						colStart:  i,
						colEnd:    k + 1,
					})
					i = k + 1
					continue
				}
			}
			i = j
		} else if line[i] == '<' && i+1 < len(line) {
			// Check for <img ... src="url" ... /> HTML tag
			if i+4 < len(line) && strings.ToLower(line[i+1:i+4]) == "img" && (line[i+4] == ' ' || line[i+4] == '\t') {
				// Find the closing >
				closeIdx := strings.Index(line[i:], ">")
				if closeIdx >= 0 {
					tag := line[i : i+closeIdx+1]
					// Extract src attribute
					src := extractHTMLAttr(tag, "src")
					alt := extractHTMLAttr(tag, "alt")
					if src != "" {
						display := alt + " 🖼️"
						if alt == "" {
							display = src
						}
						nodes = append(nodes, codeNode{
							kind:      nodeLink,
							lang:      "link",
							content:   src,
							display:   display,
							startLine: lineIdx,
							endLine:   lineIdx,
							inline:    true,
							colStart:  i,
							colEnd:    i + closeIdx + 1,
						})
					}
					i = i + closeIdx + 1
					continue
				}
			}
			// Check for <a href="url">text</a> HTML tag
			if i+2 < len(line) && strings.ToLower(line[i+1:i+2]) == "a" && i+2 < len(line) && (line[i+2] == ' ' || line[i+2] == '\t') {
				m := aTagRe.FindStringIndex(line[i:])
				if m != nil {
					sm := aTagRe.FindStringSubmatch(line[i:])
					if sm != nil {
						href := sm[1]
						text := sm[2]
						nodes = append(nodes, codeNode{
							kind:      nodeLink,
							lang:      "link",
							content:   href,
							display:   text,
							startLine: lineIdx,
							endLine:   lineIdx,
							inline:    true,
							colStart:  i,
							colEnd:    i + m[1],
						})
						i = i + m[1]
						continue
					}
				}
			}
			// Check for autolink <url>
			j := i + 1
			for j < len(line) && line[j] != '>' && line[j] != ' ' {
				j++
			}
			if j < len(line) && line[j] == '>' {
				url := line[i+1 : j]
				// Basic validation: contains :// or @ (link/email)
				if strings.Contains(url, "://") || strings.Contains(url, "@") {
					nodes = append(nodes, codeNode{
						kind:      nodeLink,
						lang:      "link",
						content:   url,
						display:   url,
						startLine: lineIdx,
						endLine:   lineIdx,
						inline:    true,
						colStart:  i,
						colEnd:    j + 1,
					})
				}
				i = j + 1
			} else {
				i++
			}
		} else if i+8 < len(line) && (line[i:i+8] == "https://" || line[i:i+7] == "http://") {
			// Bare URL (not inside [] or <>)
			j := i
			for j < len(line) && line[j] != ' ' && line[j] != '\t' && line[j] != ')' && line[j] != '>' {
				j++
			}
			// Trim trailing punctuation that's unlikely part of the URL
			for j > i && (line[j-1] == '.' || line[j-1] == ',' || line[j-1] == ';' || line[j-1] == ':') {
				j--
			}
			url := line[i:j]
			nodes = append(nodes, codeNode{
				kind:      nodeLink,
				lang:      "link",
				content:   url,
				display:   url,
				startLine: lineIdx,
				endLine:   lineIdx,
				inline:    true,
				colStart:  i,
				colEnd:    j,
			})
			i = j
		} else {
			i++
		}
	}
	return nodes
}

// extractHTMLAttr extracts the value of an HTML attribute from a tag string.
// Returns empty string if the attribute is not found.
func extractHTMLAttr(tag, attr string) string {
	// Search for attr= (case-insensitive)
	lower := strings.ToLower(tag)
	key := strings.ToLower(attr) + "="
	idx := strings.Index(lower, key)
	if idx < 0 {
		return ""
	}
	rest := tag[idx+len(key):]
	if len(rest) == 0 {
		return ""
	}
	if rest[0] == '"' || rest[0] == '\'' {
		quote := rest[0]
		end := strings.IndexByte(rest[1:], quote)
		if end < 0 {
			return ""
		}
		return rest[1 : 1+end]
	}
	// Unquoted: read until space or >
	end := strings.IndexAny(rest, " \t>")
	if end < 0 {
		return rest
	}
	return rest[:end]
}

// filteredNodeIndices returns the indices into a.codeNodes for each node
// matching the current sub-mode filter.
func (a *App) filteredNodeIndices() []int {
	var indices []int
	for i, n := range a.codeNodes {
		if a.nodeSubMode == nodeAll || n.kind == a.nodeSubMode {
			indices = append(indices, i)
		}
	}
	return indices
}

// filteredNodes returns only the codeNodes matching the current sub-mode.
func (a *App) filteredNodes() []codeNode {
	indices := a.filteredNodeIndices()
	result := make([]codeNode, len(indices))
	for i, idx := range indices {
		result[i] = a.codeNodes[idx]
	}
	return result
}

// ─────────────────────────────────────────────
// Bubbletea lifecycle
// ─────────────────────────────────────────────

type fileReloadMsg struct{ doc *parser.Document }
type watchErrMsg struct{ err error }
type clearCopyMsgCmd struct{}
type editorDoneMsg struct{}

func (clearCopyMsgCmd) ID() string { return "" }

// openEditorCmd suspends the TUI, opens $EDITOR on filePath, then signals done.
func openEditorCmd(filePath string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	c := exec.Command(editor, filePath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorDoneMsg{}
	})
}

func waitForFileChange(watcher *fsnotify.Watcher, filePath string) tea.Cmd {
	return func() tea.Msg {
		for event := range watcher.Events {
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				data, err := os.ReadFile(filePath)
				if err != nil {
					continue
				}
				return fileReloadMsg{parser.ParseMarkdown(string(data))}
			}
		}
		return nil
	}
}

func (a *App) startWatcher() tea.Cmd {
	if a.filepath == "" || a.filepath == "<stdin>" {
		return nil
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil
	}
	if err := watcher.Add(a.filepath); err != nil {
		watcher.Close()
		return nil
	}
	if a.watcher != nil {
		a.watcher.Close()
	}
	a.watcher = watcher
	return waitForFileChange(watcher, a.filepath)
}

func (a *App) Init() tea.Cmd {
	return a.startWatcher()
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

	case fileReloadMsg:
		prevIdx := a.selectedIdx
		prevHeading := ""
		if prevIdx >= 0 && prevIdx < len(a.doc.Headings) {
			prevHeading = a.doc.Headings[prevIdx].Text
		}
		a.doc = msg.doc
		if prevIdx < 0 {
			// was on root, stay on root
			a.selectedIdx = -1
		} else if prevHeading != "" {
			// try to restore by heading text
			found := false
			for i, h := range a.doc.Headings {
				if h.Text == prevHeading {
					a.selectedIdx = i
					found = true
					break
				}
			}
			if !found {
				a.selectedIdx = -1
			}
		}
		if a.selectedIdx >= len(a.doc.Headings) {
			a.selectedIdx = len(a.doc.Headings) - 1
		}
		a.rebuildSection()
		a.scrollOutlineToSelected()
		a.statusMsg = "Reloaded"
		return a, a.startWatcher()

	case editorDoneMsg:
		// Reload file after editor exits
		if a.filepath != "" && a.filepath != "<stdin>" {
			data, err := os.ReadFile(a.filepath)
			if err == nil {
				a.doc = parser.ParseMarkdown(string(data))
				a.rebuildSection()
				a.scrollOutlineToSelected()
				a.statusMsg = "Reloaded after edit"
			}
		}
		return a, a.startWatcher()

	case tea.KeyPressMsg:
		return a.handleKey(msg)

	case tea.MouseWheelMsg:
		return a.handleMouseWheel(msg)
	case tea.MouseClickMsg:
		return a.handleMouseClick(msg)
	}

	return a, nil
}

// ─────────────────────────────────────────────
// Key handling — dispatched by mode then focus
// ─────────────────────────────────────────────

func (a *App) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := msg.String()

	// Global keys that work in any mode
	switch k {
	case "ctrl+c":
		if a.watcher != nil {
			a.watcher.Close()
		}
		return a, tea.Quit
	}

	switch a.mode {
	case ModeSearch:
		return a.handleSearchKey(msg)
	case ModeContentSearch:
		return a.handleContentSearchKey(msg)
	case ModeHelp:
		a.mode = ModeNormal
		return a, nil
	case ModeThemePicker:
		return a.handleThemeKey(msg)
	case ModeNodeSelect:
		return a.handleNodeSelectKey(msg)
	case ModeJump:
		return a.handleJumpKey(msg)
	}

	// Normal mode — shared keys regardless of focus
	// If a shared key matches, cancel any pending key sequence.
	km := a.cfg.Keys
	switch {
	case config.KeyMatches(k, km.Quit):
		a.pendingKey = ""
		if a.watcher != nil {
			a.watcher.Close()
		}
		return a, tea.Quit
	case config.KeyMatches(k, km.Help):
		a.mode = ModeHelp
		return a, nil
	case config.KeyMatches(k, km.ThemePicker):
		a.mode = ModeThemePicker
		return a, nil
	case config.KeyMatches(k, km.ToggleFocus):
		a.toggleFocus()
		return a, nil
	case config.KeyMatches(k, km.ToggleSidebar):
		a.sidebarHidden = !a.sidebarHidden
		if a.sidebarHidden {
			a.focus = FocusContent
		}
		return a, nil
	case config.KeyMatches(k, km.Reload):
		if a.filepath != "" && a.filepath != "<stdin>" {
			data, err := os.ReadFile(a.filepath)
			if err == nil {
				a.doc = parser.ParseMarkdown(string(data))
				a.rebuildSection()
				a.scrollOutlineToSelected()
				a.statusMsg = "Reloaded"
			}
		}
		return a, nil
	case config.KeyMatches(k, km.Edit):
		if a.filepath != "" && a.filepath != "<stdin>" {
			return a, openEditorCmd(a.filepath)
		}
		a.statusMsg = "No file to edit"
		return a, nil
	case config.KeyMatches(k, km.NavBack):
		a.navGoBack()
		return a, nil
	case config.KeyMatches(k, km.NavForward):
		a.navGoForward()
		return a, nil
	}

	// Focus-specific keys
	if a.focus == FocusSidebar && !a.sidebarHidden {
		return a.handleSidebarKey(msg)
	}
	return a.handleContentKey(msg)
}

func (a *App) toggleFocus() {
	if a.sidebarHidden {
		return
	}
	if a.focus == FocusSidebar {
		a.focus = FocusContent
	} else {
		a.focus = FocusSidebar
	}
}

// handleMouseWheel handles mouse wheel events for content scrolling.
func (a *App) handleMouseWheel(msg tea.MouseWheelMsg) (tea.Model, tea.Cmd) {
	if a.mode != ModeNormal {
		return a, nil
	}

	switch msg.Button {
	case tea.MouseWheelUp:
		if a.focus == FocusContent || a.sidebarHidden {
			a.contentOffset -= 3
			if a.contentOffset < 0 {
				a.contentOffset = 0
			}
		}
	case tea.MouseWheelDown:
		if a.focus == FocusContent || a.sidebarHidden {
			a.contentOffset += 3
			a.clampContentOffset()
		}
	}
	return a, nil
}

// handleMouseClick handles mouse click events for sidebar clicks.
func (a *App) handleMouseClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if a.mode != ModeNormal {
		return a, nil
	}

	if msg.Button == tea.MouseLeft {
		mouse := msg.Mouse()
		// Check if click is in the sidebar
		sw := a.sidebarWidth()
		if !a.sidebarHidden && mouse.X < sw && mouse.Y >= 2 && mouse.Y < a.height-1 {
			// Click in sidebar — determine which entry was clicked
			row := mouse.Y - 2 // subtract title(1) + border-top(1)
			entryIdx := a.outlineOffset + row
			if entryIdx >= 0 && entryIdx < a.totalEntries() {
				newIdx := entryIdx - 1 // entryIdx 0 = root (-1)
				a.navigateTo(newIdx)
				a.focus = FocusSidebar
			}
		} else if mouse.X >= sw {
			a.focus = FocusContent
		}
	}
	return a, nil
}

// ─────────────────────────────────────────────
// Sidebar keys
// ─────────────────────────────────────────────

func (a *App) handleSidebarKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	km := a.cfg.Keys

	// Handle pending key sequences (gg, zz, zt, zb)
	if a.pendingKey != "" {
		combo := a.pendingKey + k
		a.pendingKey = ""
		switch {
		case config.KeyMatches(combo, km.SidebarTop):
			a.selectedIdx = -1
			a.scrollOutlineToSelected()
			a.rebuildSection()
		case config.KeyMatches(combo, km.ViewCenter):
			a.sidebarViewCenter()
		case config.KeyMatches(combo, km.ViewTop):
			a.sidebarViewTop()
		case config.KeyMatches(combo, km.ViewBottom):
			a.sidebarViewBottom()
		}
		return a, nil
	}

	// Check for prefix keys
	if k == "g" || k == "z" {
		a.pendingKey = k
		return a, nil
	}

	switch {
	case config.KeyMatches(k, km.SidebarDown):
		a.moveSidebarDown(1)
	case config.KeyMatches(k, km.SidebarUp):
		a.moveSidebarUp(1)
	case k == "pgdown":
		a.moveSidebarDown(a.outlineHeight())
	case k == "pgup":
		a.moveSidebarUp(a.outlineHeight())
	case config.KeyMatches(k, km.SidebarBottom):
		if len(a.doc.Headings) > 0 {
			a.selectedIdx = len(a.doc.Headings) - 1
		}
		a.scrollOutlineToSelected()
		a.rebuildSection()
	case config.KeyMatches(k, km.JumpHigh):
		a.sidebarJumpHigh()
	case config.KeyMatches(k, km.JumpMid):
		a.sidebarJumpMid()
	case config.KeyMatches(k, km.JumpLow):
		a.sidebarJumpLow()
	case config.KeyMatches(k, km.SidebarSearch):
		a.mode = ModeSearch
		a.searchQuery = ""
		a.searchMatches = nil
	case config.KeyMatches(k, km.NextMatch):
		a.nextSearchMatch()
	case config.KeyMatches(k, km.PrevMatch):
		a.prevSearchMatch()
	}
	return a, nil
}

func (a *App) moveSidebarDown(n int) {
	prev := a.selectedIdx
	a.selectedIdx += n
	if a.selectedIdx >= len(a.doc.Headings) {
		a.selectedIdx = len(a.doc.Headings) - 1
	}
	if a.selectedIdx != prev {
		a.scrollOutlineToSelected()
		a.rebuildSection()
	}
}

func (a *App) moveSidebarUp(n int) {
	prev := a.selectedIdx
	a.selectedIdx -= n
	if a.selectedIdx < -1 {
		a.selectedIdx = -1
	}
	if a.selectedIdx != prev {
		a.scrollOutlineToSelected()
		a.rebuildSection()
	}
}

// sidebarJumpHigh jumps to the topmost visible heading in the sidebar.
func (a *App) sidebarJumpHigh() {
	if len(a.doc.Headings) == 0 {
		return
	}
	// First visible heading is at outlineOffset
	idx := a.outlineOffset
	if idx < 0 {
		idx = 0
	}
	if idx >= len(a.doc.Headings) {
		idx = len(a.doc.Headings) - 1
	}
	a.selectedIdx = idx
	a.rebuildSection()
}

// sidebarJumpMid jumps to the middle visible heading in the sidebar.
func (a *App) sidebarJumpMid() {
	if len(a.doc.Headings) == 0 {
		return
	}
	h := a.outlineHeight()
	top := a.outlineOffset
	bot := top + h - 1
	if bot >= len(a.doc.Headings) {
		bot = len(a.doc.Headings) - 1
	}
	if top < 0 {
		top = 0
	}
	a.selectedIdx = (top + bot) / 2
	a.rebuildSection()
}

// sidebarJumpLow jumps to the bottommost visible heading in the sidebar.
func (a *App) sidebarJumpLow() {
	if len(a.doc.Headings) == 0 {
		return
	}
	h := a.outlineHeight()
	bot := a.outlineOffset + h - 1
	if bot >= len(a.doc.Headings) {
		bot = len(a.doc.Headings) - 1
	}
	a.selectedIdx = bot
	a.rebuildSection()
}

// sidebarViewCenter scrolls the sidebar so the selected heading is centered.
func (a *App) sidebarViewCenter() {
	h := a.outlineHeight()
	a.outlineOffset = a.selectedIdx - h/2
	a.clampOutlineOffset()
}

// sidebarViewTop scrolls the sidebar so the selected heading is at the top.
func (a *App) sidebarViewTop() {
	a.outlineOffset = a.selectedIdx
	a.clampOutlineOffset()
}

// sidebarViewBottom scrolls the sidebar so the selected heading is at the bottom.
func (a *App) sidebarViewBottom() {
	h := a.outlineHeight()
	a.outlineOffset = a.selectedIdx - h + 1
	a.clampOutlineOffset()
}

// clampOutlineOffset ensures outlineOffset stays within valid bounds.
func (a *App) clampOutlineOffset() {
	if a.outlineOffset < 0 {
		a.outlineOffset = 0
	}
	max := len(a.doc.Headings) - a.outlineHeight()
	if max < 0 {
		max = 0
	}
	if a.outlineOffset > max {
		a.outlineOffset = max
	}
}

// navigateTo changes the selected heading with history tracking.
func (a *App) navigateTo(idx int) {
	if idx == a.selectedIdx {
		return
	}
	a.navHistory = append(a.navHistory, a.selectedIdx)
	a.navFuture = nil
	a.selectedIdx = idx
	a.scrollOutlineToSelected()
	a.rebuildSection()
}

// navGoBack navigates to the previous heading in history.
func (a *App) navGoBack() {
	if len(a.navHistory) == 0 {
		return
	}
	a.navFuture = append(a.navFuture, a.selectedIdx)
	a.selectedIdx = a.navHistory[len(a.navHistory)-1]
	a.navHistory = a.navHistory[:len(a.navHistory)-1]
	a.scrollOutlineToSelected()
	a.rebuildSection()
}

// navGoForward navigates forward in history.
func (a *App) navGoForward() {
	if len(a.navFuture) == 0 {
		return
	}
	a.navHistory = append(a.navHistory, a.selectedIdx)
	a.selectedIdx = a.navFuture[len(a.navFuture)-1]
	a.navFuture = a.navFuture[:len(a.navFuture)-1]
	a.scrollOutlineToSelected()
	a.rebuildSection()
}

// findHeadingByAnchor finds a heading index by GitHub-style anchor slug.
func (a *App) findHeadingByAnchor(anchor string) int {
	anchor = strings.ToLower(anchor)
	for i, h := range a.doc.Headings {
		if headingToAnchor(h.Text) == anchor {
			return i
		}
	}
	return -1
}

// headingToAnchor converts heading text to a GitHub-style anchor slug.
func headingToAnchor(text string) string {
	text = strings.ToLower(text)
	var sb strings.Builder
	for _, r := range text {
		if r == ' ' || r == '-' {
			sb.WriteByte('-')
		} else if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// displayIdx returns the sidebar display index for the current selection.
// Root (selectedIdx==-1) → 0; heading i → i+1.
func (a *App) displayIdx() int {
	return a.selectedIdx + 1
}

// totalEntries returns the total number of sidebar entries (root + headings).
func (a *App) totalEntries() int {
	return len(a.doc.Headings) + 1
}

// scrollOutlineToSelected scrolls the outline viewport so the selected item is
// always visible, but does NOT move the viewport unless the selection has left it.
func (a *App) scrollOutlineToSelected() {
	h := a.outlineHeight()
	if h <= 0 {
		return
	}
	di := a.displayIdx()
	if di < a.outlineOffset {
		a.outlineOffset = di
	}
	if di >= a.outlineOffset+h {
		a.outlineOffset = di - h + 1
	}
	if a.outlineOffset < 0 {
		a.outlineOffset = 0
	}
}

// ─────────────────────────────────────────────
// Content keys
// ─────────────────────────────────────────────

func (a *App) handleContentKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	km := a.cfg.Keys

	// Handle pending key sequences (gg, zz, zt, zb)
	if a.pendingKey != "" {
		combo := a.pendingKey + k
		a.pendingKey = ""
		switch {
		case config.KeyMatches(combo, km.ContentTop):
			a.contentOffset = 0
		case config.KeyMatches(combo, km.ViewCenter):
			a.contentViewCenter()
		case config.KeyMatches(combo, km.ViewTop):
			a.contentViewTop()
		case config.KeyMatches(combo, km.ViewBottom):
			a.contentViewBottom()
		}
		return a, nil
	}

	// Check for prefix keys
	if k == "g" || k == "z" {
		a.pendingKey = k
		return a, nil
	}

	switch {
	case config.KeyMatches(k, km.ScrollDown):
		a.scrollContent(1)
	case config.KeyMatches(k, km.ScrollUp):
		a.scrollContent(-1)
	case config.KeyMatches(k, km.ScrollHalfDown):
		a.scrollContent(a.contentHeight() / 2)
	case config.KeyMatches(k, km.ScrollHalfUp):
		a.scrollContent(-a.contentHeight() / 2)
	case config.KeyMatches(k, km.ContentBottom):
		a.contentOffset = len(a.activeLines())
		a.clampContentOffset()
	case config.KeyMatches(k, km.JumpHigh):
		// H: no-op in content scroll mode (no selectable items)
	case config.KeyMatches(k, km.JumpMid):
		// M: no-op in content scroll mode
	case config.KeyMatches(k, km.JumpLow):
		// L: no-op in content scroll mode
	case config.KeyMatches(k, km.NodeSelect):
		a.enterNodeSelectMode()
	case config.KeyMatches(k, km.ContentSearch):
		a.mode = ModeContentSearch
		a.contentSearchQuery = ""
		a.contentSearchMatches = nil
		a.contentSearchIdx = 0
	case config.KeyMatches(k, km.ContentNextMatch):
		a.nextContentSearchMatch()
	case config.KeyMatches(k, km.ContentPrevMatch):
		a.prevContentSearchMatch()
	case config.KeyMatches(k, km.Jump):
		a.enterJumpMode()
	}
	return a, nil
}

// contentViewCenter scrolls so the middle of the viewport stays centered (no-op in pure scroll mode,
// but useful as a viewport reset). In content mode without selection, centers the current view.
func (a *App) contentViewCenter() {
	// No selected item in content scroll mode — no-op
}

func (a *App) contentViewTop() {
	// No selected item in content scroll mode — no-op
}

func (a *App) contentViewBottom() {
	// No selected item in content scroll mode — no-op
}

func (a *App) scrollContent(delta int) {
	a.contentOffset += delta
	a.clampContentOffset()
}

func (a *App) activeLines() []string {
	if len(a.renderedLines) == 0 {
		return a.sectionLines
	}
	return a.renderedLines
}

func (a *App) clampContentOffset() {
	if a.contentOffset < 0 {
		a.contentOffset = 0
	}
	max := len(a.activeLines()) - a.contentHeight()
	if max < 0 {
		max = 0
	}
	if a.contentOffset > max {
		a.contentOffset = max
	}
}

// ─────────────────────────────────────────────
// Node selection mode
// ─────────────────────────────────────────────

// enterNodeSelectMode transitions to ModeNodeSelect, picking the first sub-mode
// that has nodes and scrolling to the first visible node at the current offset.
func (a *App) enterNodeSelectMode() {
	if len(a.codeNodes) == 0 {
		a.statusMsg = "No selectable nodes in this section"
		return
	}

	a.mode = ModeNodeSelect
	a.nodeSubMode = nodeAll
	a.nodeSelIdx = 0

	// Find the first sub-mode that has nodes
	if len(a.filteredNodeIndices()) == 0 {
		for _, mode := range []nodeKind{nodeCodeBlock, nodeInlineCode, nodeLink} {
			a.nodeSubMode = mode
			if len(a.filteredNodeIndices()) > 0 {
				break
			}
		}
	}

	indices := a.filteredNodeIndices()
	if len(indices) == 0 {
		a.mode = ModeNormal
		a.statusMsg = "No selectable nodes in this section"
		return
	}

	// Find the first filtered node visible at the current scroll position
	a.ensureRenderedLines()
	for i, idx := range indices {
		if a.nodeRenderInfo != nil && idx < len(a.nodeRenderInfo) && a.nodeRenderInfo[idx].firstLine >= a.contentOffset {
			a.nodeSelIdx = i
			break
		}
	}

	a.scrollToFilteredNode(a.nodeSelIdx)
	a.statusMsg = ""
}

// enterJumpMode activates EasyMotion-style jump labels for visible selectable nodes.
func (a *App) enterJumpMode() {
	if len(a.codeNodes) == 0 {
		a.statusMsg = "No selectable nodes in this section"
		return
	}
	a.ensureRenderedLines()

	// Find nodes visible on screen
	visibleStart := a.contentOffset
	visibleEnd := a.contentOffset + a.contentHeight()

	var visibleNodes []int
	for i, info := range a.nodeRenderInfo {
		if info.firstLine >= visibleStart && info.firstLine < visibleEnd {
			visibleNodes = append(visibleNodes, i)
		}
	}

	if len(visibleNodes) == 0 {
		a.statusMsg = "No selectable nodes visible"
		return
	}

	// Generate single-char labels only for visible nodes (max 24)
	labels := generateLabels(len(visibleNodes))
	a.jumpLabels = make(map[string]int, len(labels))
	for i, lbl := range labels {
		a.jumpLabels[lbl] = visibleNodes[i]
	}
	a.jumpInput = ""
	a.mode = ModeJump
}

// generateLabels produces unique single-character labels (a-z, skipping f and q).
func generateLabels(n int) []string {
	chars := "abcdeghijklmnoprstuvwxyz" // 24 chars, skip f and q
	if n > len(chars) {
		n = len(chars)
	}
	labels := make([]string, n)
	for i := 0; i < n; i++ {
		labels[i] = string(chars[i])
	}
	return labels
}

// handleJumpKey processes input during jump mode.
func (a *App) handleJumpKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	switch k {
	case "esc":
		a.mode = ModeNormal
		a.jumpLabels = nil
		return a, nil
	}

	if len(msg.Text) == 0 {
		return a, nil
	}

	a.jumpInput += string(msg.Text)

	// Check for exact match
	if idx, ok := a.jumpLabels[a.jumpInput]; ok {
		// Jump to the node — enter node-select mode with "all" sub-mode
		a.mode = ModeNodeSelect
		a.nodeSubMode = nodeAll
		a.jumpLabels = nil
		a.jumpInput = ""

		// Find the position in filtered list
		indices := a.filteredNodeIndices()
		for i, nodeIdx := range indices {
			if nodeIdx == idx {
				a.nodeSelIdx = i
				break
			}
		}
		a.scrollToFilteredNode(a.nodeSelIdx)
		return a, nil
	}

	// Check if any label starts with current input (partial match)
	hasPrefix := false
	for lbl := range a.jumpLabels {
		if strings.HasPrefix(lbl, a.jumpInput) {
			hasPrefix = true
			break
		}
	}
	if !hasPrefix {
		// No possible match — cancel
		a.mode = ModeNormal
		a.jumpLabels = nil
		a.jumpInput = ""
	}

	return a, nil
}

// jumpLabelForNode returns the label for a given node index, or "" if not in jump mode.
func (a *App) jumpLabelForNode(nodeIdx int) string {
	if a.mode != ModeJump || a.jumpLabels == nil {
		return ""
	}
	for lbl, idx := range a.jumpLabels {
		if idx == nodeIdx {
			return lbl
		}
	}
	return ""
}

// ─────────────────────────────────────────────
// Node selection mode keys
// ─────────────────────────────────────────────

func (a *App) handleNodeSelectKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	km := a.cfg.Keys
	filtered := a.filteredNodes()

	// Handle pending key sequences (gg, zz, zt, zb)
	if a.pendingKey != "" {
		combo := a.pendingKey + k
		a.pendingKey = ""
		switch {
		case config.KeyMatches(combo, km.ViewCenter):
			a.nodeSelectViewCenter()
		case config.KeyMatches(combo, km.ViewTop):
			a.nodeSelectViewTop()
		case config.KeyMatches(combo, km.ViewBottom):
			a.nodeSelectViewBottom()
		}
		return a, nil
	}

	// Check for prefix keys
	if k == "z" {
		a.pendingKey = k
		return a, nil
	}

	switch {
	case config.KeyMatches(k, km.NodeExit):
		a.mode = ModeNormal
		a.copyMsg = ""
	case k == "m":
		// Cycle sub-mode
		a.nodeSubMode = (a.nodeSubMode + 1) % nodeKindCount
		a.nodeSelIdx = 0
		a.scrollToFilteredNode(0)
	case config.KeyMatches(k, km.NodeNext):
		if len(filtered) > 0 {
			a.nodeSelIdx = (a.nodeSelIdx + 1) % len(filtered)
			a.scrollToFilteredNode(a.nodeSelIdx)
		}
	case config.KeyMatches(k, km.NodePrev):
		if len(filtered) > 0 {
			a.nodeSelIdx = (a.nodeSelIdx - 1 + len(filtered)) % len(filtered)
			a.scrollToFilteredNode(a.nodeSelIdx)
		}
	case config.KeyMatches(k, km.JumpHigh):
		a.nodeSelectJumpHigh()
	case config.KeyMatches(k, km.JumpMid):
		a.nodeSelectJumpMid()
	case config.KeyMatches(k, km.JumpLow):
		a.nodeSelectJumpLow()
	case config.KeyMatches(k, km.NodeCopy):
		if len(filtered) > 0 {
			node := filtered[a.nodeSelIdx]
			if err := clipboard.WriteAll(node.content); err != nil {
				a.copyMsg = "Clipboard error: " + err.Error()
			} else {
				lang := node.lang
				if lang == "" {
					lang = "span"
				}
				if node.kind == nodeLink {
					a.copyMsg = fmt.Sprintf("Copied link: %s", node.content)
				} else {
					a.copyMsg = fmt.Sprintf("Copied %s block (%d lines)", lang, strings.Count(node.content, "\n")+1)
				}
			}
			a.statusMsg = a.copyMsg
			a.copyMsg = ""
			if a.nodeSubMode != nodeAll && a.nodeSubMode != nodeLink {
				a.mode = ModeNormal
			}
		}
	case config.KeyMatches(k, km.NodeOpen) && (a.nodeSubMode == nodeLink || (a.nodeSubMode == nodeAll && len(filtered) > 0 && filtered[a.nodeSelIdx].kind == nodeLink)):
		if len(filtered) > 0 {
			node := filtered[a.nodeSelIdx]
			url := node.content
			if url == "" {
				break
			}
			// Internal anchor link — jump to heading
			if strings.HasPrefix(url, "#") {
				anchor := url[1:]
				if idx := a.findHeadingByAnchor(anchor); idx >= 0 {
					a.mode = ModeNormal
					a.navigateTo(idx)
					a.statusMsg = fmt.Sprintf("Jumped to: %s", a.doc.Headings[idx].Text)
				} else {
					a.statusMsg = fmt.Sprintf("Heading not found: %s", anchor)
				}
			} else {
				// External link — open with configured opener
				opener := a.cfg.UI.Opener
				if opener == "" {
					opener = "xdg-open"
				}
				cmd := exec.Command(opener, url)
				cmd.Stdout = nil
				cmd.Stderr = nil
				_ = cmd.Start()
				a.statusMsg = fmt.Sprintf("Opened: %s", url)
			}
		}
	case k == "pgdown" || config.KeyMatches(k, km.ScrollHalfDown):
		a.nodeSelectScroll(a.contentHeight())
	case k == "pgup" || config.KeyMatches(k, km.ScrollHalfUp):
		a.nodeSelectScroll(-a.contentHeight())
	}
	return a, nil
}

// nodeSelectScroll scrolls the viewport in node-select mode and reselects
// the closest visible node in the new viewport.
func (a *App) nodeSelectScroll(delta int) {
	a.contentOffset += delta
	a.clampContentOffset()

	// Find the closest visible filtered node
	indices := a.filteredNodeIndices()
	if len(indices) == 0 {
		return
	}

	visibleStart := a.contentOffset
	visibleEnd := a.contentOffset + a.contentHeight()

	if delta > 0 {
		// Scrolled down — pick the topmost visible node
		for i, idx := range indices {
			if idx < len(a.nodeRenderInfo) && a.nodeRenderInfo[idx].firstLine >= visibleStart && a.nodeRenderInfo[idx].firstLine < visibleEnd {
				a.nodeSelIdx = i
				return
			}
		}
		// No visible node found — select the last one
		a.nodeSelIdx = len(indices) - 1
	} else {
		// Scrolled up — pick the bottommost visible node
		for i := len(indices) - 1; i >= 0; i-- {
			idx := indices[i]
			if idx < len(a.nodeRenderInfo) && a.nodeRenderInfo[idx].firstLine >= visibleStart && a.nodeRenderInfo[idx].firstLine < visibleEnd {
				a.nodeSelIdx = i
				return
			}
		}
		// No visible node — select the first one
		a.nodeSelIdx = 0
	}
}

// nodeSelectJumpHigh selects the topmost visible node in the viewport.
func (a *App) nodeSelectJumpHigh() {
	indices := a.filteredNodeIndices()
	if len(indices) == 0 {
		return
	}
	visibleStart := a.contentOffset
	visibleEnd := a.contentOffset + a.contentHeight()
	for i, idx := range indices {
		if idx < len(a.nodeRenderInfo) && a.nodeRenderInfo[idx].firstLine >= visibleStart && a.nodeRenderInfo[idx].firstLine < visibleEnd {
			a.nodeSelIdx = i
			return
		}
	}
}

// nodeSelectJumpMid selects the node closest to the middle of the viewport.
func (a *App) nodeSelectJumpMid() {
	indices := a.filteredNodeIndices()
	if len(indices) == 0 {
		return
	}
	midLine := a.contentOffset + a.contentHeight()/2
	bestDist := -1
	bestIdx := 0
	for i, idx := range indices {
		if idx < len(a.nodeRenderInfo) && a.nodeRenderInfo[idx].firstLine >= 0 {
			dist := a.nodeRenderInfo[idx].firstLine - midLine
			if dist < 0 {
				dist = -dist
			}
			if bestDist < 0 || dist < bestDist {
				bestDist = dist
				bestIdx = i
			}
		}
	}
	a.nodeSelIdx = bestIdx
	a.scrollToFilteredNode(a.nodeSelIdx)
}

// nodeSelectJumpLow selects the bottommost visible node in the viewport.
func (a *App) nodeSelectJumpLow() {
	indices := a.filteredNodeIndices()
	if len(indices) == 0 {
		return
	}
	visibleStart := a.contentOffset
	visibleEnd := a.contentOffset + a.contentHeight()
	for i := len(indices) - 1; i >= 0; i-- {
		idx := indices[i]
		if idx < len(a.nodeRenderInfo) && a.nodeRenderInfo[idx].firstLine >= visibleStart && a.nodeRenderInfo[idx].firstLine < visibleEnd {
			a.nodeSelIdx = i
			return
		}
	}
}

// nodeSelectViewCenter scrolls so the selected node is centered in the viewport.
func (a *App) nodeSelectViewCenter() {
	indices := a.filteredNodeIndices()
	if a.nodeSelIdx >= len(indices) {
		return
	}
	idx := indices[a.nodeSelIdx]
	if idx >= len(a.nodeRenderInfo) || a.nodeRenderInfo[idx].firstLine < 0 {
		return
	}
	a.contentOffset = a.nodeRenderInfo[idx].firstLine - a.contentHeight()/2
	a.clampContentOffset()
}

// nodeSelectViewTop scrolls so the selected node is at the top of the viewport.
func (a *App) nodeSelectViewTop() {
	indices := a.filteredNodeIndices()
	if a.nodeSelIdx >= len(indices) {
		return
	}
	idx := indices[a.nodeSelIdx]
	if idx >= len(a.nodeRenderInfo) || a.nodeRenderInfo[idx].firstLine < 0 {
		return
	}
	a.contentOffset = a.nodeRenderInfo[idx].firstLine
	a.clampContentOffset()
}

// nodeSelectViewBottom scrolls so the selected node is at the bottom of the viewport.
func (a *App) nodeSelectViewBottom() {
	indices := a.filteredNodeIndices()
	if a.nodeSelIdx >= len(indices) {
		return
	}
	idx := indices[a.nodeSelIdx]
	if idx >= len(a.nodeRenderInfo) || a.nodeRenderInfo[idx].firstLine < 0 {
		return
	}
	a.contentOffset = a.nodeRenderInfo[idx].firstLine - a.contentHeight() + 1
	a.clampContentOffset()
}

// ensureRenderedLines builds renderedLines and nodeRenderInfo if they are
// stale or missing. Called eagerly before any operation that needs them
// (e.g. scrollContentToNode) so that the data is ready before View().
func (a *App) ensureRenderedLines() {
	w := a.contentWidth() - 2
	if w < 1 {
		w = 1
	}
	if a.renderedLinesIdx == a.selectedIdx && a.renderedLinesWidth == w && a.renderedLines != nil {
		return // already up to date
	}
	markdown := strings.Join(a.sectionLines, "\n")
	a.renderedLines = a.renderGlamour(markdown, w)
	a.renderedLinesIdx = a.selectedIdx
	a.renderedLinesWidth = w
	a.nodeRenderInfo = mapNodesToRenderedLines(a.codeNodes, a.renderedLines)
	a.clampContentOffset()
}

// scrollContentToNode scrolls the content so the selected code node is visible.
func (a *App) scrollContentToNode(nodeIdx int) {
	if nodeIdx < 0 || nodeIdx >= len(a.codeNodes) {
		return
	}
	// Ensure rendered lines and node mapping are built before consulting them.
	a.ensureRenderedLines()
	// Use the rendered-line position if available, fall back to source line.
	target := a.codeNodes[nodeIdx].startLine
	if a.nodeRenderInfo != nil && nodeIdx < len(a.nodeRenderInfo) && a.nodeRenderInfo[nodeIdx].firstLine >= 0 {
		target = a.nodeRenderInfo[nodeIdx].firstLine
	}
	if target < a.contentOffset || target >= a.contentOffset+a.contentHeight() {
		a.contentOffset = target
		a.clampContentOffset()
	}
}

// scrollToFilteredNode scrolls to the node at filteredIdx within the current sub-mode.
func (a *App) scrollToFilteredNode(filteredIdx int) {
	indices := a.filteredNodeIndices()
	if filteredIdx >= 0 && filteredIdx < len(indices) {
		a.scrollContentToNode(indices[filteredIdx])
	}
}

// ─────────────────────────────────────────────
// Search mode
// ─────────────────────────────────────────────

func (a *App) handleSearchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		a.searchMatches = nil
		q := strings.ToLower(a.searchQuery)
		for i, h := range a.doc.Headings {
			if fuzzyMatch(strings.ToLower(h.Text), q) {
				a.searchMatches = append(a.searchMatches, i)
			}
		}
		a.searchIdx = 0
		if len(a.searchMatches) > 0 {
			a.navigateTo(a.searchMatches[0])
		}
		a.mode = ModeNormal

	case "esc":
		a.mode = ModeNormal
		a.searchQuery = ""

	case "backspace", "ctrl+h":
		if r := []rune(a.searchQuery); len(r) > 0 {
			a.searchQuery = string(r[:len(r)-1])
		}

	default:
		if len(msg.Text) > 0 {
			a.searchQuery += string(msg.Text)
		}
	}
	return a, nil
}

// fuzzyMatch checks if all characters in pattern appear in text in order.
func fuzzyMatch(text, pattern string) bool {
	ti := 0
	for _, pc := range pattern {
		found := false
		for ti < len([]rune(text)) {
			if []rune(text)[ti] == pc {
				ti++
				found = true
				break
			}
			ti++
		}
		if !found {
			return false
		}
	}
	return true
}

func (a *App) nextSearchMatch() {
	if len(a.searchMatches) == 0 {
		return
	}
	a.searchIdx = (a.searchIdx + 1) % len(a.searchMatches)
	a.selectedIdx = a.searchMatches[a.searchIdx]
	a.scrollOutlineToSelected()
	a.rebuildSection()
}

func (a *App) prevSearchMatch() {
	if len(a.searchMatches) == 0 {
		return
	}
	a.searchIdx = (a.searchIdx - 1 + len(a.searchMatches)) % len(a.searchMatches)
	a.selectedIdx = a.searchMatches[a.searchIdx]
	a.scrollOutlineToSelected()
	a.rebuildSection()
}

// ─────────────────────────────────────────────
// Content search
// ─────────────────────────────────────────────

func (a *App) handleContentSearchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		a.mode = ModeNormal
		if len(a.contentSearchMatches) > 0 {
			a.scrollToContentMatch(a.contentSearchIdx)
		}
	case "esc":
		a.mode = ModeNormal
		a.contentSearchQuery = ""
		a.contentSearchMatches = nil
	case "backspace", "ctrl+h":
		if r := []rune(a.contentSearchQuery); len(r) > 0 {
			a.contentSearchQuery = string(r[:len(r)-1])
		}
		a.updateContentSearchMatches()
	default:
		if len(msg.Text) > 0 {
			a.contentSearchQuery += string(msg.Text)
		}
		a.updateContentSearchMatches()
	}
	return a, nil
}

func (a *App) updateContentSearchMatches() {
	a.contentSearchMatches = nil
	a.contentSearchIdx = 0
	q := strings.ToLower(a.contentSearchQuery)
	if q == "" {
		return
	}
	stripped := make([]string, len(a.renderedLines))
	for i, l := range a.renderedLines {
		stripped[i] = ansi.Strip(l)
	}
	for i, s := range stripped {
		lower := strings.ToLower(s)
		offset := 0
		for {
			idx := strings.Index(lower[offset:], q)
			if idx == -1 {
				break
			}
			byteStart := offset + idx
			byteEnd := byteStart + len(q)
			colStart := byteColToDisplayCol(s, byteStart)
			colEnd := byteColToDisplayCol(s, byteEnd)
			a.contentSearchMatches = append(a.contentSearchMatches, contentMatch{
				line:      i,
				colStart:  colStart,
				colEnd:    colEnd,
				byteStart: byteStart,
				byteEnd:   byteEnd,
			})
			offset = byteStart + len(q)
		}
	}
	// Scroll to first match if any
	if len(a.contentSearchMatches) > 0 {
		a.scrollToContentMatch(0)
	}
}

func (a *App) scrollToContentMatch(idx int) {
	if idx < 0 || idx >= len(a.contentSearchMatches) {
		return
	}
	m := a.contentSearchMatches[idx]
	h := a.contentHeight()
	// Ensure the match line is visible
	if m.line < a.contentOffset {
		a.contentOffset = m.line
	} else if m.line >= a.contentOffset+h {
		a.contentOffset = m.line - h/2
	}
	a.clampContentOffset()
}

func (a *App) nextContentSearchMatch() {
	if len(a.contentSearchMatches) == 0 {
		return
	}
	a.contentSearchIdx = (a.contentSearchIdx + 1) % len(a.contentSearchMatches)
	a.scrollToContentMatch(a.contentSearchIdx)
}

func (a *App) prevContentSearchMatch() {
	if len(a.contentSearchMatches) == 0 {
		return
	}
	a.contentSearchIdx = (a.contentSearchIdx - 1 + len(a.contentSearchMatches)) % len(a.contentSearchMatches)
	a.scrollToContentMatch(a.contentSearchIdx)
}

// ─────────────────────────────────────────────
// Theme picker
// ─────────────────────────────────────────────

func (a *App) handleThemeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	themeNames := []string{"OceanDark", "Nord", "Dracula", "Gruvbox", "TokyoNight"}
	switch msg.String() {
	case "esc", "q", "T":
		a.mode = ModeNormal
	default:
		for i, name := range themeNames {
			if msg.String() == fmt.Sprintf("%d", i+1) {
				a.cfg.UI.Theme = name
				a.theme = ResolveTheme(a.cfg)
				a.cfg.Save()
				// Invalidate glamour renderer and rendered lines cache
				a.glamourRenderer = nil
				a.renderedLines = nil
				a.nodeRenderInfo = nil
				a.mode = ModeNormal
				return a, nil
			}
		}
	}
	return a, nil
}

// ─────────────────────────────────────────────
// Layout helpers
// ─────────────────────────────────────────────

func (a *App) sidebarWidth() int {
	if a.sidebarHidden {
		return 0
	}

	// Compute based on longest heading (with indent)
	maxLen := len("(Document)")
	for _, h := range a.doc.Headings {
		// indent(2*level) + marker(level) + space(1) + text
		entryLen := 2*h.Level + h.Level + 1 + len([]rune(h.Text))
		if entryLen > maxLen {
			maxLen = entryLen
		}
	}

	// Add 2 for borders, 1 for padding
	ideal := maxLen + 3

	// Clamp: minimum 20, maximum 40% of width
	minW := 20
	maxW := a.width * 2 / 5
	if ideal < minW {
		ideal = minW
	}
	if ideal > maxW {
		ideal = maxW
	}
	if ideal > a.width/2 {
		ideal = a.width / 2
	}
	return ideal
}

func (a *App) contentWidth() int {
	sw := a.sidebarWidth()
	if sw == 0 {
		return a.width
	}
	return a.width - sw
}

// paneInnerHeight returns the number of content lines that fit inside a bordered pane.
// Layout: title(1) + border-top(1) + inner(N) + border-bottom(1) + status(1) = height
// So inner = height - 4.
func (a *App) paneInnerHeight() int {
	h := a.height - 4
	if h < 0 {
		return 0
	}
	return h
}

// outlineHeight is an alias kept for scrollOutlineToSelected.
func (a *App) outlineHeight() int { return a.paneInnerHeight() }
func (a *App) contentHeight() int { return a.paneInnerHeight() }

// ─────────────────────────────────────────────
// View
// ─────────────────────────────────────────────

func (a *App) View() tea.View {
	if a.width == 0 {
		v := tea.NewView("Loading...")
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	switch a.mode {
	case ModeThemePicker:
		// fall through — rendered as overlay below
	}

	title := a.renderTitle()
	status := a.renderStatus()

	var body string
	if a.sidebarHidden {
		// Single bordered content pane, full width
		body = a.renderBorderedContent(a.width, a.focus == FocusContent)
	} else {
		sw := a.sidebarWidth()
		cw := a.contentWidth()
		outline := a.renderBorderedOutline(sw, a.focus == FocusSidebar)
		content := a.renderBorderedContent(cw, a.focus == FocusContent)
		body = lipgloss.JoinHorizontal(lipgloss.Top, outline, content)
	}

	view := lipgloss.JoinVertical(lipgloss.Left, title, body, status)

	if a.mode == ModeHelp {
		view = a.overlayHelp(view)
	}
	if a.mode == ModeThemePicker {
		view = a.overlayThemePicker(view)
	}

	v := tea.NewView(view)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// ─────────────────────────────────────────────
// Individual pane renderers
// ─────────────────────────────────────────────

func (a *App) renderTitle() string {
	name := filepath.Base(a.filename)
	if name == "" || name == "." {
		name = "gomd"
	}

	title := "gomd — " + name
	if bc := a.breadcrumb(); bc != "" {
		title += " > " + bc
	}

	return lipgloss.NewStyle().
		Background(a.theme.Border).
		Foreground(a.theme.Foreground).
		Bold(true).
		Width(a.width).
		Padding(0, 1).
		Render(title)
}

// breadcrumb returns the heading path from root to the current selection.
func (a *App) breadcrumb() string {
	if a.selectedIdx < 0 {
		return ""
	}
	h := a.doc.Headings[a.selectedIdx]
	var parts []string
	parts = append(parts, h.Text)

	// Walk backwards to find ancestor headings (lower level)
	for i := a.selectedIdx - 1; i >= 0; i-- {
		if a.doc.Headings[i].Level < h.Level {
			parts = append([]string{a.doc.Headings[i].Text}, parts...)
			h = a.doc.Headings[i]
		}
	}
	return strings.Join(parts, " > ")
}

// drawBox draws a bordered box around content lines.
// outerW is the full column width including border characters (2 cols).
// lines must have exactly paneInnerHeight() entries.
// focused controls border color.
func (a *App) drawBox(lines []string, outerW int, focused bool, title string) string {
	innerW := outerW - 2
	if innerW < 1 {
		innerW = 1
	}

	borderColor := a.theme.Border
	if focused {
		borderColor = a.theme.Highlight
	}
	bc := lipgloss.NewStyle().Foreground(borderColor).Background(a.theme.Background)

	// Top border: ┌─ title ──────┐
	titleRunes := []rune(title)
	topInner := innerW
	if len(titleRunes)+4 > topInner {
		topInner = len(titleRunes) + 4
	}
	var topMid string
	if title != "" {
		dashes := topInner - len(titleRunes) - 2 // 2 for spaces
		if dashes < 0 {
			dashes = 0
		}
		topMid = " " + title + " " + strings.Repeat("─", dashes)
	} else {
		topMid = strings.Repeat("─", topInner)
	}
	// Trim/pad topMid to innerW
	topMidRunes := []rune(topMid)
	if len(topMidRunes) > innerW {
		topMidRunes = topMidRunes[:innerW]
	}
	for len(topMidRunes) < innerW {
		topMidRunes = append(topMidRunes, '─')
	}
	topMid = string(topMidRunes)

	top := bc.Render("┌" + topMid + "┐")
	bottom := bc.Render("└" + strings.Repeat("─", innerW) + "┘")

	var sb strings.Builder
	sb.WriteString(top)
	sb.WriteByte('\n')
	borderL := bc.Render("│")
	borderR := bc.Render("│")
	bgStyle := lipgloss.NewStyle().
		Background(a.theme.Background).
		Foreground(a.theme.Foreground)
	// Build the ANSI opening sequence for our theme background so we can
	// inject it after any resets in glamour output.
	bgOpen := strings.TrimSuffix(bgStyle.Render(" "), " \x1b[0m")

	for _, line := range lines {
		// Normalize line to exactly innerW display columns so the right
		// border always appears in the correct column regardless of whether
		// the line contains ANSI sequences (e.g. from glamour).
		line = ansi.Truncate(line, innerW, "")
		visW := lipgloss.Width(line)

		// Replace ANSI resets with resets that re-apply our background,
		// so glamour's internal resets don't clear the theme background.
		// Glamour v2 uses \x1b[m (short reset); normalise to \x1b[0m first.
		line = strings.ReplaceAll(line, "\x1b[m", "\x1b[0m")
		line = strings.ReplaceAll(line, "\x1b[0m", "\x1b[0m"+bgOpen)

		// Paint the entire line with the theme background.
		line = bgOpen + line + "\x1b[0m"
		if visW < innerW {
			line += bgStyle.Render(strings.Repeat(" ", innerW-visW))
		}
		sb.WriteString(borderL + line + borderR)
		sb.WriteByte('\n')
	}
	sb.WriteString(bottom)
	return sb.String()
}

// renderBorderedOutline renders the outline pane with a manually drawn border.
func (a *App) renderBorderedOutline(outerW int, focused bool) string {
	innerW := outerW - 2
	if innerW < 1 {
		innerW = 1
	}
	h := a.paneInnerHeight()

	// Total entries: root + headings. outlineOffset is a display index.
	total := a.totalEntries()
	endDisplay := a.outlineOffset + h
	if endDisplay > total {
		endDisplay = total
	}

	lines := make([]string, 0, h)
	for di := a.outlineOffset; di < endDisplay; di++ {
		// di==0 is root; di>0 maps to heading di-1
		var text string
		var isSelected bool
		var headingLevel int

		if di == 0 {
			text = "(Document)"
			isSelected = a.selectedIdx == -1
			headingLevel = 0
		} else {
			hd := a.doc.Headings[di-1]
			indent := strings.Repeat("  ", hd.Level)
			marker := strings.Repeat("#", hd.Level)
			text = indent + marker + " " + hd.Text
			isSelected = a.selectedIdx == di-1
			headingLevel = hd.Level
		}

		maxRunes := innerW
		if maxRunes < 1 {
			maxRunes = 1
		}
		if len([]rune(text)) > maxRunes {
			text = string([]rune(text)[:maxRunes-1]) + "…"
		}

		if isSelected {
			lines = append(lines, lipgloss.NewStyle().
				Background(a.theme.Selected).
				Foreground(a.theme.Foreground).
				Bold(true).
				Width(innerW).
				Render(text))
		} else {
			var fg lipgloss.Color
			switch headingLevel {
			case 0:
				fg = a.theme.Foreground
			case 1:
				fg = a.theme.Heading1
			case 2:
				fg = a.theme.Heading2
			case 3:
				fg = a.theme.Heading3
			default:
				fg = a.theme.HeadingN
			}
			lines = append(lines, lipgloss.NewStyle().
				Background(a.theme.Background).
				Foreground(fg).
				Width(innerW).
				Render(text))
		}
	}
	for len(lines) < h {
		lines = append(lines, lipgloss.NewStyle().Background(a.theme.Background).Width(innerW).Render(""))
	}

	return a.drawBox(lines, outerW, focused, "Outline")
}

// renderBorderedContent renders the section content pane with a manually drawn border.
func (a *App) renderBorderedContent(outerW int, focused bool) string {
	innerW := outerW - 2
	if innerW < 1 {
		innerW = 1
	}
	inner := a.renderContentWidth(innerW)
	lines := strings.Split(inner, "\n")
	h := a.paneInnerHeight()
	// Ensure exactly h lines
	for len(lines) < h {
		lines = append(lines, "")
	}
	lines = lines[:h]

	return a.drawBox(lines, outerW, focused, "")
}

func (a *App) renderContent() string {
	return a.renderContentWidth(a.contentWidth() - 2)
}

func (a *App) renderContentWidth(w int) string {
	h := a.contentHeight()
	if w <= 0 || h <= 0 {
		return strings.Repeat("\n", h-1)
	}

	return a.renderContentGlamour(w, h)
}

// renderContentGlamour renders the current section using glamour and returns a
// newline-joined string of exactly h lines, each at most w columns wide.
// In ModeNodeSelect it overlays background highlights on selectable nodes.
func (a *App) renderContentGlamour(w, h int) string {
	a.ensureRenderedLines()

	lines := a.renderedLines

	// In node-select mode, highlight only the filtered nodes.
	if a.mode == ModeNodeSelect {
		indices := a.filteredNodeIndices()
		if len(indices) > 0 {
			filtered := make([]codeNode, len(indices))
			filteredInfo := make([]nodeRenderLoc, len(indices))
			for i, idx := range indices {
				filtered[i] = a.codeNodes[idx]
				if idx < len(a.nodeRenderInfo) {
					filteredInfo[i] = a.nodeRenderInfo[idx]
				}
			}
			lines = applyNodeHighlights(lines, filtered, a.nodeSelIdx, filteredInfo, a.theme, w)
		}
	}

	// In jump mode, dim content and overlay labels.
	if a.mode == ModeJump && a.jumpLabels != nil {
		lines = a.applyJumpLabels(lines, w)
	}

	// Apply content search highlights
	if len(a.contentSearchMatches) > 0 && a.contentSearchQuery != "" {
		lines = a.applyContentSearchHighlights(lines)
	}

	start := a.contentOffset
	if start > len(lines) {
		start = len(lines)
	}
	end := start + h
	if end > len(lines) {
		end = len(lines)
	}
	visible := lines[start:end]

	// Pad to exactly h lines
	out := make([]string, h)
	copy(out, visible)
	return strings.Join(out, "\n")
}

// mapNodesToRenderedLines maps each codeNode to its location in the glamour-rendered
// output. Returns a slice of nodeRenderLoc parallel to nodes.
//
// For fenced blocks: finds the first rendered line whose stripped text contains
// the first non-empty line of the block body, then extends to cover all body lines.
// For inline nodes: finds the rendered line containing the span text, and records
// the display-column range of that span within the stripped line.
// byteColToDisplayCol converts a byte offset within a stripped (ANSI-free) string
// to a display column count. Needed because strings.Index returns byte offsets
// but ansi.Truncate / lipgloss.Width operate in display columns (Unicode-aware).
func byteColToDisplayCol(s string, byteOff int) int {
	if byteOff <= 0 {
		return 0
	}
	if byteOff >= len(s) {
		return lipgloss.Width(s)
	}
	return lipgloss.Width(s[:byteOff])
}

func mapNodesToRenderedLines(nodes []codeNode, rendered []string) []nodeRenderLoc {
	result := make([]nodeRenderLoc, len(nodes))
	// Pre-strip ANSI from all rendered lines once.
	stripped := make([]string, len(rendered))
	for i, l := range rendered {
		stripped[i] = ansi.Strip(l)
	}

	// Nodes are ordered by source line. Their rendered positions are also
	// monotonically non-decreasing, so we scan forward from the previous
	// node's rendered position to avoid matching an earlier duplicate.
	//
	// Two separate forward cursors are maintained:
	//
	//   inlineSearchFromLine / inlineSearchFromCol
	//     Used by inline nodes (headings, backtick spans).
	//     After a fenced block this advances to lastLine+1 so that inline
	//     nodes cannot accidentally match text inside a rendered code block.
	//
	//   blockSearchFromLine
	//     Used by fenced block nodes only.
	//     Advances to firstLine (not lastLine) of the previous block so that
	//     consecutive source blocks whose rendered output overlaps (glamour
	//     can merge adjacent blocks) can each be located.
	inlineSearchFromLine := 0
	inlineSearchFromCol := -1
	blockSearchFromLine := 0

	for ni, n := range nodes {
		loc := nodeRenderLoc{firstLine: -1, lastLine: -1, spanCol: -1, spanColEnd: -1}

		if n.inline {
			if n.lang == "heading" {
				// Glamour renders headings as "  ## Text  ..."
				// Glamour renders inline code within headings with double-spaces, and
				// strips markdown link syntax ([text](url) → text).
				// Strategy: use the longest plain-text prefix before any backtick span
				// or special char as the search key, since that prefix appears verbatim.
				// When a heading starts with or contains backtick spans, glamour renders
				// them with double-spaces (e.g. `foo` → "  foo  "), so we search using
				// the padded form for backtick-starting headings.
				headingSearch := n.content
				startsWithBacktick := strings.HasPrefix(headingSearch, "`")
				if !startsWithBacktick {
					// Use plain-text prefix before the first backtick
					if idx := strings.Index(headingSearch, "`"); idx > 0 {
						headingSearch = strings.TrimRight(headingSearch[:idx], " ")
					} else {
						// No backtick: strip markdown links only
						headingSearch = mdLinkRe.ReplaceAllString(headingSearch, "$1")
					}
				}
				if startsWithBacktick || headingSearch == "" {
					// Extract inner text of first backtick span and use padded form
					if m := backtickRe.FindStringSubmatch(n.content); m != nil {
						headingSearch = "  " + m[1] + "  "
					} else {
						headingSearch = backtickRe.ReplaceAllString(n.content, "$1")
						headingSearch = mdLinkRe.ReplaceAllString(headingSearch, "$1")
					}
				}
				// Find a line containing the heading search key. Glamour renders h2+
				// with "## Text" but h1 headings are rendered without "#" (just the text).
				// Strategy: prefer lines with "#", fall back to lines where the heading
				// text appears at the start (after trimming leading spaces).
				for ri := inlineSearchFromLine; ri < len(stripped); ri++ {
					s := stripped[ri]
					hasHash := strings.Contains(s, "#")
					if !hasHash {
						// For h1: check if trimmed line equals the heading text
						trimmed := strings.TrimSpace(s)
						if trimmed != headingSearch {
							continue
						}
					}
					col := strings.Index(s, headingSearch)
					if col == -1 {
						continue
					}
					// On inlineSearchFromLine, skip columns already consumed (byte offsets).
					if ri == inlineSearchFromLine && col <= inlineSearchFromCol {
						continue
					}
					loc.firstLine = ri
					loc.lastLine = ri
					loc.spanColByte = col
					loc.spanColEndByte = col + len(headingSearch)
					loc.spanCol = byteColToDisplayCol(s, col)
					loc.spanColEnd = loc.spanCol + lipgloss.Width(headingSearch)
					break
				}
			} else if n.kind == nodeLink {
				// Link nodes: use n.display (the visible text glamour renders)
				// as the search needle. Links don't have the double-space padding
				// that inline code spans get.
				needle := n.display
				if needle == "" {
					needle = n.content // fallback
				}
				for ri := inlineSearchFromLine; ri < len(stripped); ri++ {
					s := stripped[ri]
					colOffset := 0
					if ri == inlineSearchFromLine && inlineSearchFromCol >= 0 {
						colOffset = inlineSearchFromCol + 1
						if colOffset > len(s) {
							continue
						}
						s = s[colOffset:]
					}
					idx := strings.Index(s, needle)
					if idx >= 0 {
						bestCol := colOffset + idx
						bestColEnd := bestCol + len(needle)
						loc.firstLine = ri
						loc.lastLine = ri
						loc.spanColByte = bestCol
						loc.spanColEndByte = bestColEnd
						fullLine := stripped[ri]
						loc.spanCol = byteColToDisplayCol(fullLine, bestCol)
						loc.spanColEnd = loc.spanCol + lipgloss.Width(fullLine[bestCol:bestColEnd])
						break
					}
				}
			} else {
				// when mid-sentence, e.g. `useCallback` → "  useCallback  ".
				// When at end-of-sentence before punctuation only one trailing
				// space may appear, e.g. `Object.is`. → "  Object.is .".
				// Strategy: in a single forward pass from searchFromLine, on each
				// rendered line find BOTH the padded form ("  content  ") and the
				// bare form, then take the EARLIEST (minimum column) match.  This
				// prevents a later padded occurrence from pre-empting an earlier bare
				// occurrence on the same line.
				//
				// On inlineSearchFromLine, respect inlineSearchFromCol to handle multiple spans
				// on the same rendered line (same source line).
				padded := "  " + n.content + "  "
				paddedNBSP := "\u00a0" + n.content + "\u00a0"
				needle := n.content
				for ri := inlineSearchFromLine; ri < len(stripped); ri++ {
					s := stripped[ri]
					colOffset := 0
					if ri == inlineSearchFromLine && inlineSearchFromCol >= 0 {
						colOffset = inlineSearchFromCol + 1
						if colOffset > len(s) {
							continue
						}
						s = s[colOffset:]
					}
					bestCol := -1    // byte offset
					bestColEnd := -1 // byte offset

					// Check padded form (double-space in glamour v1, NBSP in v2)
					if paddedIdx := strings.Index(s, padded); paddedIdx != -1 {
						col := colOffset + paddedIdx + 2
						bestCol = col
						bestColEnd = col + len(n.content)
					} else if nbspIdx := strings.Index(s, paddedNBSP); nbspIdx != -1 {
						col := colOffset + nbspIdx + len("\u00a0")
						bestCol = col
						bestColEnd = col + len(n.content)
					}
					// Check alternative forms for inline code spans that may not have
					// full double-space padding (e.g., adjacent spans: `j`/`k` renders
					// as "j / k" with single spaces). Require at least 1 space before
					// AND 1 space after (or end of trimmed line) to confirm it's a span.
					if bestCol < 0 && !strings.Contains(stripped[ri], "##") {
						searchFrom := 0
						if ri == inlineSearchFromLine && colOffset > 0 {
							searchFrom = 0 // already sliced via colOffset
						}
						for pos := searchFrom; pos < len(s); pos++ {
							idx := strings.Index(s[pos:], needle)
							if idx == -1 {
								break
							}
							absIdx := colOffset + pos + idx
							endIdx := absIdx + len(needle)
							fullLine := stripped[ri]
							// For longer needles (>=3 chars), require 2 spaces before to
							// distinguish code spans from prose. For short needles (1-2 chars),
							// 1 space suffices since they can't be confused with prose words.
							var hasPre bool
							if len(needle) >= 3 {
								hasPre = absIdx > 1 && fullLine[absIdx-1] == ' ' && fullLine[absIdx-2] == ' '
							} else {
								hasPre = absIdx > 0 && fullLine[absIdx-1] == ' '
							}
							// Must have space after (or be at end of trimmed content)
							hasPost := endIdx >= len(strings.TrimRight(fullLine, " ")) || (endIdx < len(fullLine) && fullLine[endIdx] == ' ')
							if hasPre && hasPost {
								bestCol = absIdx
								bestColEnd = endIdx
								break
							}
							pos += idx + len(needle)
						}
					}
					if bestCol >= 0 {
						loc.firstLine = ri
						loc.lastLine = ri
						loc.spanColByte = bestCol
						loc.spanColEndByte = bestColEnd
						// Convert to display columns for highlightSpanInLine
						fullLine := stripped[ri]
						loc.spanCol = byteColToDisplayCol(fullLine, bestCol)
						loc.spanColEnd = loc.spanCol + lipgloss.Width(fullLine[bestCol:bestColEnd])
						break
					}
				}
			}
		} else {
			// Fenced block: find the first non-empty body line as a search anchor.
			needle := ""
			for _, bodyLine := range strings.Split(n.content, "\n") {
				t := strings.TrimSpace(bodyLine)
				if t != "" {
					needle = t
					break
				}
			}
			firstLine := -1
			if needle != "" {
				for ri := blockSearchFromLine; ri < len(stripped); ri++ {
					if strings.Contains(stripped[ri], needle) {
						firstLine = ri
						break
					}
				}
			}
			if firstLine == -1 {
				// Try fence marker as fallback
				for ri := blockSearchFromLine; ri < len(stripped); ri++ {
					if strings.Contains(stripped[ri], "```") || strings.Contains(stripped[ri], "~~~") {
						firstLine = ri
						break
					}
				}
			}
			if firstLine >= 0 {
				// Glamour renders fenced code blocks with 4-space indentation (vs
				// 2-space for prose). Scan forward from firstLine while lines are
				// either blank or start with at least 4 spaces to find the last
				// line of the block.
				lastLine := firstLine
				for ri := firstLine + 1; ri < len(stripped); ri++ {
					l := stripped[ri]
					trimmed := strings.TrimSpace(l)
					if trimmed == "" || strings.HasPrefix(l, "    ") {
						lastLine = ri
						continue
					}
					break
				}
				// Trim trailing blank lines.
				for lastLine > firstLine && strings.TrimSpace(stripped[lastLine]) == "" {
					lastLine--
				}
				loc.firstLine = firstLine
				loc.lastLine = lastLine
			}
		}

		result[ni] = loc
		// Advance search cursors.
		//
		// inlineSearchFromLine / inlineSearchFromCol: used by inline nodes.
		//   After a fenced block advances to lastLine+1 so inline nodes cannot
		//   match text that appears inside the rendered code block.
		//   After an inline node stays on loc.firstLine with col bumped past
		//   the span, allowing subsequent nodes on the same rendered line.
		//
		// blockSearchFromLine: used by fenced block nodes.
		//   Advances to firstLine only (not lastLine) so that consecutive
		//   source blocks whose rendered output overlaps can each be located.
		if loc.firstLine >= 0 {
			if !n.inline && loc.lastLine > loc.firstLine {
				// Fenced block: inline cursor jumps past the block.
				newInline := loc.lastLine + 1
				if newInline > inlineSearchFromLine {
					inlineSearchFromLine = newInline
					inlineSearchFromCol = -1
				}
				// Block cursor: only advance to firstLine.
				if loc.firstLine > blockSearchFromLine {
					blockSearchFromLine = loc.firstLine
				}
			} else {
				// Inline node: bump both inline and block cursors.
				if loc.firstLine > inlineSearchFromLine {
					inlineSearchFromLine = loc.firstLine
					inlineSearchFromCol = -1
				}
				if loc.spanColEndByte > 0 {
					inlineSearchFromCol = loc.spanColEndByte - 1
				}
				if loc.firstLine > blockSearchFromLine {
					blockSearchFromLine = loc.firstLine
				}
			}
		}
	}
	return result
}

// highlightSpanInLine applies a highlight style to the substring of a glamour-rendered
// line at display columns [colStart, colEnd). The rest of the line retains its original
// ANSI styling. Returns the modified line.
func highlightSpanInLine(line string, colStart, colEnd int, style lipgloss.Style) string {
	if colStart < 0 || colEnd <= colStart {
		return line
	}
	// Split the rendered string at display column boundaries using ansi.Truncate.
	// prefix = first colStart columns
	// middle = next (colEnd-colStart) columns
	// suffix = everything after colEnd
	prefix := ansi.Truncate(line, colStart, "")
	withMid := ansi.Truncate(line, colEnd, "")
	// middle is withMid minus prefix — we can get it by stripping the prefix bytes
	// from withMid. Since ansi.Truncate preserves ANSI codes, we use TruncateLeft.
	middle := ansi.TruncateLeft(withMid, colStart, "")
	suffix := ansi.TruncateLeft(line, colEnd, "")
	// Re-render the middle span with the highlight style.
	plainMiddle := ansi.Strip(middle)
	return prefix + style.Render(plainMiddle) + suffix
}

// applyContentSearchHighlights highlights all content search matches in the rendered lines.
func (a *App) applyContentSearchHighlights(lines []string) []string {
	if len(a.contentSearchMatches) == 0 {
		return lines
	}
	out := make([]string, len(lines))
	copy(out, lines)

	matchStyle := lipgloss.NewStyle().Background(lipgloss.Color("#5f5f00")).Foreground(lipgloss.Color("#ffffff"))
	currentStyle := lipgloss.NewStyle().Background(lipgloss.Color("#af8700")).Foreground(lipgloss.Color("#000000")).Bold(true)

	// Group matches by line for efficient processing
	// Apply from right to left to preserve column positions
	lineMatches := make(map[int][]int) // line -> list of match indices
	for i, m := range a.contentSearchMatches {
		lineMatches[m.line] = append(lineMatches[m.line], i)
	}

	for lineIdx, matchIdxs := range lineMatches {
		if lineIdx >= len(out) {
			continue
		}
		// Apply in reverse order (right to left) to preserve positions
		for i := len(matchIdxs) - 1; i >= 0; i-- {
			mi := matchIdxs[i]
			m := a.contentSearchMatches[mi]
			style := matchStyle
			if mi == a.contentSearchIdx {
				style = currentStyle
			}
			out[lineIdx] = highlightSpanInLine(out[lineIdx], m.colStart, m.colEnd, style)
		}
	}
	return out
}

// applyNodeHighlights returns a copy of rendered lines with highlights applied
// for node-select mode:
//   - All selectable inline/block nodes get a dim "available" background tint.
//   - The currently selected node gets a bright "selected" background.
//
// applyJumpLabels overlays labels next to selectable nodes with subtle highlights.
// Content remains unchanged; nodes get a background tint and labels appear beside them.
func (a *App) applyJumpLabels(lines []string, w int) []string {
	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ffffff")).
		Background(lipgloss.Color("#ff5f00"))

	// Copy lines
	out := make([]string, len(lines))
	copy(out, lines)

	nodeBg := lipgloss.NewStyle().Background(a.theme.Selected)

	// Group inline labels by line so we can apply right-to-left (preserving positions).
	type inlineLabel struct {
		lbl     string
		col     int // spanCol
		colEnd  int // spanColEnd
		nodeIdx int
	}
	lineLabels := make(map[int][]inlineLabel)

	for lbl, nodeIdx := range a.jumpLabels {
		if nodeIdx >= len(a.nodeRenderInfo) {
			continue
		}
		info := a.nodeRenderInfo[nodeIdx]
		if info.firstLine < 0 || info.firstLine >= len(out) {
			continue
		}

		node := a.codeNodes[nodeIdx]

		if node.inline && info.spanCol >= 0 {
			lineLabels[info.firstLine] = append(lineLabels[info.firstLine], inlineLabel{
				lbl: lbl, col: info.spanCol, colEnd: info.spanColEnd, nodeIdx: nodeIdx,
			})
		} else {
			// Block node: highlight all lines of the block and prepend label on first
			labelRendered := labelStyle.Render(lbl)
			firstLine := info.firstLine
			lastLine := info.lastLine
			if lastLine >= len(out) {
				lastLine = len(out) - 1
			}
			for li := firstLine; li <= lastLine; li++ {
				stripped := ansi.Strip(out[li])
				out[li] = nodeBg.Render(stripped)
			}
			out[firstLine] = labelRendered + " " + out[firstLine]
		}
	}

	// Apply inline labels per line, sorted right-to-left to preserve column positions.
	for lineIdx, labels := range lineLabels {
		// Sort by column descending
		sort.Slice(labels, func(i, j int) bool {
			return labels[i].col > labels[j].col
		})
		stripped := ansi.Strip(lines[lineIdx])
		runes := []rune(stripped)
		// Apply each label from right to left
		for _, il := range labels {
			col := il.col
			if col > len(runes) {
				col = len(runes)
			}
			spanEnd := il.colEnd
			if spanEnd > len(runes) {
				spanEnd = len(runes)
			}
			labelRendered := labelStyle.Render(il.lbl)
			span := string(runes[col:spanEnd])
			// Replace runes[col:spanEnd] with label + highlighted span
			replacement := []rune(labelRendered + nodeBg.Render(span))
			newRunes := make([]rune, 0, len(runes)-spanEnd+col+len(replacement))
			newRunes = append(newRunes, runes[:col]...)
			newRunes = append(newRunes, replacement...)
			newRunes = append(newRunes, runes[spanEnd:]...)
			runes = newRunes
		}
		out[lineIdx] = string(runes)
	}

	return out
}

// For inline nodes the highlight is applied only to the span, preserving surrounding
// glamour styling. For fenced blocks the whole-line background is applied (appropriate
// since the block is the selectable unit).
func applyNodeHighlights(lines []string, nodes []codeNode, selIdx int, info []nodeRenderLoc, theme Theme, w int) []string {
	out := make([]string, len(lines))
	copy(out, lines)

	dimStyle := lipgloss.NewStyle().Background(theme.Code)
	selStyle := lipgloss.NewStyle().Background(theme.NodeSel).Foreground(theme.Background).Bold(true)

	// Apply available highlights first (lower priority), then selected (higher).
	for pass := 0; pass < 2; pass++ {
		for ni, n := range nodes {
			isSelected := ni == selIdx
			if pass == 0 && isSelected {
				continue
			}
			if pass == 1 && !isSelected {
				continue
			}
			loc := info[ni]
			if loc.firstLine < 0 {
				continue
			}
			style := dimStyle
			if isSelected {
				style = selStyle
			}

			if n.inline {
				// Surgical span highlight — only colour the backtick content.
				li := loc.firstLine
				if li < len(out) {
					out[li] = highlightSpanInLine(out[li], loc.spanCol, loc.spanColEnd, style)
				}
			} else {
				// Full-line background for all lines of the block.
				for li := loc.firstLine; li <= loc.lastLine && li < len(out); li++ {
					plain := ansi.Strip(out[li])
					visW := lipgloss.Width(plain)
					if visW < w {
						plain += strings.Repeat(" ", w-visW)
					}
					out[li] = style.Render(plain)
				}
			}
		}
	}
	return out
}

func (a *App) renderStatus() string {
	// Build plain (unstyled) left/right text first so we can measure widths correctly.
	var leftPlain string
	switch a.mode {
	case ModeSearch:
		leftPlain = "/ " + a.searchQuery + "█"
	case ModeContentSearch:
		matchInfo := ""
		if len(a.contentSearchMatches) > 0 {
			matchInfo = fmt.Sprintf(" [%d/%d]", a.contentSearchIdx+1, len(a.contentSearchMatches))
		} else if a.contentSearchQuery != "" {
			matchInfo = " [0/0]"
		}
		leftPlain = "/ " + a.contentSearchQuery + "█" + matchInfo
	case ModeNodeSelect:
		if a.copyMsg != "" {
			leftPlain = "✓ " + a.copyMsg
		} else {
			filtered := a.filteredNodes()
			modeName := "CODE"
			switch a.nodeSubMode {
			case nodeInlineCode:
				modeName = "INLINE"
			case nodeLink:
				modeName = "LINKS"
			case nodeAll:
				modeName = "ALL"
			}
			if len(filtered) > 0 {
				leftPlain = fmt.Sprintf("%s [%d/%d]  m:mode  y:copy  j/k:nav  Esc:exit",
					modeName, a.nodeSelIdx+1, len(filtered))
			} else {
				leftPlain = fmt.Sprintf("%s [0/0]  m:mode  Esc:exit", modeName)
			}
		}
	case ModeJump:
		leftPlain = "JUMP: type label to jump (Esc to cancel)"
		if a.jumpInput != "" {
			leftPlain = fmt.Sprintf("JUMP: %s…", a.jumpInput)
		}
	default:
		if a.selectedIdx < 0 {
			leftPlain = fmt.Sprintf("[0/%d] (Document)", len(a.doc.Headings))
		} else if a.selectedIdx < len(a.doc.Headings) {
			h := a.doc.Headings[a.selectedIdx]
			pos := fmt.Sprintf("[%d/%d]", a.selectedIdx+1, len(a.doc.Headings))
			heading := strings.Repeat("#", h.Level) + " " + h.Text
			leftPlain = pos + " " + heading
		}
		if a.statusMsg != "" {
			leftPlain += "  " + a.statusMsg
		}
	}

	var rightPlain string
	switch a.mode {
	case ModeNodeSelect:
		rightPlain = ""
	case ModeJump:
		rightPlain = ""
	default:
		if a.sidebarHidden {
			rightPlain = "w:sidebar  Tab:focus  e:edit  i:interactive  /:search  T:theme  ?:help  q:quit"
		} else {
			rightPlain = "Tab:focus  w:hide sidebar  e:edit  i:interactive  /:search  T:theme  ?:help  q:quit"
		}
	}

	// Use rune counts (no ANSI codes yet) for padding calculation.
	// Subtract 2 for the 1-cell padding on each side added by lipgloss.
	innerWidth := a.width - 2
	usedWidth := len([]rune(leftPlain)) + len([]rune(rightPlain))
	pad := innerWidth - usedWidth
	if pad < 1 {
		pad = 1
	}

	// Now apply colour styling.
	var content string
	switch a.mode {
	case ModeSearch:
		content = lipgloss.NewStyle().Foreground(a.theme.Search).Bold(true).Render(leftPlain) +
			strings.Repeat(" ", pad) + rightPlain
	case ModeNodeSelect:
		content = lipgloss.NewStyle().Foreground(a.theme.NodeSel).Bold(true).Render(leftPlain)
	case ModeJump:
		content = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5f00")).Bold(true).Render(leftPlain)
	default:
		content = leftPlain + strings.Repeat(" ", pad) + rightPlain
	}

	return lipgloss.NewStyle().
		Background(a.theme.StatusBar).
		Foreground(a.theme.Foreground).
		Width(a.width).
		Padding(0, 1).
		Render(content)
}

func (a *App) overlayHelp(background string) string {
	var focusState string
	if a.sidebarHidden {
		focusState = "sidebar hidden (w to restore)"
	} else if a.focus == FocusSidebar {
		focusState = "sidebar focused"
	} else {
		focusState = "content focused"
	}

	km := a.cfg.Keys

	// fmtKeys formats a key binding string for display:
	// "j,down" → "j / ↓", "ctrl+d,ctrl+f,pgdown, " → "Ctrl+D / Ctrl+F / PgDn / Space"
	fmtKeys := func(binding string) string {
		var parts []string
		for _, k := range strings.Split(binding, ",") {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			switch k {
			case "up":
				k = "↑"
			case "down":
				k = "↓"
			case "left":
				k = "←"
			case "right":
				k = "→"
			case "tab":
				k = "Tab"
			case "shift+tab":
				k = "Shift+Tab"
			case "enter":
				k = "Enter"
			case "esc":
				k = "Esc"
			case "pgup":
				k = "PgUp"
			case "pgdown":
				k = "PgDn"
			case "space":
				k = "Space"
			default:
				if strings.HasPrefix(k, "ctrl+") {
					k = "Ctrl+" + strings.ToUpper(k[5:])
				}
			}
			parts = append(parts, k)
		}
		return strings.Join(parts, " / ")
	}

	type helpEntry struct {
		keys string
		desc string
	}

	// Build sections from config.
	global := []helpEntry{
		{fmtKeys(km.ToggleFocus), "Toggle focus sidebar ↔ content"},
		{fmtKeys(km.ToggleSidebar), "Hide / show sidebar"},
		{fmtKeys(km.Edit), "Open file in $EDITOR"},
		{fmtKeys(km.Reload), "Reload file"},
		{fmtKeys(km.ThemePicker), "Open theme picker"},
		{fmtKeys(km.Help), "Toggle this help"},
		{fmtKeys(km.Quit), "Quit"},
	}

	sidebar := []helpEntry{
		{fmtKeys(km.SidebarDown), "Select next heading"},
		{fmtKeys(km.SidebarUp), "Select previous heading"},
		{fmtKeys(km.SidebarTop) + " / " + fmtKeys(km.SidebarBottom), "Jump to root / last heading"},
		{fmtKeys(km.JumpHigh) + " / " + fmtKeys(km.JumpMid) + " / " + fmtKeys(km.JumpLow), "Top / mid / bottom of visible area"},
		{fmtKeys(km.ViewCenter) + " / " + fmtKeys(km.ViewTop) + " / " + fmtKeys(km.ViewBottom), "Center / top / bottom in viewport"},
		{fmtKeys(km.SidebarSearch), "Search headings"},
		{fmtKeys(km.NextMatch) + " / " + fmtKeys(km.PrevMatch), "Next / previous match"},
	}

	content := []helpEntry{
		{fmtKeys(km.ScrollDown), "Scroll down one line"},
		{fmtKeys(km.ScrollUp), "Scroll up one line"},
		{fmtKeys(km.ScrollHalfDown), "Page down"},
		{fmtKeys(km.ScrollHalfUp), "Page up"},
		{fmtKeys(km.ContentTop) + " / " + fmtKeys(km.ContentBottom), "Jump to top / bottom"},
		{fmtKeys(km.ContentSearch), "Search content"},
		{fmtKeys(km.NodeSelect), "Enter interactive mode"},
		{fmtKeys(km.Jump), "Jump mode (EasyMotion labels)"},
	}

	interactive := []helpEntry{
		{fmtKeys(km.NodeNext), "Next node"},
		{fmtKeys(km.NodePrev), "Previous node"},
		{"m", "Cycle sub-mode (all/code/inline/links)"},
		{fmtKeys(km.JumpHigh) + " / " + fmtKeys(km.JumpMid) + " / " + fmtKeys(km.JumpLow), "Top / mid / bottom visible node"},
		{fmtKeys(km.ViewCenter) + " / " + fmtKeys(km.ViewTop) + " / " + fmtKeys(km.ViewBottom), "Center / top / bottom in viewport"},
		{fmtKeys(km.NodeCopy), "Copy to clipboard"},
		{fmtKeys(km.NodeOpen), "Open link"},
		{fmtKeys(km.NodeExit), "Exit interactive mode"},
	}

	// Find max key width across all sections for alignment.
	maxKeyW := 0
	for _, entries := range [][]helpEntry{global, sidebar, content, interactive} {
		for _, e := range entries {
			if w := len([]rune(e.keys)); w > maxKeyW {
				maxKeyW = w
			}
		}
	}

	renderSection := func(entries []helpEntry) string {
		var lines []string
		for _, e := range entries {
			pad := maxKeyW - len([]rune(e.keys)) + 2
			if pad < 2 {
				pad = 2
			}
			lines = append(lines, "    "+e.keys+strings.Repeat(" ", pad)+e.desc)
		}
		return strings.Join(lines, "\n")
	}

	text := fmt.Sprintf("gomd — Keyboard Shortcuts   [%s]\n\n", focusState)
	text += "  GLOBAL\n" + renderSection(global) + "\n\n"
	text += "  SIDEBAR  (when focused)\n" + renderSection(sidebar) + "\n\n"
	text += "  CONTENT  (when focused)\n" + renderSection(content) + "\n\n"
	text += "  INTERACTIVE  (press i from content)\n" + renderSection(interactive) + "\n\n"
	text += "  Press any key to dismiss"

	// Compute modal width from content.
	modalW := 0
	for _, line := range strings.Split(text, "\n") {
		if w := lipgloss.Width(line); w > modalW {
			modalW = w
		}
	}
	modalW += 6 // padding (1 each side) + border (1 each side) + margin
	if modalW > a.width-4 {
		modalW = a.width - 4
	}

	// Build bordered modal
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(a.theme.Border).
		Background(a.theme.Background).
		Foreground(a.theme.Foreground).
		Width(modalW-2). // inner width
		Padding(0, 1)

	modal := boxStyle.Render(text)
	modalRenderedLines := strings.Split(modal, "\n")
	modalH := len(modalRenderedLines)

	// Calculate centering offsets
	startRow := (a.height - modalH) / 2
	if startRow < 0 {
		startRow = 0
	}
	startCol := (a.width - modalW) / 2
	if startCol < 0 {
		startCol = 0
	}

	// ANSI-aware overlay of modal onto background
	bgLines := strings.Split(background, "\n")
	for i, mLine := range modalRenderedLines {
		row := startRow + i
		if row >= len(bgLines) {
			break
		}
		mWidth := lipgloss.Width(mLine)
		left := ansi.Truncate(bgLines[row], startCol, "")
		right := ansi.TruncateLeft(bgLines[row], startCol+mWidth, "")
		bgLines[row] = left + mLine + right
	}

	return strings.Join(bgLines, "\n")
}

func (a *App) overlayThemePicker(background string) string {
	themeNames := []string{"OceanDark", "Nord", "Dracula", "Gruvbox", "TokyoNight"}

	// Header with current resolved theme name.
	var lines []string
	lines = append(lines, fmt.Sprintf("Theme: %s", a.theme.Name), "")

	// Theme selection list.
	lines = append(lines, "Select Theme (press number):", "")
	for i, name := range themeNames {
		marker := "  "
		if name == a.cfg.UI.Theme {
			marker = "→ "
		}
		lines = append(lines, fmt.Sprintf("%s%d. %s", marker, i+1, name))
	}

	// Resolved color swatches.
	lines = append(lines, "", "Resolved Colors:", "")
	type colorEntry struct {
		label string
		value lipgloss.Color
	}
	entries := []colorEntry{
		{"Border", a.theme.Border},
		{"Selected", a.theme.Selected},
		{"Heading1", a.theme.Heading1},
		{"Heading2", a.theme.Heading2},
		{"Heading3", a.theme.Heading3},
		{"HeadingN", a.theme.HeadingN},
		{"Background", a.theme.Background},
		{"Foreground", a.theme.Foreground},
		{"StatusBar", a.theme.StatusBar},
		{"Highlight", a.theme.Highlight},
		{"Code", a.theme.Code},
		{"Search", a.theme.Search},
		{"NodeSel", a.theme.NodeSel},
	}
	for _, e := range entries {
		hex := string(e.value)
		swatch := lipgloss.NewStyle().
			Background(e.value).
			Foreground(lipgloss.Color("#000000")).
			Render(fmt.Sprintf(" %-10s ", hex))
		lines = append(lines, fmt.Sprintf("  %-12s %s", e.label, swatch))
	}

	lines = append(lines, "", "Esc / q / T to dismiss")

	text := strings.Join(lines, "\n")

	// Responsive modal width.
	modalW := 50
	if modalW > a.width-4 {
		modalW = a.width - 4
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(a.theme.Border).
		Background(a.theme.Background).
		Foreground(a.theme.Foreground).
		Width(modalW-2).
		Padding(0, 1)

	modal := boxStyle.Render(text)
	modalRenderedLines := strings.Split(modal, "\n")
	modalH := len(modalRenderedLines)

	startRow := (a.height - modalH) / 2
	if startRow < 0 {
		startRow = 0
	}
	startCol := (a.width - modalW) / 2
	if startCol < 0 {
		startCol = 0
	}

	bgLines := strings.Split(background, "\n")
	for i, mLine := range modalRenderedLines {
		row := startRow + i
		if row >= len(bgLines) {
			break
		}
		mWidth := lipgloss.Width(mLine)
		left := ansi.Truncate(bgLines[row], startCol, "")
		right := ansi.TruncateLeft(bgLines[row], startCol+mWidth, "")
		bgLines[row] = left + mLine + right
	}

	return strings.Join(bgLines, "\n")
}

// ─────────────────────────────────────────────
// Syntax highlighting
// ─────────────────────────────────────────────

func highlightCode(code, lang string) string {
	if lang == "" {
		return code
	}
	lx := lexers.Get(lang)
	if lx == nil {
		lx = lexers.Fallback
	}
	lx = chroma.Coalesce(lx)

	sty := styles.Get("monokai")
	if sty == nil {
		sty = styles.Fallback
	}

	fmtr := formatters.Get("terminal256")
	if fmtr == nil {
		return code
	}

	var sb strings.Builder
	it, err := lx.Tokenise(nil, code)
	if err != nil {
		return code
	}
	if err := fmtr.Format(&sb, sty, it); err != nil {
		return code
	}
	return sb.String()
}

// ─────────────────────────────────────────────
// Entry point
// ─────────────────────────────────────────────

func Run(doc *parser.Document, filename, filePath string, cfg config.Config) error {
	app := NewApp(doc, filename, filePath, cfg)
	p := tea.NewProgram(app)
	_, err := p.Run()
	return err
}
