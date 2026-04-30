// Package tui provides the interactive terminal user interface for gomd.
package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/glamour"
	tea "github.com/charmbracelet/bubbletea"
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

const (
	ModeNormal       AppMode = iota
	ModeSearch                // sidebar heading search (/)
	ModeHelp
	ModeThemePicker
	ModeNodeSelect // interactive node selection in content pane
)

// ─────────────────────────────────────────────
// CodeNode — a selectable code block inside a section
// ─────────────────────────────────────────────

type codeNode struct {
	lang      string
	content   string  // raw code without fence lines / backticks
	startLine int     // 0-based line index in sectionLines
	endLine   int     // inclusive (== startLine for inline)
	inline    bool    // true for backtick inline code spans
	colStart  int     // byte offset of opening backtick in the line (inline only)
	colEnd    int     // byte offset just past closing backtick (inline only)
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
	glamourRenderer   interface{ Render(string) (string, error) }
	glamourWidth      int    // innerW for which renderer was built
	glamourStyleName  string // gomd theme name for which renderer was built

	// rendered lines cache — rebuilt when section or width changes
	renderedLinesWidth int   // innerW for which renderedLines was built
	renderedLinesIdx   int   // selectedIdx for which renderedLines was built
	// nodeRenderedLine[i] = first rendered-line index that belongs to codeNodes[i].
	// -1 means the node wasn't found in the rendered output.
	// Rebuilt whenever renderedLines is rebuilt.
	nodeRenderedLine []int

	// Search (sidebar heading search)
	mode          AppMode
	searchQuery   string
	searchMatches []int
	searchIdx     int

	// Interactive node selection
	codeNodes    []codeNode // code blocks found in current section
	nodeSelIdx   int        // which code node is highlighted
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
		focus:       FocusSidebar,
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
			spaces := 8 - (col % 8)
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
	a.nodeRenderedLine = nil
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
				// Scan for inline backtick spans on this line
				nodes = append(nodes, extractInlineNodes(line, i)...)
			}
		} else {
			if strings.HasPrefix(trimmed, fence) {
				nodes = append(nodes, codeNode{
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
	case ModeHelp:
		a.mode = ModeNormal
		return a, nil
	case ModeThemePicker:
		return a.handleThemeKey(msg)
	case ModeNodeSelect:
		return a.handleNodeSelectKey(msg)
	}

	// Normal mode — shared keys regardless of focus
	switch k {
	case "q":
		if a.watcher != nil {
			a.watcher.Close()
		}
		return a, tea.Quit
	case "?":
		a.mode = ModeHelp
		return a, nil
	case "T":
		a.mode = ModeThemePicker
		return a, nil
	case "tab":
		a.toggleFocus()
		return a, nil
	case "w":
		a.sidebarHidden = !a.sidebarHidden
		if a.sidebarHidden {
			a.focus = FocusContent
		}
		return a, nil
	case "r":
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
	case "e":
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
	switch msg.String() {
	case "j", "down":
		a.moveSidebarDown(1)
	case "k", "up":
		a.moveSidebarUp(1)
	case "pgdown":
		a.moveSidebarDown(a.outlineHeight())
	case "pgup":
		a.moveSidebarUp(a.outlineHeight())
	case "g":
		a.selectedIdx = -1
		a.scrollOutlineToSelected()
		a.rebuildSection()
	case "G":
		if len(a.doc.Headings) > 0 {
			a.selectedIdx = len(a.doc.Headings) - 1
		}
		a.scrollOutlineToSelected()
		a.rebuildSection()
	case "/":
		a.mode = ModeSearch
		a.searchQuery = ""
		a.searchMatches = nil
	case "n":
		a.nextSearchMatch()
	case "N":
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
	switch msg.String() {
	case "j", "down":
		a.scrollContent(1)
	case "k", "up":
		a.scrollContent(-1)
	case "ctrl+d", "ctrl+f", "pgdown":
		a.scrollContent(a.contentHeight() / 2)
	case "ctrl+u", "ctrl+b", "pgup":
		a.scrollContent(-a.contentHeight() / 2)
	case "g":
		a.contentOffset = 0
	case "G":
		a.contentOffset = len(a.activeLines())
		a.clampContentOffset()
	case "i":
		// Enter node selection mode if there are code blocks
		if len(a.codeNodes) > 0 {
			a.mode = ModeNodeSelect
			a.nodeSelIdx = 0
			a.scrollContentToNode(a.nodeSelIdx)
			a.statusMsg = ""
		} else {
			a.statusMsg = "No code blocks in this section"
		}
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
// Node selection mode keys
// ─────────────────────────────────────────────

func (a *App) handleNodeSelectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "i":
		a.mode = ModeNormal
		a.copyMsg = ""
	case "j", "down", "tab":
		if len(a.codeNodes) > 0 {
			a.nodeSelIdx = (a.nodeSelIdx + 1) % len(a.codeNodes)
			a.scrollContentToNode(a.nodeSelIdx)
		}
	case "k", "up", "shift+tab":
		if len(a.codeNodes) > 0 {
			a.nodeSelIdx = (a.nodeSelIdx - 1 + len(a.codeNodes)) % len(a.codeNodes)
			a.scrollContentToNode(a.nodeSelIdx)
		}
	case "y":
		if len(a.codeNodes) > 0 {
			node := a.codeNodes[a.nodeSelIdx]
			if err := clipboard.WriteAll(node.content); err != nil {
				a.copyMsg = "Clipboard error: " + err.Error()
			} else {
				lang := node.lang
				if lang == "" {
					lang = "block"
				}
				a.copyMsg = fmt.Sprintf("Copied %s block (%d lines)", lang, strings.Count(node.content, "\n")+1)
			}
			// Exit interactive mode after copy, keep copy feedback in statusMsg
			a.statusMsg = a.copyMsg
			a.copyMsg = ""
			a.mode = ModeNormal
		}
	}
	return a, nil
}

// scrollContentToNode scrolls the content so the selected code node is visible.
func (a *App) scrollContentToNode(nodeIdx int) {
	if nodeIdx < 0 || nodeIdx >= len(a.codeNodes) {
		return
	}
	// Use the rendered-line position if available, fall back to source line.
	target := a.codeNodes[nodeIdx].startLine
	if a.nodeRenderedLine != nil && nodeIdx < len(a.nodeRenderedLine) && a.nodeRenderedLine[nodeIdx] >= 0 {
		target = a.nodeRenderedLine[nodeIdx]
	}
	if target < a.contentOffset || target >= a.contentOffset+a.contentHeight() {
		a.contentOffset = target
		a.clampContentOffset()
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
				a.nodeRenderedLine = nil
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
	case ModeHelp:
		return a.renderHelp()
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

	return lipgloss.JoinVertical(lipgloss.Left, title, body, status)
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
	// Rebuild rendered lines if section or width changed.
	if a.renderedLinesIdx != a.selectedIdx || a.renderedLinesWidth != w {
		markdown := strings.Join(a.sectionLines, "\n")
		a.renderedLines = a.renderGlamour(markdown, w)
		a.renderedLinesIdx = a.selectedIdx
		a.renderedLinesWidth = w
		a.nodeRenderedLine = mapNodesToRenderedLines(a.codeNodes, a.renderedLines)
		// Re-clamp offset now that renderedLines is fresh.
		a.clampContentOffset()
	}

	lines := a.renderedLines

	// In node-select mode, build a highlighted copy of the lines.
	if a.mode == ModeNodeSelect && len(a.codeNodes) > 0 {
		lines = applyNodeHighlights(lines, a.codeNodes, a.nodeSelIdx, a.nodeRenderedLine, a.theme, w)
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

// mapNodesToRenderedLines maps each codeNode to the first rendered-line index
// that contains its content. Returns a slice parallel to nodes; -1 means not found.
//
// Strategy:
//   - For fenced blocks: search for the first rendered line whose stripped text
//     contains the first non-empty line of the block body (or the fence marker).
//   - For inline nodes: search for the first rendered line whose stripped text
//     contains the backtick span text.
func mapNodesToRenderedLines(nodes []codeNode, rendered []string) []int {
	result := make([]int, len(nodes))
	// Pre-strip ANSI from all rendered lines once.
	stripped := make([]string, len(rendered))
	for i, l := range rendered {
		stripped[i] = ansi.Strip(l)
	}

	for ni, n := range nodes {
		result[ni] = -1
		var needle string
		if n.inline {
			// The inline span text including backticks as rendered by glamour
			// (glamour renders inline code as the bare text, no backticks).
			needle = n.content
		} else {
			// For blocks, use the first non-empty body line.
			needle = ""
			for _, bodyLine := range strings.Split(n.content, "\n") {
				trimmed := strings.TrimSpace(bodyLine)
				if trimmed != "" {
					needle = trimmed
					break
				}
			}
			if needle == "" {
				// Empty block — fall back to fence marker.
				needle = "```"
			}
		}
		for ri, s := range stripped {
			if strings.Contains(s, needle) {
				result[ni] = ri
				break
			}
		}
	}
	return result
}

// applyNodeHighlights returns a copy of rendered lines with background colours
// applied for node-select mode:
//   - All lines belonging to any selectable node get a dim "available" background.
//   - All lines belonging to the currently selected node get a bright "selected" background.
func applyNodeHighlights(lines []string, nodes []codeNode, selIdx int, nodeRenderedLine []int, theme Theme, w int) []string {
	out := make([]string, len(lines))
	copy(out, lines)

	// Build a map: rendered line index -> highlight level (1=available, 2=selected)
	type hlLevel int
	const (
		hlNone      hlLevel = 0
		hlAvailable hlLevel = 1
		hlSelected  hlLevel = 2
	)
	lineHL := make(map[int]hlLevel, len(lines))

	for ni, n := range nodes {
		start := nodeRenderedLine[ni]
		if start < 0 {
			continue
		}
		level := hlAvailable
		if ni == selIdx {
			level = hlSelected
		}
		// For inline nodes, highlight just the one line.
		// For blocks, estimate the rendered span by counting body lines + fence lines.
		end := start
		if !n.inline {
			bodyLines := len(strings.Split(strings.TrimRight(n.content, "\n"), "\n"))
			// rendered block = fence-open + body + fence-close + possible blank lines;
			// be conservative: highlight from start to start+bodyLines+1
			end = start + bodyLines + 1
			if end >= len(lines) {
				end = len(lines) - 1
			}
		}
		for li := start; li <= end; li++ {
			if existing := lineHL[li]; existing < level {
				lineHL[li] = level
			}
		}
	}

	dimBg := theme.Code       // re-use the code-block bg as "available" tint
	selBg := theme.NodeSel    // bright selected bg
	selFg := theme.Background // contrast fg for selected

	for li, level := range lineHL {
		if li >= len(out) {
			break
		}
		stripped := ansi.Strip(out[li])
		// Pad to full width so the background colour fills the line.
		visW := lipgloss.Width(stripped)
		if visW < w {
			stripped += strings.Repeat(" ", w-visW)
		}
		switch level {
		case hlAvailable:
			out[li] = lipgloss.NewStyle().Background(dimBg).Render(stripped)
		case hlSelected:
			out[li] = lipgloss.NewStyle().Background(selBg).Foreground(selFg).Bold(true).Render(stripped)
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
	case ModeNodeSelect:
		if a.copyMsg != "" {
			leftPlain = "✓ " + a.copyMsg
		} else if len(a.codeNodes) > 0 {
			n := a.codeNodes[a.nodeSelIdx]
			lang := n.lang
			if lang == "" {
				lang = "code"
			}
			leftPlain = fmt.Sprintf("NODE [%d/%d] %s  y:copy  j/k:next/prev  Esc:exit",
				a.nodeSelIdx+1, len(a.codeNodes), lang)
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

func (a *App) renderHelp() string {
	var focusState string
	if a.sidebarHidden {
		focusState = "sidebar hidden (w to restore)"
	} else if a.focus == FocusSidebar {
		focusState = "sidebar focused"
	} else {
		focusState = "content focused"
	}

	text := fmt.Sprintf(`  gomd — Keyboard Shortcuts   [%s]

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
    Ctrl+D/F     Page down
    Ctrl+U/B     Page up
    g / G        Jump to top / bottom
    i            Enter interactive node selection

  NODE SELECTION  (press i from content)
    j / ↓ / Tab       Next code block
    k / ↑ / Shift+Tab Previous code block
    y            Copy to clipboard and exit
    Esc / q / i  Exit node selection
`, focusState)

	return lipgloss.NewStyle().
		Background(a.theme.Background).
		Foreground(a.theme.Foreground).
		Width(a.width).
		Height(a.height).
		Padding(1, 2).
		Render(text)
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
