// Package tui provides the interactive terminal user interface for gomd.
package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	outlineOffset int // first visible heading index
	selectedIdx   int // currently selected heading index

	// Content — shows only the section of the selected heading
	sectionLines  []string // lines of the current section
	contentOffset int      // scroll offset within sectionLines

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
		doc:      doc,
		filename: filename,
		filepath: filePath,
		cfg:      cfg,
		theme:    GetTheme(cfg.UI.Theme),
		focus:    FocusSidebar,
	}
	a.rebuildSection()
	return a
}

// ─────────────────────────────────────────────
// Section helpers
// ─────────────────────────────────────────────

// rebuildSection recomputes sectionLines and codeNodes for the selected heading.
func (a *App) rebuildSection() {
	a.contentOffset = 0
	a.nodeSelIdx = 0
	a.codeNodes = nil

	if len(a.doc.Headings) == 0 {
		a.sectionLines = strings.Split(a.doc.Content, "\n")
		return
	}

	// Clamp
	if a.selectedIdx < 0 {
		a.selectedIdx = 0
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
	// Trim trailing blank lines
	section = strings.TrimRight(section, "\n")
	a.sectionLines = strings.Split(section, "\n")

	// Extract code nodes from this section
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

func (clearCopyMsgCmd) ID() string { return "" }

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
		prevHeading := ""
		if a.selectedIdx < len(a.doc.Headings) {
			prevHeading = a.doc.Headings[a.selectedIdx].Text
		}
		a.doc = msg.doc
		// Try to keep selection on same heading by text
		if prevHeading != "" {
			for i, h := range a.doc.Headings {
				if h.Text == prevHeading {
					a.selectedIdx = i
					break
				}
			}
		}
		if a.selectedIdx >= len(a.doc.Headings) {
			a.selectedIdx = len(a.doc.Headings) - 1
		}
		if a.selectedIdx < 0 {
			a.selectedIdx = 0
		}
		a.rebuildSection()
		a.scrollOutlineToSelected()
		a.statusMsg = "Reloaded"
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
	case "g":
		a.selectedIdx = 0
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
	if len(a.doc.Headings) == 0 {
		return
	}
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
	if a.selectedIdx < 0 {
		a.selectedIdx = 0
	}
	if a.selectedIdx != prev {
		a.scrollOutlineToSelected()
		a.rebuildSection()
	}
}

// scrollOutlineToSelected scrolls the outline viewport so the selected item is
// always visible, but does NOT move the viewport unless the selection has left it.
func (a *App) scrollOutlineToSelected() {
	h := a.outlineHeight()
	if h <= 0 {
		return
	}
	if a.selectedIdx < a.outlineOffset {
		a.outlineOffset = a.selectedIdx
	}
	if a.selectedIdx >= a.outlineOffset+h {
		a.outlineOffset = a.selectedIdx - h + 1
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
	case "ctrl+d", "ctrl+f":
		a.scrollContent(a.contentHeight() / 2)
	case "ctrl+u", "ctrl+b":
		a.scrollContent(-a.contentHeight() / 2)
	case "g":
		a.contentOffset = 0
	case "G":
		a.contentOffset = len(a.sectionLines)
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

func (a *App) clampContentOffset() {
	if a.contentOffset < 0 {
		a.contentOffset = 0
	}
	max := len(a.sectionLines) - a.contentHeight()
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
	case "k", "up":
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
		}
	}
	return a, nil
}

// scrollContentToNode scrolls the content so the selected code node is visible.
func (a *App) scrollContentToNode(nodeIdx int) {
	if nodeIdx < 0 || nodeIdx >= len(a.codeNodes) {
		return
	}
	node := a.codeNodes[nodeIdx]
	// Show the opening fence at the top of the viewport if possible
	target := node.startLine
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
	return a.width - sw - 1 // -1 for divider
}

func (a *App) outlineHeight() int {
	h := a.height - 2 // title row + status row
	if h < 0 {
		return 0
	}
	return h
}

func (a *App) contentHeight() int {
	h := a.height - 2 // title row + status row
	if h < 0 {
		return 0
	}
	return h
}

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
		body = a.renderContent()
	} else {
		divStyle := lipgloss.NewStyle().Foreground(a.theme.Border)
		divLines := make([]string, a.outlineHeight())
		for i := range divLines {
			divLines[i] = "│"
		}
		divider := divStyle.Render(strings.Join(divLines, "\n"))

		if a.focus == FocusSidebar {
			// Outline on the left, section content on the right
			body = lipgloss.JoinHorizontal(lipgloss.Top, a.renderOutline(), divider, a.renderContent())
		} else {
			// Content focused: section content on the left, outline on the right
			body = lipgloss.JoinHorizontal(lipgloss.Top, a.renderContentPane(), divider, a.renderOutline())
		}
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

	focusIndicator := ""
	if a.focus == FocusContent {
		focusIndicator = " [content]"
	}

	return lipgloss.NewStyle().
		Background(a.theme.Border).
		Foreground(a.theme.Foreground).
		Bold(true).
		Width(a.width).
		Padding(0, 1).
		Render("gomd — " + name + focusIndicator)
}

func (a *App) renderOutline() string {
	// Width depends on which side this pane is on.
	// When sidebar-focused: outline is on the left (sidebarWidth).
	// When content-focused: outline is on the right (contentWidth).
	var w int
	if a.focus == FocusSidebar {
		w = a.sidebarWidth()
	} else {
		w = a.contentWidth()
	}
	h := a.outlineHeight()

	headings := a.doc.Headings
	end := a.outlineOffset + h
	if end > len(headings) {
		end = len(headings)
	}

	lines := make([]string, 0, h)

	// Active sidebar border color depends on focus
	selBg := a.theme.Selected
	if a.focus == FocusSidebar {
		selBg = a.theme.Highlight // brighter when focused
	}

	for i := a.outlineOffset; i < end; i++ {
		hd := headings[i]
		indent := strings.Repeat("  ", hd.Level-1)
		marker := strings.Repeat("#", hd.Level)
		text := indent + marker + " " + hd.Text

		// Truncate
		maxRunes := w - 2
		if maxRunes < 1 {
			maxRunes = 1
		}
		if len([]rune(text)) > maxRunes {
			text = string([]rune(text)[:maxRunes-1]) + "…"
		}

		if i == a.selectedIdx {
			lines = append(lines, lipgloss.NewStyle().
				Background(selBg).
				Foreground(a.theme.Background).
				Bold(true).
				Width(w-1).
				Render(text))
		} else {
			var fg lipgloss.Color
			switch hd.Level {
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
				Width(w-1).
				Render(text))
		}
	}

	// Pad
	for len(lines) < h {
		lines = append(lines, strings.Repeat(" ", w-1))
	}

	return lipgloss.NewStyle().Width(w).MaxWidth(w).
		Render(strings.Join(lines, "\n"))
}

// renderContentPane renders the section content in the narrow (sidebar-width) left column,
// used when content pane is focused and the layout is swapped.
func (a *App) renderContentPane() string {
	return a.renderContentWidth(a.sidebarWidth())
}

func (a *App) renderContent() string {
	return a.renderContentWidth(a.contentWidth())
}

func (a *App) renderContentWidth(w int) string {
	h := a.contentHeight()
	if w <= 0 || h <= 0 {
		return ""
	}

	lines := a.sectionLines
	start := a.contentOffset
	if start > len(lines) {
		start = len(lines)
	}
	end := start + h
	if end > len(lines) {
		end = len(lines)
	}
	visible := lines[start:end]

	// Build set of line ranges that belong to the currently-selected node
	// (only in ModeNodeSelect)
	nodeHighlightLines := map[int]bool{}
	var selectedInlineNode *codeNode // non-nil when selected node is inline
	if a.mode == ModeNodeSelect && a.nodeSelIdx < len(a.codeNodes) {
		n := a.codeNodes[a.nodeSelIdx]
		if n.inline {
			selectedInlineNode = &a.codeNodes[a.nodeSelIdx]
		} else {
			for li := n.startLine; li <= n.endLine; li++ {
				nodeHighlightLines[li] = true
			}
		}
	}

	rendered := make([]string, 0, h)
	inCode := false
	var fence, codeLang string
	var codeBody []string
	var fenceDocLine int // absolute index in sectionLines

	for relIdx, line := range visible {
		docLine := start + relIdx // absolute index in sectionLines
		trimmed := strings.TrimSpace(line)

		// ── Code fence detection ──────────────────────────────────
		if !inCode {
			if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
				inCode = true
				fence = trimmed[:3]
				codeLang = strings.TrimSpace(trimmed[3:])
				codeBody = nil
				fenceDocLine = docLine

				fenceLine := lipgloss.NewStyle().Foreground(a.theme.HeadingN).Render(line)
				if nodeHighlightLines[docLine] {
					fenceLine = lipgloss.NewStyle().
						Foreground(a.theme.NodeSel).Bold(true).Render(line)
				}
				rendered = append(rendered, fenceLine)
				continue
			}
		} else {
			if strings.HasPrefix(trimmed, fence) {
				// closing fence — flush the accumulated code
				highlighted := highlightCode(strings.Join(codeBody, "\n"), codeLang)
				hLines := strings.Split(highlighted, "\n")
				// Re-check line numbers for the body lines
				for bi, hl := range hLines {
					bodyDocLine := fenceDocLine + 1 + bi
					if nodeHighlightLines[bodyDocLine] {
						hl = lipgloss.NewStyle().
							Background(a.theme.Code).
							Foreground(a.theme.Foreground).
							Render(hl)
					}
					rendered = append(rendered, hl)
				}
				// closing fence
				closeLine := lipgloss.NewStyle().Foreground(a.theme.HeadingN).Render(line)
				if nodeHighlightLines[docLine] {
					closeLine = lipgloss.NewStyle().
						Foreground(a.theme.NodeSel).Bold(true).Render(line)
				}
				rendered = append(rendered, closeLine)
				inCode = false
				fence = ""
				codeLang = ""
				codeBody = nil
				continue
			}
			// still inside block
			codeBody = append(codeBody, line)
			continue // will be flushed when closing fence is found
		}

		// ── Heading styling ───────────────────────────────────────
		if strings.HasPrefix(line, "#") {
			level := 0
			for level < len(line) && line[level] == '#' {
				level++
			}
			var fg lipgloss.Color
			switch level {
			case 1:
				fg = a.theme.Heading1
			case 2:
				fg = a.theme.Heading2
			case 3:
				fg = a.theme.Heading3
			default:
				fg = a.theme.HeadingN
			}
			rendered = append(rendered, lipgloss.NewStyle().Foreground(fg).Bold(true).Render(line))
			continue
		}

		// ── Plain line ────────────────────────────────────────────
		// Check if this line contains the selected inline code span
		if selectedInlineNode != nil && docLine == selectedInlineNode.startLine &&
			selectedInlineNode.colEnd <= len(line) {
			n := selectedInlineNode
			before := line[:n.colStart]
			span := line[n.colStart:n.colEnd]
			after := line[n.colEnd:]
			highlighted := lipgloss.NewStyle().
				Background(a.theme.NodeSel).
				Foreground(a.theme.Background).
				Bold(true).
				Render(span)
			rendered = append(rendered, before+highlighted+after)
			continue
		}
		rendered = append(rendered, line)
	}

	// Flush unclosed code block (e.g. scrolled past closing fence)
	if inCode && len(codeBody) > 0 {
		highlighted := highlightCode(strings.Join(codeBody, "\n"), codeLang)
		rendered = append(rendered, strings.Split(highlighted, "\n")...)
	}

	// Pad to height
	for len(rendered) < h {
		rendered = append(rendered, "")
	}
	// Trim to height (highlighting can produce extra lines)
	rendered = rendered[:h]

	return lipgloss.NewStyle().Width(w).MaxWidth(w).
		Render(strings.Join(rendered, "\n"))
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
		if a.selectedIdx < len(a.doc.Headings) {
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
			rightPlain = "w:sidebar  Tab:focus  i:nodes  /:search  T:theme  ?:help  q:quit"
		} else {
			rightPlain = "Tab:focus  w:hide sidebar  i:nodes  /:search  T:theme  ?:help  q:quit"
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
    r            Reload file
    T            Open theme picker
    ?            Toggle this help
    q / Ctrl+C   Quit

  SIDEBAR  (when focused)
    j / ↓        Select next heading
    k / ↑        Select previous heading
    g / G        Jump to first / last heading
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
    j / ↓ / Tab  Next code block
    k / ↑        Previous code block
    y            Copy code block content to clipboard
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
