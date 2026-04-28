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
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"

	"github.com/grota/gomd/internal/config"
	"github.com/grota/gomd/internal/parser"
)

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
}

// Built-in themes
var themes = map[string]Theme{
	"OceanDark": {
		Name:       "OceanDark",
		Border:     lipgloss.Color("#4a6fa5"),
		Selected:   lipgloss.Color("#2d5986"),
		Heading1:   lipgloss.Color("#6fb3d2"),
		Heading2:   lipgloss.Color("#59c2a5"),
		Heading3:   lipgloss.Color("#82aaff"),
		HeadingN:   lipgloss.Color("#7f9fbf"),
		Background: lipgloss.Color("#1a2332"),
		Foreground: lipgloss.Color("#c5d4e8"),
		StatusBar:  lipgloss.Color("#253545"),
		Highlight:  lipgloss.Color("#ffd700"),
		Code:       lipgloss.Color("#1e2a3a"),
		Search:     lipgloss.Color("#ff6b6b"),
	},
	"Nord": {
		Name:       "Nord",
		Border:     lipgloss.Color("#4c566a"),
		Selected:   lipgloss.Color("#3b4252"),
		Heading1:   lipgloss.Color("#88c0d0"),
		Heading2:   lipgloss.Color("#81a1c1"),
		Heading3:   lipgloss.Color("#5e81ac"),
		HeadingN:   lipgloss.Color("#616e88"),
		Background: lipgloss.Color("#2e3440"),
		Foreground: lipgloss.Color("#d8dee9"),
		StatusBar:  lipgloss.Color("#3b4252"),
		Highlight:  lipgloss.Color("#ebcb8b"),
		Code:       lipgloss.Color("#272c36"),
		Search:     lipgloss.Color("#bf616a"),
	},
	"Dracula": {
		Name:       "Dracula",
		Border:     lipgloss.Color("#6272a4"),
		Selected:   lipgloss.Color("#44475a"),
		Heading1:   lipgloss.Color("#bd93f9"),
		Heading2:   lipgloss.Color("#ff79c6"),
		Heading3:   lipgloss.Color("#8be9fd"),
		HeadingN:   lipgloss.Color("#6272a4"),
		Background: lipgloss.Color("#282a36"),
		Foreground: lipgloss.Color("#f8f8f2"),
		StatusBar:  lipgloss.Color("#44475a"),
		Highlight:  lipgloss.Color("#f1fa8c"),
		Code:       lipgloss.Color("#21222c"),
		Search:     lipgloss.Color("#ff5555"),
	},
	"Gruvbox": {
		Name:       "Gruvbox",
		Border:     lipgloss.Color("#504945"),
		Selected:   lipgloss.Color("#3c3836"),
		Heading1:   lipgloss.Color("#fabd2f"),
		Heading2:   lipgloss.Color("#b8bb26"),
		Heading3:   lipgloss.Color("#83a598"),
		HeadingN:   lipgloss.Color("#928374"),
		Background: lipgloss.Color("#282828"),
		Foreground: lipgloss.Color("#ebdbb2"),
		StatusBar:  lipgloss.Color("#3c3836"),
		Highlight:  lipgloss.Color("#fe8019"),
		Code:       lipgloss.Color("#1d2021"),
		Search:     lipgloss.Color("#fb4934"),
	},
	"TokyoNight": {
		Name:       "TokyoNight",
		Border:     lipgloss.Color("#3b4261"),
		Selected:   lipgloss.Color("#283457"),
		Heading1:   lipgloss.Color("#7aa2f7"),
		Heading2:   lipgloss.Color("#7dcfff"),
		Heading3:   lipgloss.Color("#bb9af7"),
		HeadingN:   lipgloss.Color("#565f89"),
		Background: lipgloss.Color("#1a1b26"),
		Foreground: lipgloss.Color("#c0caf5"),
		StatusBar:  lipgloss.Color("#1f2335"),
		Highlight:  lipgloss.Color("#e0af68"),
		Code:       lipgloss.Color("#16161e"),
		Search:     lipgloss.Color("#f7768e"),
	},
}

// GetTheme returns a theme by name, defaulting to OceanDark.
func GetTheme(name string) Theme {
	if t, ok := themes[name]; ok {
		return t
	}
	return themes["OceanDark"]
}

// AppMode represents the current application mode.
type AppMode int

const (
	ModeNormal AppMode = iota
	ModeSearch
	ModeDocSearch
	ModeHelp
	ModeThemePicker
	ModeFilePicker
)

// App is the main TUI application state.
type App struct {
	doc      *parser.Document
	filename string
	filepath string
	cfg      config.Config
	theme    Theme

	// Layout
	width  int
	height int

	// Outline pane
	outlineOffset int
	selectedIdx   int

	// Content pane
	contentOffset int
	contentLines  []string

	// Search
	mode       AppMode
	searchQuery string
	searchMatches []int
	searchIdx  int

	// File watcher
	watcher *fsnotify.Watcher

	// Status
	statusMsg string
}

// NewApp creates a new TUI application.
func NewApp(doc *parser.Document, filename, filePath string, cfg config.Config) *App {
	theme := GetTheme(cfg.UI.Theme)
	app := &App{
		doc:      doc,
		filename: filename,
		filepath: filePath,
		cfg:      cfg,
		theme:    theme,
	}
	app.buildContentLines()
	return app
}

type fileReloadMsg struct {
	doc *parser.Document
}

type watchErrMsg struct{ err error }

func (a *App) startWatcher() tea.Cmd {
	if a.filepath == "" || a.filepath == "<stdin>" {
		return nil
	}
	return func() tea.Msg {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return watchErrMsg{err}
		}
		if err := watcher.Add(a.filepath); err != nil {
			watcher.Close()
			return watchErrMsg{err}
		}
		a.watcher = watcher

		for event := range watcher.Events {
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				data, err := os.ReadFile(a.filepath)
				if err != nil {
					continue
				}
				doc := parser.ParseMarkdown(string(data))
				return fileReloadMsg{doc}
			}
		}
		return nil
	}
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
		a.doc = msg.doc
		a.buildContentLines()
		// Clamp selection
		if a.selectedIdx >= len(a.doc.Headings) {
			a.selectedIdx = len(a.doc.Headings) - 1
		}
		if a.selectedIdx < 0 {
			a.selectedIdx = 0
		}
		a.statusMsg = "File reloaded"
		return a, a.startWatcher()

	case tea.KeyMsg:
		return a.handleKey(msg)
	}

	return a, nil
}

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch a.mode {
	case ModeSearch:
		return a.handleSearchKey(msg)
	case ModeHelp:
		if msg.String() == "q" || msg.String() == "?" || msg.String() == "esc" {
			a.mode = ModeNormal
		}
		return a, nil
	case ModeThemePicker:
		return a.handleThemeKey(msg)
	}

	// Normal mode
	switch msg.String() {
	case "q", "ctrl+c":
		if a.watcher != nil {
			a.watcher.Close()
		}
		return a, tea.Quit

	case "j", "down":
		a.moveDown(1)
	case "k", "up":
		a.moveUp(1)
	case "g":
		a.selectedIdx = 0
		a.outlineOffset = 0
		a.contentOffset = 0
	case "G":
		if len(a.doc.Headings) > 0 {
			a.selectedIdx = len(a.doc.Headings) - 1
		}
		a.scrollOutlineToSelected()
		a.scrollContentToHeading()
	case "ctrl+d", "ctrl+f":
		a.scrollContent(a.contentHeight() / 2)
	case "ctrl+u", "ctrl+b":
		a.scrollContent(-a.contentHeight() / 2)
	case "J":
		a.scrollContent(3)
	case "K":
		a.scrollContent(-3)
	case "/":
		a.mode = ModeSearch
		a.searchQuery = ""
		a.searchMatches = nil
	case "n":
		a.nextMatch()
	case "N":
		a.prevMatch()
	case "?":
		a.mode = ModeHelp
	case "T":
		a.mode = ModeThemePicker
	case "enter", " ":
		a.scrollContentToHeading()
	case "r":
		// Manual reload
		if a.filepath != "" && a.filepath != "<stdin>" {
			data, err := os.ReadFile(a.filepath)
			if err == nil {
				a.doc = parser.ParseMarkdown(string(data))
				a.buildContentLines()
				a.statusMsg = "Reloaded"
			}
		}
	}

	return a, nil
}

func (a *App) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Execute search
		a.searchMatches = nil
		query := strings.ToLower(a.searchQuery)
		for i, h := range a.doc.Headings {
			if strings.Contains(strings.ToLower(h.Text), query) {
				a.searchMatches = append(a.searchMatches, i)
			}
		}
		a.searchIdx = 0
		if len(a.searchMatches) > 0 {
			a.selectedIdx = a.searchMatches[0]
			a.scrollOutlineToSelected()
			a.scrollContentToHeading()
		}
		a.mode = ModeNormal

	case "esc":
		a.mode = ModeNormal
		a.searchQuery = ""

	case "backspace", "ctrl+h":
		if len(a.searchQuery) > 0 {
			runes := []rune(a.searchQuery)
			a.searchQuery = string(runes[:len(runes)-1])
		}

	default:
		if len(msg.Runes) > 0 {
			a.searchQuery += string(msg.Runes)
		}
	}

	return a, nil
}

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

func (a *App) moveDown(n int) {
	if len(a.doc.Headings) == 0 {
		return
	}
	a.selectedIdx += n
	if a.selectedIdx >= len(a.doc.Headings) {
		a.selectedIdx = len(a.doc.Headings) - 1
	}
	a.scrollOutlineToSelected()
	a.scrollContentToHeading()
}

func (a *App) moveUp(n int) {
	a.selectedIdx -= n
	if a.selectedIdx < 0 {
		a.selectedIdx = 0
	}
	a.scrollOutlineToSelected()
	a.scrollContentToHeading()
}

func (a *App) nextMatch() {
	if len(a.searchMatches) == 0 {
		return
	}
	a.searchIdx = (a.searchIdx + 1) % len(a.searchMatches)
	a.selectedIdx = a.searchMatches[a.searchIdx]
	a.scrollOutlineToSelected()
	a.scrollContentToHeading()
}

func (a *App) prevMatch() {
	if len(a.searchMatches) == 0 {
		return
	}
	a.searchIdx = (a.searchIdx - 1 + len(a.searchMatches)) % len(a.searchMatches)
	a.selectedIdx = a.searchMatches[a.searchIdx]
	a.scrollOutlineToSelected()
	a.scrollContentToHeading()
}

func (a *App) scrollContent(delta int) {
	a.contentOffset += delta
	if a.contentOffset < 0 {
		a.contentOffset = 0
	}
	maxOffset := len(a.contentLines) - a.contentHeight()
	if maxOffset < 0 {
		maxOffset = 0
	}
	if a.contentOffset > maxOffset {
		a.contentOffset = maxOffset
	}
}

func (a *App) scrollContentToHeading() {
	if a.selectedIdx < 0 || a.selectedIdx >= len(a.doc.Headings) {
		return
	}
	h := a.doc.Headings[a.selectedIdx]
	// Find the line in contentLines that corresponds to this heading
	target := 0
	for i, line := range a.contentLines {
		// Each heading starts a section; find by content
		prefix := strings.Repeat("#", h.Level) + " " + h.Text
		if strings.Contains(line, prefix) {
			target = i
			break
		}
	}
	a.contentOffset = target
	maxOffset := len(a.contentLines) - a.contentHeight()
	if maxOffset < 0 {
		maxOffset = 0
	}
	if a.contentOffset > maxOffset {
		a.contentOffset = maxOffset
	}
}

func (a *App) scrollOutlineToSelected() {
	outlineH := a.outlineHeight()
	if a.selectedIdx < a.outlineOffset {
		a.outlineOffset = a.selectedIdx
	}
	if a.selectedIdx >= a.outlineOffset+outlineH {
		a.outlineOffset = a.selectedIdx - outlineH + 1
	}
	if a.outlineOffset < 0 {
		a.outlineOffset = 0
	}
}

// buildContentLines converts document content into displayable lines with syntax highlighting.
func (a *App) buildContentLines() {
	if a.doc == nil {
		a.contentLines = nil
		return
	}
	a.contentLines = strings.Split(a.doc.Content, "\n")
}

// Layout helpers
func (a *App) outlineWidth() int {
	if a.width < 40 {
		return a.width / 3
	}
	return a.width / 4
}

func (a *App) contentWidth() int {
	return a.width - a.outlineWidth() - 1
}

func (a *App) outlineHeight() int {
	return a.height - 3 // title + status
}

func (a *App) contentHeight() int {
	return a.height - 3
}

func (a *App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	outline := a.renderOutline()
	content := a.renderContent()
	status := a.renderStatus()

	// Render title bar
	title := a.renderTitle()

	// Join panes side by side
	divider := strings.Repeat("│\n", a.height-2)
	dividerStyle := lipgloss.NewStyle().Foreground(a.theme.Border)

	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		outline,
		dividerStyle.Render(divider),
		content,
	)

	switch a.mode {
	case ModeSearch:
		status = a.renderSearchBar()
	case ModeHelp:
		return a.renderHelp()
	case ModeThemePicker:
		return a.renderThemePicker()
	}

	return lipgloss.JoinVertical(lipgloss.Left, title, body, status)
}

func (a *App) renderTitle() string {
	titleStyle := lipgloss.NewStyle().
		Background(a.theme.Border).
		Foreground(a.theme.Foreground).
		Bold(true).
		Width(a.width).
		Padding(0, 1)

	name := filepath.Base(a.filename)
	if name == "" || name == "." {
		name = "gomd"
	}
	return titleStyle.Render("gomd — " + name)
}

func (a *App) renderOutline() string {
	w := a.outlineWidth()
	h := a.outlineHeight()

	var lines []string
	headings := a.doc.Headings
	end := a.outlineOffset + h
	if end > len(headings) {
		end = len(headings)
	}

	for i := a.outlineOffset; i < end; i++ {
		heading := headings[i]
		indent := strings.Repeat("  ", heading.Level-1)
		prefix := strings.Repeat("#", heading.Level)

		var headingStyle lipgloss.Style
		switch heading.Level {
		case 1:
			headingStyle = lipgloss.NewStyle().Foreground(a.theme.Heading1).Bold(true)
		case 2:
			headingStyle = lipgloss.NewStyle().Foreground(a.theme.Heading2)
		case 3:
			headingStyle = lipgloss.NewStyle().Foreground(a.theme.Heading3)
		default:
			headingStyle = lipgloss.NewStyle().Foreground(a.theme.HeadingN)
		}

		text := indent + prefix + " " + heading.Text
		// Truncate if too wide
		maxLen := w - 2
		if len([]rune(text)) > maxLen {
			text = string([]rune(text)[:maxLen-1]) + "…"
		}

		rendered := headingStyle.Render(text)
		if i == a.selectedIdx {
			rendered = lipgloss.NewStyle().
				Background(a.theme.Selected).
				Width(w - 1).
				Render(text)
		}

		lines = append(lines, rendered)
	}

	// Pad to height
	for len(lines) < h {
		lines = append(lines, strings.Repeat(" ", w))
	}

	style := lipgloss.NewStyle().Width(w).MaxWidth(w)
	return style.Render(strings.Join(lines, "\n"))
}

func (a *App) renderContent() string {
	w := a.contentWidth()
	h := a.contentHeight()

	lines := a.contentLines
	start := a.contentOffset
	end := start + h
	if start > len(lines) {
		start = len(lines)
	}
	if end > len(lines) {
		end = len(lines)
	}

	visible := lines[start:end]

	// Apply syntax highlighting to code blocks and heading styling
	var rendered []string
	inCode := false
	var codeLang string
	var codeLines []string
	var codeStart int

	for i, line := range visible {
		trimmed := strings.TrimSpace(line)
		if !inCode && (strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")) {
			inCode = true
			codeLang = strings.TrimSpace(trimmed[3:])
			codeLines = nil
			codeStart = i
			rendered = append(rendered, lipgloss.NewStyle().Foreground(a.theme.HeadingN).Render(line))
			continue
		}
		if inCode {
			if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
				// Highlight accumulated code
				highlighted := highlightCode(strings.Join(codeLines, "\n"), codeLang, w)
				hLines := strings.Split(highlighted, "\n")
				// Replace already-appended fence line with styled version
				_ = codeStart
				rendered = append(rendered, hLines...)
				rendered = append(rendered, lipgloss.NewStyle().Foreground(a.theme.HeadingN).Render(line))
				inCode = false
				codeLines = nil
			} else {
				codeLines = append(codeLines, line)
			}
			continue
		}

		// Style headings
		if strings.HasPrefix(line, "#") {
			level := 0
			for level < len(line) && line[level] == '#' {
				level++
			}
			var s lipgloss.Style
			switch level {
			case 1:
				s = lipgloss.NewStyle().Foreground(a.theme.Heading1).Bold(true)
			case 2:
				s = lipgloss.NewStyle().Foreground(a.theme.Heading2).Bold(true)
			case 3:
				s = lipgloss.NewStyle().Foreground(a.theme.Heading3)
			default:
				s = lipgloss.NewStyle().Foreground(a.theme.HeadingN)
			}
			rendered = append(rendered, s.Render(line))
			continue
		}

		// Highlight search query
		if a.searchQuery != "" && len(a.searchMatches) > 0 {
			low := strings.ToLower(line)
			if idx := strings.Index(low, strings.ToLower(a.searchQuery)); idx >= 0 {
				before := line[:idx]
				match := line[idx : idx+len(a.searchQuery)]
				after := line[idx+len(a.searchQuery):]
				highlighted := before + lipgloss.NewStyle().Foreground(a.theme.Search).Bold(true).Render(match) + after
				rendered = append(rendered, highlighted)
				continue
			}
		}

		rendered = append(rendered, line)
	}

	// Flush remaining code block
	if inCode && len(codeLines) > 0 {
		highlighted := highlightCode(strings.Join(codeLines, "\n"), codeLang, w)
		rendered = append(rendered, strings.Split(highlighted, "\n")...)
	}

	// Pad to height
	for len(rendered) < h {
		rendered = append(rendered, "")
	}

	style := lipgloss.NewStyle().Width(w).MaxWidth(w)
	return style.Render(strings.Join(rendered, "\n"))
}

func (a *App) renderStatus() string {
	style := lipgloss.NewStyle().
		Background(a.theme.StatusBar).
		Foreground(a.theme.Foreground).
		Width(a.width).
		Padding(0, 1)

	var parts []string
	if a.selectedIdx < len(a.doc.Headings) {
		h := a.doc.Headings[a.selectedIdx]
		parts = append(parts, fmt.Sprintf("[%d/%d]", a.selectedIdx+1, len(a.doc.Headings)))
		parts = append(parts, strings.Repeat("#", h.Level)+" "+h.Text)
	}
	if a.statusMsg != "" {
		parts = append(parts, " — "+a.statusMsg)
	}

	msg := strings.Join(parts, " ")
	help := "q:quit j/k:nav /:search T:theme ?:help"

	// Pad between message and help
	pad := a.width - len(msg) - len(help) - 2
	if pad < 1 {
		pad = 1
	}

	return style.Render(msg + strings.Repeat(" ", pad) + help)
}

func (a *App) renderSearchBar() string {
	style := lipgloss.NewStyle().
		Background(a.theme.Search).
		Foreground(lipgloss.Color("#ffffff")).
		Width(a.width).
		Padding(0, 1)
	return style.Render("Search: " + a.searchQuery + "█")
}

func (a *App) renderHelp() string {
	helpText := `
  gomd — Keyboard Shortcuts

  NAVIGATION
    j / ↓        Move down in outline
    k / ↑        Move up in outline
    g            Jump to first heading
    G            Jump to last heading
    Enter / Space  Scroll content to selected heading

  CONTENT SCROLLING
    J            Scroll content down
    K            Scroll content up
    Ctrl+D / Ctrl+F  Page down
    Ctrl+U / Ctrl+B  Page up

  SEARCH
    /            Start search
    n            Next match
    N            Previous match
    Esc          Cancel search

  OTHER
    r            Reload file
    T            Open theme picker
    ?            Toggle this help
    q / Ctrl+C   Quit
`
	style := lipgloss.NewStyle().
		Background(a.theme.Background).
		Foreground(a.theme.Foreground).
		Width(a.width).
		Height(a.height).
		Padding(1, 2)

	return style.Render(helpText)
}

func (a *App) renderThemePicker() string {
	themeNames := []string{"OceanDark", "Nord", "Dracula", "Gruvbox", "TokyoNight"}
	var lines []string
	lines = append(lines, "  Select Theme (press number):")
	lines = append(lines, "")
	for i, name := range themeNames {
		marker := "  "
		if name == a.cfg.UI.Theme {
			marker = "→ "
		}
		lines = append(lines, fmt.Sprintf("  %s%d. %s", marker, i+1, name))
	}
	lines = append(lines, "")
	lines = append(lines, "  Esc / q to cancel")

	style := lipgloss.NewStyle().
		Background(a.theme.Background).
		Foreground(a.theme.Foreground).
		Width(a.width).
		Height(a.height).
		Padding(1, 2)

	return style.Render(strings.Join(lines, "\n"))
}

// highlightCode applies syntax highlighting to code using chroma.
func highlightCode(code, lang string, width int) string {
	if lang == "" {
		return code
	}

	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		return code
	}

	var sb strings.Builder
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code
	}

	if err := formatter.Format(&sb, style, iterator); err != nil {
		return code
	}

	return sb.String()
}

// Run starts the TUI application.
func Run(doc *parser.Document, filename, filePath string, cfg config.Config) error {
	app := NewApp(doc, filename, filePath, cfg)
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
