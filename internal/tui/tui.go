// Package tui provides the interactive terminal user interface for gomd.
package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
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

func GetTheme(name string) Theme {
	if t, ok := themes[name]; ok {
		return t
	}
	return themes["OceanDark"]
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
)

const (
	nodeKindCount  = 3  // number of node sub-modes (code, inline, link)
	helpModalWidth = 60 // fixed width for the help modal overlay
	tabStopWidth   = 8  // tab expansion width
)

// ─────────────────────────────────────────────
// CodeNode — a selectable node inside a section
// ─────────────────────────────────────────────

type nodeKind int

const (
	nodeCodeBlock nodeKind = iota
	nodeInlineCode
	nodeLink
)

type codeNode struct {
	kind      nodeKind
	lang      string
	content   string // raw code without fence lines / backticks; for links: URL
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
	codeNodes    []codeNode // all selectable nodes in current section
	nodeSelIdx   int        // which node is highlighted (within filtered list)
	nodeSubMode  nodeKind   // current sub-mode filter
	copyMsg      string     // transient "Copied!" feedback

	// File watcher
	watcher *fsnotify.Watcher

	// Status
	statusMsg string
}

// ─────────────────────────────────────────────
// Construction
// ─────────────────────────────────────────────

func NewApp(doc *parser.Document, filename, filePath string, cfg config.Config) *App {
	a := &App{
		doc:         doc,
		filename:    filename,
		filepath:    filePath,
		cfg:         cfg,
		theme:       GetTheme(cfg.UI.Theme),
		focus:       FocusContent,
		selectedIdx: -1, // start on the root (Document) node
	}
	a.rebuildSection()
	return a
}

// ─────────────────────────────────────────────
// Section helpers
// ─────────────────────────────────────────────

// rebuildSection recomputes sectionLines and codeNodes for the selected heading.
// glamourStyleFor maps a gomd theme name to a glamour standard style name.
func glamourStyleFor(themeName string) string {
	switch themeName {
	case "Dracula":
		return "dracula"
	case "TokyoNight":
		return "tokyo-night"
	default:
		return "dark"
	}
}

// ensureGlamourRenderer returns a cached glamour renderer for the given inner width
// and current theme, rebuilding it only when those change.
func (a *App) ensureGlamourRenderer(innerW int) interface{ Render(string) (string, error) } {
	styleName := glamourStyleFor(a.theme.Name)
	if a.glamourRenderer != nil && a.glamourWidth == innerW && a.glamourStyleName == styleName {
		return a.glamourRenderer
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(styleName),
		glamour.WithWordWrap(innerW),
	)
	if err != nil {
		return nil
	}
	a.glamourRenderer = r
	a.glamourWidth = innerW
	a.glamourStyleName = styleName
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

// renderGlamour renders a markdown string through glamour and returns the output
// split into lines, with a trailing empty line stripped.
func (a *App) renderGlamour(markdown string, innerW int) []string {
	r := a.ensureGlamourRenderer(innerW)
	if r == nil {
		return strings.Split(markdown, "\n")
	}
	out, err := r.Render(markdown)
	if err != nil {
		return strings.Split(markdown, "\n")
	}
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

	// selectedIdx == -1 means the root (Document) node: show entire file
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
		// Check for [text](url)
		if line[i] == '[' {
			// Find closing ]
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
				// Find closing )
				k := j + 1
				for k < len(line) && line[k] != ')' {
					k++
				}
				if k < len(line) {
					url := line[j+1 : k]
					nodes = append(nodes, codeNode{
						kind:      nodeLink,
						lang:      "link",
						content:   url,
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
		} else {
			i++
		}
	}
	return nodes
}

// filteredNodeIndices returns the indices into a.codeNodes for each node
// matching the current sub-mode filter.
func (a *App) filteredNodeIndices() []int {
	var indices []int
	for i, n := range a.codeNodes {
		if n.kind == a.nodeSubMode {
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

	case tea.KeyMsg:
		return a.handleKey(msg)
	}

	return a, nil
}

// ─────────────────────────────────────────────
// Key handling — dispatched by mode then focus
// ─────────────────────────────────────────────

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	}

	// Normal mode — shared keys regardless of focus
	km := a.cfg.Keys
	switch {
	case config.KeyMatches(k, km.Quit):
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

// ─────────────────────────────────────────────
// Sidebar keys
// ─────────────────────────────────────────────

func (a *App) handleSidebarKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	km := a.cfg.Keys
	switch {
	case config.KeyMatches(k, km.SidebarDown):
		a.moveSidebarDown(1)
	case config.KeyMatches(k, km.SidebarUp):
		a.moveSidebarUp(1)
	case k == "pgdown":
		a.moveSidebarDown(a.outlineHeight())
	case k == "pgup":
		a.moveSidebarUp(a.outlineHeight())
	case config.KeyMatches(k, km.SidebarTop):
		a.selectedIdx = -1
		a.scrollOutlineToSelected()
		a.rebuildSection()
	case config.KeyMatches(k, km.SidebarBottom):
		if len(a.doc.Headings) > 0 {
			a.selectedIdx = len(a.doc.Headings) - 1
		}
		a.scrollOutlineToSelected()
		a.rebuildSection()
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

func (a *App) handleContentKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	km := a.cfg.Keys
	switch {
	case config.KeyMatches(k, km.ScrollDown):
		a.scrollContent(1)
	case config.KeyMatches(k, km.ScrollUp):
		a.scrollContent(-1)
	case config.KeyMatches(k, km.ScrollHalfDown):
		a.scrollContent(a.contentHeight() / 2)
	case config.KeyMatches(k, km.ScrollHalfUp):
		a.scrollContent(-a.contentHeight() / 2)
	case config.KeyMatches(k, km.ContentTop):
		a.contentOffset = 0
	case config.KeyMatches(k, km.ContentBottom):
		a.contentOffset = len(a.activeLines())
		a.clampContentOffset()
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
	}
	return a, nil
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
	a.nodeSubMode = nodeCodeBlock
	a.nodeSelIdx = 0

	// Find the first sub-mode that has nodes
	if len(a.filteredNodeIndices()) == 0 {
		for _, mode := range []nodeKind{nodeInlineCode, nodeLink} {
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

// ─────────────────────────────────────────────
// Node selection mode keys
// ─────────────────────────────────────────────

func (a *App) handleNodeSelectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	km := a.cfg.Keys
	filtered := a.filteredNodes()
	switch {
	case config.KeyMatches(k, km.NodeExit):
		a.mode = ModeNormal
		a.copyMsg = ""
	case k == "m":
		// Cycle sub-mode: code -> inline -> links -> code ...
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
			a.mode = ModeNormal
		}
	}
	return a, nil
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

func (a *App) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		a.searchMatches = nil
		q := strings.ToLower(a.searchQuery)
		for i, h := range a.doc.Headings {
			if strings.Contains(strings.ToLower(h.Text), q) {
				a.searchMatches = append(a.searchMatches, i)
			}
		}
		a.searchIdx = 0
		if len(a.searchMatches) > 0 {
			a.selectedIdx = a.searchMatches[0]
			a.scrollOutlineToSelected()
			a.rebuildSection()
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
		if len(msg.Runes) > 0 {
			a.searchQuery += string(msg.Runes)
		}
	}
	return a, nil
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

func (a *App) handleContentSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		if len(msg.Runes) > 0 {
			a.contentSearchQuery += string(msg.Runes)
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

func (a *App) handleThemeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	themeNames := []string{"OceanDark", "Nord", "Dracula", "Gruvbox", "TokyoNight"}
	switch msg.String() {
	case "esc", "q", "T":
		a.mode = ModeNormal
	default:
		for i, name := range themeNames {
			if msg.String() == fmt.Sprintf("%d", i+1) {
				a.theme = GetTheme(name)
				a.cfg.UI.Theme = name
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
	if a.width < 60 {
		return a.width / 3
	}
	return a.width / 4
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

func (a *App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	switch a.mode {
	case ModeThemePicker:
		return a.renderThemePicker()
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

	return view
}

// ─────────────────────────────────────────────
// Individual pane renderers
// ─────────────────────────────────────────────

func (a *App) renderTitle() string {
	name := filepath.Base(a.filename)
	if name == "" || name == "." {
		name = "gomd"
	}

	return lipgloss.NewStyle().
		Background(a.theme.Border).
		Foreground(a.theme.Foreground).
		Bold(true).
		Width(a.width).
		Padding(0, 1).
		Render("gomd — " + name)
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
	bc := lipgloss.NewStyle().Foreground(borderColor)

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
	for _, line := range lines {
		// Normalize line to exactly innerW display columns so the right
		// border always appears in the correct column regardless of whether
		// the line contains ANSI sequences (e.g. from glamour).
		line = ansi.Truncate(line, innerW, "")
		visW := lipgloss.Width(line)
		if visW < innerW {
			line += strings.Repeat(" ", innerW-visW)
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
				Foreground(fg).
				Width(innerW).
				Render(text))
		}
	}
	for len(lines) < h {
		lines = append(lines, lipgloss.NewStyle().Width(innerW).Render(""))
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
			} else {
				// Glamour renders inline code spans with two spaces on each side
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

					// Check padded form
					if paddedIdx := strings.Index(s, padded); paddedIdx != -1 {
						col := colOffset + paddedIdx + 2
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
							// Must have space before (or be at start after indent)
							hasPre := absIdx > 0 && fullLine[absIdx-1] == ' '
							// Must have space after (or be at end of trimmed content)
							hasPost := endIdx >= len(strings.TrimRight(fullLine, " ")) || fullLine[endIdx] == ' '
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
			}
			if len(filtered) > 0 {
				leftPlain = fmt.Sprintf("%s [%d/%d]  m:mode  y:copy  j/k:nav  Esc:exit",
					modeName, a.nodeSelIdx+1, len(filtered))
			} else {
				leftPlain = fmt.Sprintf("%s [0/0]  m:mode  Esc:exit", modeName)
			}
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
	default:
		if a.sidebarHidden {
			rightPlain = "w:sidebar  Tab:focus  e:edit  i:nodes  /:search  T:theme  ?:help  q:quit"
		} else {
			rightPlain = "Tab:focus  w:hide sidebar  e:edit  i:nodes  /:search  T:theme  ?:help  q:quit"
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

	text := fmt.Sprintf(`gomd — Keyboard Shortcuts   [%s]

  GLOBAL
    Tab          Toggle focus between sidebar ↔ content
    w            Hide / show sidebar
    e            Open file in $EDITOR
    r            Reload file
    T            Open theme picker
    ?            Toggle this help
    q / Ctrl+C   Quit

  SIDEBAR  (when focused)
    j / ↓        Select next heading
    k / ↑        Select previous heading
    g / G        Jump to root / last heading
    /            Search headings
    n / N        Next / previous search match

  CONTENT  (when focused)
    j / ↓        Scroll down one line
    k / ↑        Scroll up one line
    Space/Ctrl+D Page down
    Ctrl+U/B     Page up
    g / G        Jump to top / bottom
    /            Search content
    i            Enter interactive node selection

  NODE SELECTION  (press i from content)
    j / ↓ / Tab       Next node
    k / ↑ / Shift+Tab Previous node
    m            Cycle sub-mode (code/inline/links)
    y            Copy to clipboard and exit
    Esc / q / i  Exit node selection

  Press any key to dismiss`, focusState)

	// Fixed modal width
	modalW := helpModalWidth
	if modalW > a.width-4 {
		modalW = a.width - 4
	}

	// Build bordered modal
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(a.theme.Border).
		Background(a.theme.Background).
		Foreground(a.theme.Foreground).
		Width(modalW - 2). // inner width
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

func (a *App) renderThemePicker() string {
	themeNames := []string{"OceanDark", "Nord", "Dracula", "Gruvbox", "TokyoNight"}
	var lines []string
	lines = append(lines, "  Select Theme (press number):", "")
	for i, name := range themeNames {
		marker := "  "
		if name == a.cfg.UI.Theme {
			marker = "→ "
		}
		lines = append(lines, fmt.Sprintf("  %s%d. %s", marker, i+1, name))
	}
	lines = append(lines, "", "  Esc / q / T to cancel")

	return lipgloss.NewStyle().
		Background(a.theme.Background).
		Foreground(a.theme.Foreground).
		Width(a.width).
		Height(a.height).
		Padding(1, 2).
		Render(strings.Join(lines, "\n"))
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
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
