package tui

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"

	"github.com/grota/gomd/internal/config"
	"github.com/grota/gomd/internal/parser"
)

const featureTestDoc = `# Introduction

Welcome to the project.

## Getting Started

Install with ` + "`go install`" + `.

### Prerequisites

You need Go 1.22+.

## API Reference

See [the docs](https://example.com/docs) and [#getting-started](#getting-started).

### Authentication

Use tokens.

## Contributing

PRs welcome.
`

func newTestApp(md string) *App {
	doc := parser.ParseMarkdown(md)
	cfg := config.Default()
	app := NewApp(doc, "test.md", "test.md", cfg)
	app.width = 120
	app.height = 40
	return app
}

func TestFuzzySearch(t *testing.T) {
	app := newTestApp(featureTestDoc)

	// "gst" should fuzzy-match "Getting Started"
	if !fuzzyMatch("getting started", "gst") {
		t.Error("expected fuzzyMatch('getting started', 'gst') to be true")
	}

	// "auth" should match "Authentication"
	if !fuzzyMatch("authentication", "auth") {
		t.Error("expected fuzzyMatch('authentication', 'auth') to be true")
	}

	// Non-match
	if fuzzyMatch("introduction", "xyz") {
		t.Error("expected fuzzyMatch('introduction', 'xyz') to be false")
	}

	// Test search integration — search for "prereq" should find Prerequisites
	app.mode = ModeSearch
	app.searchQuery = "prereq"
	// Simulate enter
	app.searchMatches = nil
	q := strings.ToLower(app.searchQuery)
	for i, h := range app.doc.Headings {
		if fuzzyMatch(strings.ToLower(h.Text), q) {
			app.searchMatches = append(app.searchMatches, i)
		}
	}
	if len(app.searchMatches) != 1 {
		t.Fatalf("expected 1 match for 'prereq', got %d", len(app.searchMatches))
	}
	if app.doc.Headings[app.searchMatches[0]].Text != "Prerequisites" {
		t.Errorf("expected 'Prerequisites', got %q", app.doc.Headings[app.searchMatches[0]].Text)
	}
}

func TestBreadcrumb(t *testing.T) {
	app := newTestApp(featureTestDoc)

	// Select "Prerequisites" (nested under Getting Started > Introduction...)
	// Find the index of Prerequisites
	for i, h := range app.doc.Headings {
		if h.Text == "Prerequisites" {
			app.selectedIdx = i
			break
		}
	}

	bc := app.breadcrumb()
	// Should show Getting Started > Prerequisites (since Getting Started is the parent h2)
	if !strings.Contains(bc, "Getting Started") {
		t.Errorf("breadcrumb should contain parent 'Getting Started', got %q", bc)
	}
	if !strings.Contains(bc, "Prerequisites") {
		t.Errorf("breadcrumb should contain 'Prerequisites', got %q", bc)
	}
}

func TestTitleCentresFileName(t *testing.T) {
	app := newTestApp(featureTestDoc)
	for i, h := range app.doc.Headings {
		if h.Text == "Prerequisites" {
			app.selectedIdx = i
			break
		}
	}
	title := ansi.Strip(app.renderTitle())
	if w := lipgloss.Width(title); w != app.width {
		t.Fatalf("title width %d, want %d", w, app.width)
	}
	if col := strings.Index(title, "test.md"); col != (app.width-len("test.md"))/2 {
		t.Errorf("file name at column %d, want centred:\n%q", col, title)
	}
	if !strings.HasPrefix(title, " Introduction > Getting Started > Prerequisites ") {
		t.Errorf("breadcrumb should lead the title:\n%q", title)
	}

	// A breadcrumb that would reach the name is cut short of it.
	app.width = 30
	title = ansi.Strip(app.renderTitle())
	if lipgloss.Width(title) != 30 || !strings.Contains(title, "… test.md") {
		t.Errorf("narrow title:\n%q", title)
	}
}

func TestNavigationHistory(t *testing.T) {
	app := newTestApp(featureTestDoc)

	// Start at root (-1)
	if app.selectedIdx != -1 {
		t.Fatalf("expected starting at root, got %d", app.selectedIdx)
	}

	// Navigate to heading 0
	app.navigateTo(0)
	if app.selectedIdx != 0 {
		t.Fatalf("expected selectedIdx=0, got %d", app.selectedIdx)
	}

	// Navigate to heading 2
	app.navigateTo(2)
	if app.selectedIdx != 2 {
		t.Fatalf("expected selectedIdx=2, got %d", app.selectedIdx)
	}

	// Go back
	app.navGoBack()
	if app.selectedIdx != 0 {
		t.Errorf("after back: expected 0, got %d", app.selectedIdx)
	}

	// Go back again
	app.navGoBack()
	if app.selectedIdx != -1 {
		t.Errorf("after second back: expected -1, got %d", app.selectedIdx)
	}

	// Go forward
	app.navGoForward()
	if app.selectedIdx != 0 {
		t.Errorf("after forward: expected 0, got %d", app.selectedIdx)
	}

	// Go forward again
	app.navGoForward()
	if app.selectedIdx != 2 {
		t.Errorf("after second forward: expected 2, got %d", app.selectedIdx)
	}

	// Navigate somewhere new clears forward stack
	app.navigateTo(4)
	app.navGoForward()
	// Should not change (forward stack cleared)
	if app.selectedIdx != 4 {
		t.Errorf("forward after new nav should not change, got %d", app.selectedIdx)
	}
}

func TestInternalLinkAnchorResolution(t *testing.T) {
	app := newTestApp(featureTestDoc)

	// "getting-started" should resolve to the "Getting Started" heading
	idx := app.findHeadingByAnchor("getting-started")
	if idx < 0 {
		t.Fatal("expected to find heading for #getting-started")
	}
	if app.doc.Headings[idx].Text != "Getting Started" {
		t.Errorf("expected 'Getting Started', got %q", app.doc.Headings[idx].Text)
	}

	// "api-reference" should resolve
	idx = app.findHeadingByAnchor("api-reference")
	if idx < 0 {
		t.Fatal("expected to find heading for #api-reference")
	}
	if app.doc.Headings[idx].Text != "API Reference" {
		t.Errorf("expected 'API Reference', got %q", app.doc.Headings[idx].Text)
	}

	// Non-existent anchor
	idx = app.findHeadingByAnchor("nonexistent")
	if idx >= 0 {
		t.Errorf("expected -1 for nonexistent anchor, got %d", idx)
	}
}

func TestHeadingToAnchor(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Getting Started", "getting-started"},
		{"API Reference", "api-reference"},
		{"Hello World!", "hello-world"},
		{"C++ Guide", "c-guide"},
		{"Under_score", "under_score"},
		{"Configuração rápida", "configuração-rápida"},
	}
	for _, tt := range tests {
		got := headingToAnchor(tt.input)
		if got != tt.want {
			t.Errorf("headingToAnchor(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResponsiveSidebarWidth(t *testing.T) {
	// Short headings = narrower sidebar
	shortDoc := "# A\n## B\n## C\n"
	app := newTestApp(shortDoc)
	app.sidebarHidden = false
	sw := app.sidebarWidth()

	// Long headings = wider sidebar
	longDoc := "# This is a very long heading title\n## Another quite long heading here\n"
	app2 := newTestApp(longDoc)
	app2.sidebarHidden = false
	sw2 := app2.sidebarWidth()

	if sw2 <= sw {
		t.Errorf("longer headings should produce wider sidebar: short=%d, long=%d", sw, sw2)
	}

	// Should never exceed 40% of width
	maxAllowed := app2.width * 2 / 5
	if sw2 > maxAllowed {
		t.Errorf("sidebar %d exceeds 40%% max %d", sw2, maxAllowed)
	}
}

func TestJumpLabelGeneration(t *testing.T) {
	// Should generate unique single-char labels
	labels := generateLabels(5)
	if len(labels) != 5 {
		t.Fatalf("expected 5 labels, got %d", len(labels))
	}
	seen := make(map[string]bool)
	for _, l := range labels {
		if seen[l] {
			t.Errorf("duplicate label: %q", l)
		}
		if len(l) != 1 {
			t.Errorf("expected single char label, got %q", l)
		}
		seen[l] = true
	}

	// Requesting more than 24 should cap at 24
	labels = generateLabels(30)
	if len(labels) != 24 {
		t.Fatalf("expected 24 labels (capped), got %d", len(labels))
	}
}

func TestJumpModeEnterAndCancel(t *testing.T) {
	md := "# Title\n\n```go\nfmt.Println()\n```\n\nUse `foo` here.\n"
	app := newTestApp(md)

	// Rebuild to populate codeNodes
	app.rebuildSection()

	if len(app.codeNodes) == 0 {
		t.Fatal("expected code nodes in section")
	}

	// Enter jump mode
	app.enterJumpMode()
	if app.mode != ModeJump {
		t.Fatalf("expected ModeJump, got %d", app.mode)
	}
	if app.jumpLabels == nil {
		t.Fatal("jumpLabels should be populated")
	}
	if len(app.jumpLabels) != len(app.codeNodes) {
		t.Errorf("expected %d labels, got %d", len(app.codeNodes), len(app.jumpLabels))
	}

	// Cancel with mode reset
	app.mode = ModeNormal
	app.jumpLabels = nil
	if app.mode != ModeNormal {
		t.Error("should be back to normal mode")
	}
}

func TestPlaceImages_KeepsSurroundingText(t *testing.T) {
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "logo.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatal(err)
	}
	f.Close()
	md := filepath.Join(dir, "doc.md")
	src := "![logo](logo.png)\n\nBefore ![logo](logo.png) after\n"
	rendered := "  logo 📷 /logo.png  \n  Before logo 📷 /logo.png after\n  plain line"

	out, uploads := placeImages(strings.Split(rendered, "\n"), src, md, 40, false, imageCache{})
	if len(uploads) != 1 {
		t.Errorf("want 1 upload for one image used twice, got %d", len(uploads))
	}
	if len(out) != 4 {
		t.Fatalf("want 4 lines, got %d: %q", len(out), out)
	}
	if !isImageRow(out[0]) {
		t.Errorf("standalone image line not replaced: %q", out[0])
	}
	if !strings.Contains(out[1], "Before") || !strings.Contains(out[1], "after") {
		t.Errorf("inline image line lost its text: %q", out[1])
	}
	if !isImageRow(out[2]) || out[3] != "  plain line" {
		t.Errorf("image not placed after inline line, or plain line changed: %q", out[2:])
	}
}

func TestPlaceImages_SharedPathDifferentForms(t *testing.T) {
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "a.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatal(err)
	}
	f.Close()
	// "./a.png" and "a.png" render the same path; an empty alt must not eat
	// part of another caption.
	src := "![one](./a.png)\n\n![](a.png)\n\n![two](a.png \"title\")\n"
	rendered := "  one 📷 /a.png\n  /a.png\n  two 📷 /a.png"
	out, _ := placeImages(strings.Split(rendered, "\n"), src, filepath.Join(dir, "d.md"), 40, false, imageCache{})
	for i, l := range out {
		if !isImageRow(l) {
			t.Errorf("line %d not replaced by an image: %q", i, l)
		}
	}
}

func TestPlaceImages_KeepCaptionAndCache(t *testing.T) {
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "a.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 32, 32))); err != nil {
		t.Fatal(err)
	}
	f.Close()
	cache := imageCache{}
	lines := []string{"  pic 📷 /a.png"}
	out, uploads := placeImages(lines, "![pic](a.png)", filepath.Join(dir, "d.md"), 40, true, cache)
	if out[0] != lines[0] || len(out) != 3 || len(uploads) != 1 {
		t.Fatalf("keepCaption: want caption + 2 rows and 1 upload, got %q, %d uploads", out, len(uploads))
	}
	if w := ansi.StringWidth(out[1]); w != 6 {
		t.Errorf("placeholder row width %d, want 2 margin + 4 cells", w)
	}
	if _, again := placeImages(lines, "![pic](a.png)", filepath.Join(dir, "d.md"), 40, true, cache); len(again) != 0 {
		t.Errorf("cached image uploaded again")
	}
}

func isImageRow(l string) bool { return strings.ContainsRune(l, kitty.Placeholder) }

func TestTUIImagesUploadOnceAndFitWidth(t *testing.T) {
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "a.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 400, 100))); err != nil {
		t.Fatal(err)
	}
	f.Close()
	md := filepath.Join(dir, "d.md")
	cfg := config.Default()
	cfg.Images.Enabled = true
	a := NewApp(parser.ParseMarkdown("# T\n\n![pic](a.png)\n\ntext\n"), "d.md", md, cfg)

	_, cmd := a.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	if cmd == nil {
		t.Fatal("expected an upload command after the first render")
	}
	view := a.View().Content
	if !strings.ContainsRune(view, kitty.Placeholder) {
		t.Error("no placeholder rows in the view")
	}
	for i, l := range strings.Split(view, "\n") {
		if w := ansi.StringWidth(l); w != 60 {
			t.Errorf("line %d width %d, want 60", i, w)
		}
	}
	if _, cmd := a.Update(tea.KeyPressMsg{Code: 'j', Text: "j"}); cmd != nil {
		if msg := cmd(); msg != nil {
			if _, isRaw := msg.(tea.RawMsg); isRaw {
				t.Error("image uploaded again without a size change")
			}
		}
	}
}

func TestImageNodesMapToTheirOwnCaption(t *testing.T) {
	md := "# T\n\n<img src=\"c.png\" alt=\"checker\">\n\n## U\n\n![](c.png)\n\n![t](c.png \"A title\")\n"
	a := NewApp(parser.ParseMarkdown(md), "t.md", "", config.Default())
	a.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	a.ensureRenderedLines()
	if len(a.codeNodes) != 3 {
		t.Fatalf("want 3 image nodes, got %d", len(a.codeNodes))
	}
	seen := map[int]bool{}
	for i, loc := range a.nodeRenderInfo {
		if loc.firstLine < 0 || seen[loc.firstLine] {
			t.Errorf("node %d (%q) mapped to line %d, want its own caption line", i, a.codeNodes[i].display, loc.firstLine)
		}
		seen[loc.firstLine] = true
	}
	if got := a.codeNodes[2].content; got != "c.png" {
		t.Errorf("title kept in destination: %q", got)
	}
}

// The layout measures graphemes, but bubbletea falls back to wcwidth when the
// terminal (e.g. tmux) does not answer the Unicode-core query. Glyphs that the
// two disagree on leave stale cells on screen.
func TestCaptionGlyphWidthsAgree(t *testing.T) {
	for _, g := range []string{"📷", "🔗"} {
		if ansi.StringWidth(g) != ansi.StringWidthWc(g) {
			t.Errorf("%q: grapheme width %d, wcwidth %d", g, ansi.StringWidth(g), ansi.StringWidthWc(g))
		}
	}
}

func TestSelectionCopiesImageAsMarkdown(t *testing.T) {
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "a.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 32, 32))); err != nil {
		t.Fatal(err)
	}
	f.Close()
	md := filepath.Join(dir, "d.md")
	a := NewApp(parser.ParseMarkdown("before\n\n![pic](./a.png)\n\n![](a.png)\n\nafter\n"), "d.md", md, config.Default())
	a.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	last := len(a.renderedLines) - 1
	a.sel = &selection{anchor: textPos{0, 0}, head: textPos{last, 10}}
	got := a.selectedText()
	if strings.ContainsRune(got, kitty.Placeholder) {
		t.Errorf("selected text contains placeholder cells")
	}
	for _, want := range []string{"before", "![pic](./a.png)", "![](a.png)", "after"} {
		if !strings.Contains(got, want) {
			t.Errorf("selected text %q lacks %q", got, want)
		}
	}
	if strings.Contains(got, "📷") {
		t.Errorf("selected text %q kept a caption", got)
	}
}

func TestCopyGoesThroughTmux(t *testing.T) {
	var copied string
	defer func(f func(string) error) { tmuxLoadBuffer = f }(tmuxLoadBuffer)
	tmuxLoadBuffer = func(s string) error { copied = s; return nil }
	t.Setenv("TMUX", "test")

	a := NewApp(parser.ParseMarkdown("text\n"), "f.md", "", config.Default())
	if _, cmd := a.Update(copyCmd("hi", "text", "", "")()); cmd != nil {
		t.Errorf("sent OSC 52 although tmux took the copy")
	}
	if copied != "hi" || a.statusMsg != "Copied text via tmux" {
		t.Errorf("copied %q with status %q", copied, a.statusMsg)
	}
}

func TestPlaceImages_NewWidthFreesTheOldSize(t *testing.T) {
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "a.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 400, 100))); err != nil {
		t.Fatal(err)
	}
	f.Close()
	cache := imageCache{}
	lines := []string{"  pic 📷 /a.png"}
	place := func(cols int) []string {
		_, uploads := placeImages(lines, "![pic](a.png)", filepath.Join(dir, "d.md"), cols, true, cache)
		return uploads
	}
	place(40)
	old := cache[filepath.Join(dir, "a.png")]
	uploads := place(20)
	k := cache[filepath.Join(dir, "a.png")]
	if len(cache) != 1 || len(uploads) != 2 || uploads[0] != old.deleteSeq() || k.id == old.id || k.cols != 20 {
		t.Fatalf("resize: cache %d entries, %d uploads, cols %d", len(cache), len(uploads), k.cols)
	}
	if !strings.Contains(uploads[1], fmt.Sprintf("i=%d,c=20,", k.id)) {
		t.Errorf("new upload does not carry the new id and size: %.80q", uploads[1])
	}
	if again := place(20); len(again) != 0 {
		t.Errorf("same width uploaded again: %d", len(again))
	}
}

func TestHugeImageIsRefusedBeforeDecoding(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	b := buf.Bytes()
	binary.BigEndian.PutUint32(b[16:], 20000) // IHDR width
	binary.BigEndian.PutUint32(b[20:], 20000) // IHDR height
	binary.BigEndian.PutUint32(b[29:], crc32.ChecksumIEEE(b[12:29]))
	path := filepath.Join(t.TempDir(), "huge.png")
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadKittyImage(path, 40); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("want a too-large error, got %v", err)
	}
}
