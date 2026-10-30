// Package tui provides the interactive terminal user interface for gomd.
package tui

import (
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/fsnotify/fsnotify"

	"github.com/grota/gomd/internal/config"
	"github.com/grota/gomd/internal/parser"
)

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
	line     int // rendered line index
	colStart int // display column start
	colEnd   int // display column end
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
	display   string // for links: the visible text glamour renders (e.g. link text, "alt 📷 path")
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

	// Theme selection: the light/dark pair, the system appearance it follows
	// (mode 2031), and the picker while it is open.
	themeSetting ThemeSetting
	dark         bool
	picker       *themePicker
	themeLists   [2][]pickerEntry

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

	// Images (Kitty Unicode placeholders)
	images         imageCache // uploaded images, freed on exit
	pendingUploads []string   // uploads produced by the last render, sent by Update

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
	sel       *selection // mouse selection in the content pane, nil when none
	srcMap    *sourceMap // renderedLines to source, built on first use
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
		focus:         FocusContent,
		selectedIdx:   -1, // start on the root (Document) node
		sidebarHidden: cfg.UI.SidebarHidden,
		images:        imageCache{},
		dark:          true,
	}
	var err error
	if a.themeSetting, err = ParseThemeSetting(cfg.UI.Theme); err != nil {
		a.themeSetting, _ = ParseThemeSetting("")
	}
	a.applyTheme()
	a.rebuildSection()
	return a
}

// ─────────────────────────────────────────────
// Bubbletea lifecycle
// ─────────────────────────────────────────────

type fileReloadMsg struct{ doc *parser.Document }
type editorDoneMsg struct{ err error }

// openURL launches the configured opener on url and returns a status message.
func (a *App) openURL(url string) string {
	if strings.HasPrefix(url, "-") {
		return "Refusing to open: " + url // would be parsed as an option
	}
	args := strings.Fields(a.cfg.UI.Opener)
	if len(args) == 0 {
		args = []string{"xdg-open"}
		if runtime.GOOS == "darwin" {
			args = []string{"open"}
		}
	}
	cmd := exec.Command(args[0], append(args[1:], url)...)
	if err := cmd.Start(); err != nil {
		return "Open failed: " + err.Error()
	}
	go cmd.Wait() // reap the child so it does not linger as a zombie
	return "Opened: " + url
}

// openEditorCmd suspends the TUI, opens $EDITOR on filePath, then signals
// done. $EDITOR may include arguments, e.g. "code --wait".
func openEditorCmd(filePath string) tea.Cmd {
	args := strings.Fields(os.Getenv("EDITOR"))
	if len(args) == 0 {
		args = []string{"vi"}
	}
	c := exec.Command(args[0], append(args[1:], filePath)...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorDoneMsg{err}
	})
}

// waitForFileChange waits for the next write to filePath. The watcher is on
// the parent directory because editors that save atomically replace the file,
// which would silently drop a watch on the file itself.
func waitForFileChange(watcher *fsnotify.Watcher, filePath string) tea.Cmd {
	target := filepath.Clean(filePath)
	return func() tea.Msg {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return nil
				}
				if filepath.Clean(event.Name) != target || event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
					continue
				}
				data, err := os.ReadFile(filePath)
				if err != nil {
					continue
				}
				return fileReloadMsg{parser.ParseMarkdown(string(data))}
			case _, ok := <-watcher.Errors:
				if !ok {
					return nil
				}
			}
		}
	}
}

func (a *App) startWatcher() tea.Cmd {
	if a.filepath == "" {
		return nil
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil
	}
	if err := watcher.Add(filepath.Dir(a.filepath)); err != nil {
		watcher.Close()
		return nil
	}
	a.watcher = watcher
	return waitForFileChange(watcher, a.filepath)
}

// reload swaps in a freshly parsed document, keeping the selected heading when
// its text still exists and dropping indexes that refer to the old document.
// The scroll position survives when the same section is shown again.
func (a *App) reload(doc *parser.Document, status string) {
	prevHeading := ""
	if a.selectedIdx >= 0 && a.selectedIdx < len(a.doc.Headings) {
		prevHeading = a.doc.Headings[a.selectedIdx].Text
	}
	wasRoot, offset := a.selectedIdx < 0, a.contentOffset
	a.doc = doc
	a.selectedIdx = -1
	for i, h := range doc.Headings {
		if prevHeading != "" && h.Text == prevHeading {
			a.selectedIdx = i
			break
		}
	}
	a.navHistory, a.navFuture = nil, nil
	a.searchMatches, a.searchIdx = nil, 0
	a.rebuildSection()
	if wasRoot == (a.selectedIdx < 0) {
		a.contentOffset = offset // clamped when the section is rendered
	}
	a.scrollOutlineToSelected()
	a.statusMsg = status
}

// reloadFromDisk re-reads the file, reporting failures in the status bar.
func (a *App) reloadFromDisk(status string) {
	if a.filepath == "" {
		a.statusMsg = "Nothing to reload (stdin)"
		return
	}
	data, err := os.ReadFile(a.filepath)
	if err != nil {
		a.statusMsg = "Reload failed: " + err.Error()
		return
	}
	a.reload(parser.ParseMarkdown(string(data)), status)
}

func (a *App) Init() tea.Cmd {
	// Follow the system appearance: report it now and on every change.
	return tea.Batch(a.startWatcher(), tea.Raw(ansi.SetModeLightDark+ansi.RequestLightDarkReport))
}

// setTheme shows t and drops everything rendered with the previous theme.
func (a *App) setTheme(t Theme) {
	a.theme = t
	a.glamourRenderer = nil
	a.renderedLines = nil
	a.nodeRenderInfo = nil
}

// applyTheme shows the configured theme for the current system mode.
func (a *App) applyTheme() {
	t, err := ThemeFor(a.cfg, a.dark)
	if err != nil {
		a.statusMsg = "Theme: " + err.Error()
	}
	a.setTheme(t)
}

// setDark follows a system appearance report; while the picker is open the
// switch waits until it closes.
func (a *App) setDark(dark bool) {
	if a.dark == dark {
		return
	}
	a.dark = dark
	if a.picker != nil {
		a.picker.switched = true
		return
	}
	a.applyTheme()
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m, cmd := a.update(msg)
	// Render here rather than lazily in View, because only Update can send
	// the image uploads (View cannot return commands).
	if a.cfg.Images.Enabled && a.width > 0 {
		a.ensureRenderedLines()
		if len(a.pendingUploads) > 0 {
			cmd = tea.Batch(cmd, tea.Raw(strings.Join(a.pendingUploads, "")))
			a.pendingUploads = nil
		}
	}
	return m, cmd
}

func (a *App) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

	case fileReloadMsg:
		a.reload(msg.doc, "Reloaded")
		return a, waitForFileChange(a.watcher, a.filepath)

	case uv.DarkColorSchemeEvent:
		a.setDark(true)
	case uv.LightColorSchemeEvent:
		a.setDark(false)
	case themesDownloadedMsg:
		a.handleThemesDownloaded(msg)

	case editorDoneMsg:
		a.reloadFromDisk("Reloaded after edit")
		if msg.err != nil {
			a.statusMsg = "Editor failed: " + msg.err.Error()
		}
		return a, nil

	case tea.KeyPressMsg:
		return a.handleKey(msg)

	case tea.MouseWheelMsg:
		return a.handleMouseWheel(msg)
	case tea.MouseClickMsg:
		return a.handleMouseClick(msg)
	case tea.MouseMotionMsg:
		return a.handleMouseMotion(msg)
	case tea.MouseReleaseMsg:
		return a.handleMouseRelease(msg)
	case clipboardDoneMsg:
		return a.handleClipboardDone(msg)
	}

	return a, nil
}

// ─────────────────────────────────────────────
// Entry point
// ─────────────────────────────────────────────

// Run starts the TUI; dark is the system appearance found at startup.
func Run(doc *parser.Document, filename, filePath string, cfg config.Config, dark bool) error {
	app := NewApp(doc, filename, filePath, cfg)
	app.dark = dark
	app.applyTheme()
	p := tea.NewProgram(app)
	_, err := p.Run()
	os.Stdout.WriteString(ansi.ResetModeLightDark)
	for _, k := range app.images {
		os.Stdout.WriteString(k.deleteSeq())
	}
	return err
}
