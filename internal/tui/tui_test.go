package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/fsnotify/fsnotify"

	"github.com/grota/gomd/internal/config"
	"github.com/grota/gomd/internal/parser"
)

// ─────────────────────────────────────────
// helpers

func renderStripped(t *testing.T, src string, width int) (rendered, stripped []string) {
	t.Helper()
	r, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		t.Fatalf("glamour renderer: %v", err)
	}
	out, err := r.Render(src)
	if err != nil {
		t.Fatalf("glamour render: %v", err)
	}
	rendered = strings.Split(strings.TrimRight(out, "\n"), "\n")
	stripped = make([]string, len(rendered))
	for i, l := range rendered {
		stripped[i] = ansi.Strip(l)
	}
	return
}

// ─────────────────────────────────────────
// extractCodeNodes

func TestExtractCodeNodes_FencedBlock(t *testing.T) {
	lines := []string{
		"## Title",
		"",
		"```go",
		"fmt.Println(\"hello\")",
		"```",
		"",
		"Some prose.",
	}
	nodes := sectionNodes(lines)
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node (fenced), got %d", len(nodes))
	}
	b := nodes[0]
	if b.lang != "go" || b.inline {
		t.Errorf("unexpected fenced node: %+v", b)
	}
	if !strings.Contains(b.content, "fmt.Println") {
		t.Errorf("fenced content missing body: %q", b.content)
	}
}

func TestExtractCodeNodes_InlineCode(t *testing.T) {
	lines := []string{
		"Use `useState` and `useEffect` in your component.",
	}
	nodes := sectionNodes(lines)
	if len(nodes) != 2 {
		t.Fatalf("expected 2 inline nodes, got %d", len(nodes))
	}
	if nodes[0].content != "useState" || !nodes[0].inline {
		t.Errorf("unexpected node[0]: %+v", nodes[0])
	}
	if nodes[1].content != "useEffect" || !nodes[1].inline {
		t.Errorf("unexpected node[1]: %+v", nodes[1])
	}
}

func TestExtractCodeNodes_HeadingWithBacktick(t *testing.T) {
	// Headings are no longer extracted as nodes, but inline code within them is
	lines := []string{"## `useCallback` hook"}
	nodes := sectionNodes(lines)
	if len(nodes) != 1 {
		t.Fatalf("expected 1 inline code node from heading, got %d", len(nodes))
	}
	if nodes[0].content != "useCallback" || nodes[0].kind != nodeInlineCode {
		t.Errorf("unexpected node: %+v", nodes[0])
	}
}

func TestExtractCodeNodes_NoNodes(t *testing.T) {
	lines := []string{"Just plain prose.", "", "No code here."}
	nodes := sectionNodes(lines)
	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(nodes))
	}
}

// ─────────────────────────────────────────
// highlightSpanInLine

func TestHighlightSpanInLine_Basic(t *testing.T) {
	line := "hello world foo"
	// Highlight "world" (cols 6-11)
	theme := themes["OceanDark"]
	style := lipgloss.NewStyle().Background(theme.NodeSel).Foreground(theme.Background).Bold(true)
	result := highlightSpanInLine(line, 6, 11, style)
	plain := ansi.Strip(result)
	if plain != line {
		t.Errorf("stripped result changed text: got %q, want %q", plain, line)
	}
	if !strings.Contains(result, "world") {
		t.Errorf("result does not contain 'world': %q", result)
	}
	// Result must reconstruct the full original text when stripped.
	if plain != "hello world foo" {
		t.Errorf("stripped result wrong: %q", plain)
	}
}

func TestHighlightSpanInLine_BulletLine(t *testing.T) {
	// Simulate a glamour-rendered bullet line containing a multibyte bullet char.
	// "• foo bar" — bullet is 3 bytes, 1 display col.
	// Span "bar" starts at display col 6.
	line := "• foo bar"
	theme := themes["OceanDark"]
	style := lipgloss.NewStyle().Background(theme.NodeSel).Foreground(theme.Background).Bold(true)
	result := highlightSpanInLine(line, 6, 9, style)
	plain := ansi.Strip(result)
	if plain != line {
		t.Errorf("stripped result changed: %q != %q", plain, line)
	}
}

func TestHighlightSpanInLine_NoOpOnInvalidRange(t *testing.T) {
	line := "some text"
	theme := themes["OceanDark"]
	style := lipgloss.NewStyle().Background(theme.NodeSel).Foreground(theme.Background).Bold(true)
	// colStart < 0
	if got := highlightSpanInLine(line, -1, 5, style); got != line {
		t.Errorf("negative colStart should return line unchanged")
	}
	// colEnd <= colStart
	if got := highlightSpanInLine(line, 5, 5, style); got != line {
		t.Errorf("equal col should return line unchanged")
	}
}

// ─────────────────────────────────────────
// mapNodesToRenderedLines

const testMD = `## MyHeading

Some prose with ` + "`useState`" + ` and ` + "`useEffect`" + ` inline.

` + "```go" + `
func main() {
    fmt.Println("hello")
}
` + "```" + `

More text after.
`

func TestMapNodes_InlineCodeFound(t *testing.T) {
	lines := strings.Split(strings.TrimRight(testMD, "\n"), "\n")
	nodes := sectionNodes(lines)
	_, stripped := renderStripped(t, testMD, 80)
	info := mapNodesToRenderedLines(nodes, stripped)

	// Find useState and useEffect nodes
	var useStateIdx, useEffectIdx int = -1, -1
	for i, n := range nodes {
		if n.inline && n.content == "useState" {
			useStateIdx = i
		}
		if n.inline && n.content == "useEffect" {
			useEffectIdx = i
		}
	}
	if useStateIdx < 0 || useEffectIdx < 0 {
		t.Fatal("useState or useEffect node not found in extractCodeNodes output")
	}
	for _, idx := range []int{useStateIdx, useEffectIdx} {
		loc := info[idx]
		if loc.firstLine < 0 {
			t.Errorf("node[%d] (%q) not found in rendered", idx, nodes[idx].content)
			continue
		}
		lineText := stripped[loc.firstLine]
		if !strings.Contains(lineText, nodes[idx].content) {
			t.Errorf("node[%d] firstLine=%d does not contain %q: %q",
				idx, loc.firstLine, nodes[idx].content, lineText)
		}
	}
}

func TestMapNodes_FencedBlockBounds(t *testing.T) {
	lines := strings.Split(strings.TrimRight(testMD, "\n"), "\n")
	nodes := sectionNodes(lines)
	_, stripped := renderStripped(t, testMD, 80)
	info := mapNodesToRenderedLines(nodes, stripped)

	// Find the fenced block node
	var fencedIdx int = -1
	for i, n := range nodes {
		if !n.inline && n.lang == "go" {
			fencedIdx = i
			break
		}
	}
	if fencedIdx < 0 {
		t.Fatal("fenced block node not found")
	}
	loc := info[fencedIdx]
	if loc.firstLine < 0 {
		t.Fatal("fenced block not found in rendered")
	}
	if loc.lastLine <= loc.firstLine {
		t.Errorf("fenced block lastLine=%d should be > firstLine=%d", loc.lastLine, loc.firstLine)
	}
	// lastLine must not point to prose after the block
	lastText := strings.TrimSpace(stripped[loc.lastLine])
	if strings.HasPrefix(lastText, "More text") {
		t.Errorf("fenced block lastLine=%d points to prose after block: %q", loc.lastLine, lastText)
	}
	// lastLine must be 4-space indented code or blank (inside the block)
	if !strings.HasPrefix(stripped[loc.lastLine], "    ") && strings.TrimSpace(stripped[loc.lastLine]) != "" {
		t.Errorf("fenced block lastLine=%d not code-indented: %q", loc.lastLine, stripped[loc.lastLine])
	}
}

func TestMapNodes_InlineSpanDisplayCols(t *testing.T) {
	// Bullet lines have multibyte chars; display cols must differ from byte offsets.
	src := "## H\n\n• Use `useState` in your component.\n"
	lines := strings.Split(strings.TrimRight(src, "\n"), "\n")
	nodes := sectionNodes(lines)
	_, stripped := renderStripped(t, src, 80)
	info := mapNodesToRenderedLines(nodes, stripped)

	var inlineIdx int = -1
	for i, n := range nodes {
		if n.inline && n.content == "useState" {
			inlineIdx = i
			break
		}
	}
	if inlineIdx < 0 {
		t.Fatal("useState node not found")
	}
	loc := info[inlineIdx]
	if loc.firstLine < 0 {
		t.Fatal("useState not found in rendered output")
	}
	// spanColByte and spanCol must differ (bullet is 3 bytes, 1 display col)
	lineText := stripped[loc.firstLine]
	if !strings.Contains(lineText, "•") {
		t.Skip("bullet not present in rendered line — cannot test byte/display divergence")
	}
	if loc.spanColByte == loc.spanCol {
		t.Errorf("expected spanColByte (%d) != spanCol (%d) due to multibyte bullet",
			loc.spanColByte, loc.spanCol)
	}
}

// TestMapNodes_InlineNotInsideFencedBlock is a regression test for the bug where an
// inline node whose text also appears inside a preceding fenced block was mapped to
// the in-block occurrence instead of the later prose occurrence.
//
// Setup: a fenced block that contains "SlowList", followed by prose that also
// contains `SlowList` as an inline code span. The inline node must map to the
// prose line, not to the line inside the block.
func TestMapNodes_InlineNotInsideFencedBlock(t *testing.T) {
	src := "## Title\n\nPreamble.\n\n" +
		"```jsx\n" +
		"function App() {\n" +
		"  return <SlowList />;\n" +
		"}\n" +
		"```\n\n" +
		"Use `SlowList` carefully.\n"

	lines := strings.Split(strings.TrimRight(src, "\n"), "\n")
	nodes := sectionNodes(lines)
	_, stripped := renderStripped(t, src, 80)
	info := mapNodesToRenderedLines(nodes, stripped)

	// Find fenced block and SlowList inline node indices.
	fencedIdx, inlineIdx := -1, -1
	for i, n := range nodes {
		if !n.inline && n.lang == "jsx" {
			fencedIdx = i
		}
		if n.inline && n.content == "SlowList" {
			inlineIdx = i
		}
	}
	if fencedIdx < 0 || inlineIdx < 0 {
		t.Fatalf("could not find expected nodes: fencedIdx=%d inlineIdx=%d", fencedIdx, inlineIdx)
	}

	fencedLoc := info[fencedIdx]
	inlineLoc := info[inlineIdx]

	if fencedLoc.firstLine < 0 {
		t.Fatal("fenced block not found in rendered output")
	}
	if inlineLoc.firstLine < 0 {
		t.Fatal("inline SlowList node not found in rendered output")
	}

	// The inline node must be outside the fenced block's line range.
	if inlineLoc.firstLine >= fencedLoc.firstLine && inlineLoc.firstLine <= fencedLoc.lastLine {
		t.Errorf("inline SlowList mapped to line %d which is inside fenced block [%d..%d]",
			inlineLoc.firstLine, fencedLoc.firstLine, fencedLoc.lastLine)
	}

	// The prose line must contain "SlowList" as a plain word (not inside the block).
	proseLine := stripped[inlineLoc.firstLine]
	if !strings.Contains(proseLine, "SlowList") {
		t.Errorf("inline SlowList mapped to line %d which does not contain 'SlowList': %q",
			inlineLoc.firstLine, proseLine)
	}
	// The prose line should have 2-space indent (glamour prose), not 4-space (code block).
	if strings.HasPrefix(proseLine, "    ") {
		t.Errorf("inline SlowList mapped to a 4-space-indented line (inside block?): %q", proseLine)
	}
}

// TestMapNodes_FullREADME verifies that all inline code spans in a real-world
// document (a frozen copy of the project README) map to their rendered positions.
func TestMapNodes_FullREADME(t *testing.T) {
	content, err := os.ReadFile("testdata/readme.md")
	if err != nil {
		t.Fatal(err)
	}
	src := string(content)
	lines := strings.Split(src, "\n")
	nodes := sectionNodes(lines)

	rendered, stripped := renderStripped(t, src, 140)
	locs := mapNodesToRenderedLines(nodes, rendered)

	var failures int
	for i, n := range nodes {
		if !n.inline {
			continue
		}
		// Single-char inline codes in headings can't be reliably mapped (known limitation)
		if len(n.content) <= 1 {
			continue
		}
		// Link URLs are rendered differently by glamour (truncated, reformatted)
		// and cannot be reliably mapped back to their raw form.
		if strings.HasPrefix(n.content, "http://") || strings.HasPrefix(n.content, "https://") {
			continue
		}
		// Image src paths are rendered differently by glamour (./prefix stripped, etc.)
		if strings.HasPrefix(n.content, "./") || strings.HasSuffix(n.content, ".png") ||
			strings.HasSuffix(n.content, ".jpg") || strings.HasSuffix(n.content, ".gif") ||
			strings.HasSuffix(n.content, ".svg") {
			continue
		}
		loc := locs[i]
		if loc.firstLine == -1 {
			t.Errorf("node %d (content=%q, srcLine=%d) NOT MAPPED", i, n.content, n.startLine)
			failures++
			continue
		}
		line := stripped[loc.firstLine]
		if loc.spanColEndByte > len(line) {
			t.Errorf("node %d (content=%q, srcLine=%d) spanColEndByte %d > line len %d",
				i, n.content, n.startLine, loc.spanColEndByte, len(line))
			failures++
			continue
		}
		actual := line[loc.spanColByte:loc.spanColEndByte]
		if actual != n.content {
			t.Errorf("node %d (content=%q, srcLine=%d) -> rendered line %d col %d-%d has %q",
				i, n.content, n.startLine, loc.firstLine, loc.spanColByte, loc.spanColEndByte, actual)
			failures++
		}
	}
	if failures > 0 {
		t.Logf("%d failures out of %d nodes", failures, len(nodes))
	}
}

func TestViewTinyTerminalDoesNotPanic(t *testing.T) {
	doc := parser.ParseMarkdown("# A\n\ntext\n\n## B\n\n```go\nx\n```\n")
	for _, hidden := range []bool{true, false} {
		for w := 1; w <= 12; w++ {
			for h := 1; h <= 8; h++ {
				cfg := config.Default()
				cfg.UI.SidebarHidden = hidden
				a := NewApp(doc, "f.md", "", cfg)
				a.Update(tea.WindowSizeMsg{Width: w, Height: h})
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Errorf("sidebarHidden=%v %dx%d: panic: %v", hidden, w, h, r)
						}
					}()
					a.View()
				}()
			}
		}
	}
}

func TestReloadDroppingAllHeadings(t *testing.T) {
	a := NewApp(parser.ParseMarkdown("# A\n\n## B\n\ntext\n"), "f.md", "", config.Default())
	a.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	a.navigateTo(1)
	a.reload(parser.ParseMarkdown("no headings any more\n"), "Reloaded")
	if a.selectedIdx != -1 || len(a.navHistory) != 0 {
		t.Fatalf("stale state after reload: selectedIdx=%d navHistory=%v", a.selectedIdx, a.navHistory)
	}
	a.View()
	a.navGoBack()
	a.View()
}

func TestSidebarJumpsUseDisplayRows(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 40; i++ {
		sb.WriteString("# H" + string(rune('a'+i%26)) + "\n\n")
	}
	cfg := config.Default()
	cfg.UI.SidebarHidden = false
	a := NewApp(parser.ParseMarkdown(sb.String()), "f.md", "", cfg)
	a.Update(tea.WindowSizeMsg{Width: 80, Height: 16})
	h := a.outlineHeight()

	a.outlineOffset = 0
	a.sidebarJumpHigh()
	if a.selectedIdx != -1 {
		t.Errorf("H at top: want root (-1), got %d", a.selectedIdx)
	}
	a.sidebarJumpLow()
	if a.selectedIdx != h-2 {
		t.Errorf("L: want last visible heading %d, got %d", h-2, a.selectedIdx)
	}
	a.selectedIdx = len(a.doc.Headings) - 1
	a.sidebarViewBottom()
	if last := a.displayIdx(); last < a.outlineOffset || last >= a.outlineOffset+h {
		t.Errorf("zb: last heading row %d not in window [%d,%d)", last, a.outlineOffset, a.outlineOffset+h)
	}
}

func TestApplyJumpLabelsAfterWideChar(t *testing.T) {
	a := NewApp(parser.ParseMarkdown("x\n"), "f.md", "", config.Default())
	a.codeNodes = []codeNode{{inline: true}}
	a.nodeRenderInfo = []nodeRenderLoc{{firstLine: 0, lastLine: 0, spanCol: 6, spanColEnd: 7}}
	a.jumpLabels = map[string]int{"q": 0}
	got := ansi.Strip(a.applyJumpLabels([]string{"🔗 ab x cd"}, 40)[0])
	if want := "🔗 ab qx cd"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestViewLinesFillTerminalWidth(t *testing.T) {
	src, err := os.ReadFile("testdata/readme.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, hidden := range []bool{true, false} {
		cfg := config.Default()
		cfg.UI.SidebarHidden = hidden
		a := NewApp(parser.ParseMarkdown(string(src)), "r.md", "", cfg)
		a.Update(tea.WindowSizeMsg{Width: 120, Height: 35})
		for i, l := range strings.Split(a.View().Content, "\n") {
			if w := ansi.StringWidth(l); w != 120 {
				t.Errorf("sidebarHidden=%v line %d: width %d, want 120: %q", hidden, i, w, ansi.Strip(l))
			}
		}
	}
}

func TestContentBorderWithSidebar(t *testing.T) {
	cfg := config.Default()
	cfg.UI.ContentBorder = config.ContentBorderWithSidebar
	a := NewApp(parser.ParseMarkdown("# A\n\ntext\n"), "f.md", "", cfg)
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 10})

	view := ansi.Strip(a.View().Content)
	lines := strings.Split(view, "\n")
	if len(lines) != 10 || strings.ContainsAny(view, "┌│") {
		t.Errorf("sidebar hidden: want 10 borderless lines, got %d:\n%s", len(lines), view)
	}
	for i, l := range lines[1 : len(lines)-1] {
		if w := lipgloss.Width(l); w != 120 {
			t.Errorf("line %d is %d columns wide, want 120", i+1, w)
		}
	}

	a.sidebarHidden = false
	if view := ansi.Strip(a.View().Content); strings.Count(view, "┌") != 2 {
		t.Errorf("sidebar shown: want both panes boxed:\n%s", view)
	}
}

func TestHighlightSpansManyMatchesStayLinear(t *testing.T) {
	line := "\x1b[38;5;252mThe quick brown fox jumps over the lazy dog, then everyone sees the event\x1b[0m"
	a := &App{renderedLines: []string{line}, contentSearchQuery: "e"}
	a.findContentSearchMatches()
	out := a.applyContentSearchHighlights(a.renderedLines, 0, 1)[0]
	if got := ansi.Strip(out); got != ansi.Strip(line) {
		t.Errorf("text changed: %q", got)
	}
	if len(out) > 40*len(line) {
		t.Errorf("%d matches grew the line from %d to %d bytes", len(a.contentSearchMatches), len(line), len(out))
	}
}

func TestHighlightSpansRestoresStyle(t *testing.T) {
	red := "\x1b[31m"
	style := lipgloss.NewStyle().Bold(true)
	out := highlightSpans(red+"ab cd ef\x1b[0m", []highlightSpan{{0, 2, style}, {6, 8, style}})
	if ansi.Strip(out) != "ab cd ef" {
		t.Fatalf("text changed: %q", out)
	}
	// After the first span, " cd " must be red again.
	i := strings.Index(out, " cd ")
	if i < 0 || !strings.HasSuffix(out[:i], red) {
		t.Errorf("style not restored before %q: %q", " cd ", out)
	}
}

func TestMouseDragSelectsAndCopies(t *testing.T) {
	for _, border := range []string{config.ContentBorderAlways, config.ContentBorderWithSidebar} {
		cfg := config.Default()
		cfg.UI.ContentBorder = border
		a := NewApp(parser.ParseMarkdown("alpha beta\n\ngamma delta\n"), "f.md", "", cfg)
		a.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
		a.View()

		// Find the screen cells of "beta" and "gamma" in the rendered lines.
		x0, y0 := a.contentOrigin()
		var bx, by, gx, gy int
		for i, l := range a.renderedLines {
			s := ansi.Strip(l)
			if j := strings.Index(s, "beta"); j >= 0 {
				bx, by = x0+j, y0+i
			}
			if j := strings.Index(s, "gamma"); j >= 0 {
				gx, gy = x0+j+len("gamma")-1, y0+i
			}
		}
		a.Update(tea.MouseClickMsg{X: bx, Y: by, Button: tea.MouseLeft})
		a.Update(tea.MouseMotionMsg{X: gx, Y: gy, Button: tea.MouseLeft})
		_, cmd := a.Update(tea.MouseReleaseMsg{X: gx, Y: gy, Button: tea.MouseLeft})

		if got := a.selectedText(); got != "beta\n\ngamma" {
			t.Errorf("%s: selected %q, want %q", border, got, "beta\n\ngamma")
		}
		if cmd == nil {
			t.Fatalf("%s: no clipboard command", border)
		}
		if _, cmd = a.Update(cmd()); !strings.HasPrefix(a.statusMsg, "Copied the selection via ") {
			t.Errorf("%s: no OSC 52 copy (status %q)", border, a.statusMsg)
		}
		if !strings.Contains(a.View().Content, "beta") {
			t.Errorf("%s: selection highlight lost the text", border)
		}
	}
}

func TestStatusBarFitsAt80Columns(t *testing.T) {
	cfg := config.Default()
	cfg.Images.Enabled = false
	cfg.Keys.Quit = "x"
	a := NewApp(parser.ParseMarkdown("# A fairly long heading title here\n\ntext\n"), "f.md", "", cfg)
	a.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	for _, idx := range []int{-1, 0} {
		a.navigateTo(idx)
		if n := len(strings.Split(a.View().Content, "\n")); n != 24 {
			t.Fatalf("selectedIdx %d: view has %d lines, want 24", idx, n)
		}
	}
	a.Update(tea.WindowSizeMsg{Width: 200, Height: 24})
	if status := ansi.Strip(a.renderStatus()); !strings.Contains(status, "x:quit") {
		t.Fatalf("status ignores the quit binding: %q", status)
	}
}

func TestReloadKeepsScroll(t *testing.T) {
	cfg := config.Default()
	cfg.Images.Enabled = false
	md := strings.Repeat("line\n\n", 200)
	a := NewApp(parser.ParseMarkdown(md), "f.md", "", cfg)
	a.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	a.View()
	a.scrollContent(100)
	a.reload(parser.ParseMarkdown(md+"more\n"), "Reloaded")
	a.View()
	if a.contentOffset != 100 {
		t.Fatalf("contentOffset = %d after reload, want 100", a.contentOffset)
	}
}

func TestPreProcessLeavesCodeAlone(t *testing.T) {
	in := "```html\n<img src=\"a.png\" alt=\"x\">\n```\n\nUse `<a href=\"u\">t</a>` or <a href=\"v\">w</a> and <img src=\"b.png\">\n"
	want := "```html\n<img src=\"a.png\" alt=\"x\">\n```\n\nUse `<a href=\"u\">t</a>` or [w](v) and ![](b.png)\n"
	if got := PreProcessMarkdown(in); got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

func TestExtractCodeNodes_NestedFence(t *testing.T) {
	lines := strings.Split("````md\n```go\nx := 1\n```\n````\n~~~\ny\n~~~~\n", "\n")
	nodes := sectionNodes(lines)
	if len(nodes) != 2 || nodes[0].content != "```go\nx := 1\n```" || nodes[0].lang != "md" || nodes[1].content != "y" || nodes[1].endLine != 7 {
		t.Fatalf("got %+v", nodes)
	}
}

func TestPercentEncodedAnchor(t *testing.T) {
	a := NewApp(parser.ParseMarkdown("# Intro\n\n## Configuração rápida\n"), "f.md", "", config.Default())
	if got := a.findHeadingByAnchor("configura%C3%A7%C3%A3o-r%C3%A1pida"); got != 1 {
		t.Fatalf("findHeadingByAnchor = %d, want 1", got)
	}
}

func TestFuzzyMatch(t *testing.T) {
	for _, tt := range []struct {
		text, pattern string
		want          bool
	}{
		{"getting started", "gst", true},
		{"getting started", "tsg", false},
		{"configuração", "çã", true},
		{"abc", "", true},
		{"", "a", false},
	} {
		if got := fuzzyMatch(tt.text, tt.pattern); got != tt.want {
			t.Errorf("fuzzyMatch(%q, %q) = %v", tt.text, tt.pattern, got)
		}
	}
}

func TestDuplicateHeadingAnchors(t *testing.T) {
	a := NewApp(parser.ParseMarkdown("# Usage\n\n# Usage\n\n# Usage\n"), "f.md", "", config.Default())
	for anchor, want := range map[string]int{"usage": 0, "usage-1": 1, "usage-2": 2, "usage-3": -1} {
		if got := a.findHeadingByAnchor(anchor); got != want {
			t.Errorf("%s: got %d, want %d", anchor, got, want)
		}
	}
}

func TestFollowMarkdownLinkAndBack(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "main.md")
	other := filepath.Join(dir, "docs", "other file.md")
	os.MkdirAll(filepath.Dir(other), 0o755)
	os.WriteFile(main, []byte("# Main\n\n## Links\n\nSee [setup](docs/other%20file.md#setup).\n"), 0o644)
	os.WriteFile(other, []byte("# Other\n\n## Intro\n\n## Setup\n\ntext\n"), 0o644)
	data, _ := os.ReadFile(main)
	cfg := config.Default()
	cfg.Images.Enabled = false
	a := NewApp(parser.ParseMarkdown(string(data)), main, main, cfg)
	a.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	a.navigateTo(1)

	path, anchor, ok := a.localMarkdownLink("docs/other%20file.md#setup")
	if !ok || path != other || anchor != "setup" {
		t.Fatalf("localMarkdownLink = %q, %q, %v", path, anchor, ok)
	}
	for _, l := range []string{"https://x.org/a.md", "mailto:a@b.md", "image.png", "#setup"} {
		if _, _, ok := a.localMarkdownLink(l); ok {
			t.Errorf("%q treated as a local markdown file", l)
		}
	}
	a.openMarkdownLink(path, anchor)
	if a.filepath != other || a.selectedIdx != 2 {
		t.Fatalf("after following: file %q, heading %d", a.filepath, a.selectedIdx)
	}
	// A late reload from the old file's watcher must not replace the document.
	a.Update(fileReloadMsg{main, parser.ParseMarkdown("# Stale\n")})
	if a.doc.Headings[0].Text != "Other" {
		t.Fatal("a reload of the previous file replaced the document")
	}
	a.navGoBack()
	if a.filepath != main || a.selectedIdx != 1 {
		t.Fatalf("after back: file %q, heading %d", a.filepath, a.selectedIdx)
	}
	a.navGoForward()
	if a.filepath != other || a.selectedIdx != 2 {
		t.Fatalf("after forward: file %q, heading %d", a.filepath, a.selectedIdx)
	}
	if a.openMarkdownLink(filepath.Join(dir, "missing.md"), ""); !strings.Contains(a.statusMsg, "Open failed") || a.filepath != other {
		t.Fatalf("missing file: status %q, file %q", a.statusMsg, a.filepath)
	}
	if a.watcher != nil {
		a.watcher.Close()
	}
}

func TestWatcherReadsASaveOnceComplete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.md")
	os.WriteFile(path, []byte("# Old\n"), 0o644)
	w, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := w.Add(dir); err != nil {
		t.Fatal(err)
	}
	got := make(chan tea.Msg, 1)
	go func() { got <- waitForFileChange(w, path)() }()
	// A save written in two parts, the second shortly after the first.
	f, _ := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0)
	f.WriteString("# Ne")
	time.Sleep(reloadDelay / 3)
	f.WriteString("w\n")
	f.Close()
	select {
	case msg := <-got:
		m, ok := msg.(fileReloadMsg)
		if !ok || len(m.doc.Headings) != 1 || m.doc.Headings[0].Text != "New" {
			t.Fatalf("got %#v", msg)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no reload")
	}
}

func TestEditorArgs(t *testing.T) {
	for _, tt := range []struct {
		editor string
		want   string
	}{
		{"", "vi +7 f.md"},
		{"nvim", "nvim +7 f.md"},
		{"/usr/bin/vim -u NONE", "/usr/bin/vim -u NONE +7 f.md"},
		{"code --wait", "code --wait --goto f.md:7"},
		{"hx", "hx f.md:7"},
		{"unknown-editor", "unknown-editor f.md"},
	} {
		if got := strings.Join(editorArgs(strings.Fields(tt.editor), "f.md", 7), " "); got != tt.want {
			t.Errorf("%q: got %q, want %q", tt.editor, got, tt.want)
		}
	}
	if got := strings.Join(editorArgs([]string{"nvim"}, "f.md", 0), " "); got != "nvim f.md" {
		t.Errorf("no line: got %q", got)
	}
}

func TestEditorLine(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("---\ntitle: x\n---\n\n# Top\n\n")
	for i := 1; i <= 60; i++ {
		fmt.Fprintf(&sb, "Paragraph %d.\n\n", i)
	}
	sb.WriteString("## Tail\n\nNeedle here.\n")
	cfg := config.Default()
	cfg.Images.Enabled = false
	a := NewApp(parser.ParseMarkdown(sb.String()), "f.md", "f.md", cfg)
	a.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	a.View()
	// File lines: front matter 1-3, blank 4, "# Top" 5, blank 6, then
	// "Paragraph i." on line 5+2i, "## Tail" on 127, "Needle here." on 129.
	if got := a.editorLine(); got != 5 {
		t.Errorf("top of the document: line %d, want 5", got)
	}
	a.scrollContent(10)
	a.View()
	var n int
	for i := a.contentOffset; n == 0 && i < len(a.renderedLines); i++ {
		fmt.Sscanf(strings.TrimSpace(ansi.Strip(a.renderedLines[i])), "Paragraph %d.", &n)
	}
	if got := a.editorLine(); n == 0 || got != 5+2*n {
		t.Errorf("scrolled: line %d, want %d (Paragraph %d)", got, 5+2*n, n)
	}
	a.contentSearchQuery = "needle"
	a.findContentSearchMatches()
	a.scrollToContentMatch(0)
	if got := a.editorLine(); got != 129 {
		t.Errorf("search match: line %d, want 129", got)
	}
	a.contentOffset = 0 // the match is now off screen
	if got := a.editorLine(); got != 5 {
		t.Errorf("match off screen: line %d, want the first shown line 5", got)
	}
	a.contentSearchQuery, a.contentSearchMatches = "", nil
	a.navigateTo(1)
	a.View()
	if got := a.editorLine(); got != 127 {
		t.Errorf("section: line %d, want 127", got)
	}
}

func TestExtractCodeNodes_EmptyAndListBlocks(t *testing.T) {
	lines := strings.Split("```\n```\n\n- item:\n\n  ```sh\n  echo a\n    indented\n  ```\n", "\n")
	nodes := sectionNodes(lines)
	if len(nodes) != 1 || nodes[0].content != "echo a\n  indented" {
		t.Fatalf("want only the list block, dedented by its fence: %+v", nodes)
	}
}

func TestExtractCodeNodes_QuoteAndIndentedBlocks(t *testing.T) {
	lines := strings.Split("> ```sh\n> echo q\n> ```\n\n    indented\n\ntext\n\n    ```\n    fenced or indented\n    ```\n", "\n")
	var got []string
	for _, n := range sectionNodes(lines) {
		got = append(got, fmt.Sprintf("%d-%d %v %q", n.startLine, n.endLine, n.indented, n.content))
	}
	// The last block is indented code, fence lines included, as glamour shows
	// it.
	want := []string{`0-2 false "echo q"`, `4-4 true "indented"`, "8-10 true \"```\\nfenced or indented\\n```\""}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// spansOn returns, for each code span the source map places in the app's
// section (without the text search fallback), the text highlighted on
// screen.
func spansOn(a *App) map[string]string {
	locs := make([]nodeRenderLoc, len(a.codeNodes))
	for i := range locs {
		locs[i] = nodeRenderLoc{firstLine: -1, lastLine: -1, spanCol: -1, spanColEnd: -1}
	}
	placeNodesFromSource(a.srcMap, a.codeNodes, a.sectionLines, a.renderedLines, locs)
	got := map[string]string{}
	for i, n := range a.codeNodes {
		if loc := locs[i]; n.kind == nodeInlineCode && loc.firstLine >= 0 {
			got[n.content] = ansi.Cut(ansi.Strip(a.renderedLines[loc.firstLine]), loc.spanCol, loc.spanColEnd)
		}
	}
	return got
}

func TestWrappedTableCellsArePlaced(t *testing.T) {
	// The description wraps, and its end ("then `after`") is shown after the
	// row's next cell ("`third`"), which comes later in the source.
	md := "| Key | Description | Note |\n| --- | ----------- | ---- |\n" +
		"| `a` | a description long enough to wrap in its narrow column, then `after` | `third` |\n" +
		"| `b` | short | `fourth` |\n\nAnd `tail` after the table.\n"
	cfg := config.Default()
	cfg.Images.Enabled = false
	a := NewApp(parser.ParseMarkdown(md), "f.md", "", cfg)
	a.Update(tea.WindowSizeMsg{Width: 52, Height: 30})
	a.ensureRenderedLines()
	got := spansOn(a)
	for _, s := range []string{"a", "after", "third", "b", "fourth", "tail"} {
		if got[s] != s {
			t.Errorf("%q highlighted on %q", s, got[s])
		}
	}
	if !strings.Contains(strings.Join(a.renderedLines, "\n"), "after") || len(a.renderedLines) < 8 {
		t.Fatalf("the cell did not wrap:\n%s", strings.Join(a.renderedLines, "\n"))
	}
}

func TestLinkDestinationOnNextLineIsPlaced(t *testing.T) {
	// The destination starts on the line after "(", as CommonMark allows.
	md := "A [fork](\nhttps://github.com/go-yaml/yaml/) of\n`one`, kept by [YAML](\n  https://github.com/yaml/).\n"
	cfg := config.Default()
	cfg.Images.Enabled = false
	a := NewApp(parser.ParseMarkdown(md), "f.md", "", cfg)
	a.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	a.ensureRenderedLines()
	locs := make([]nodeRenderLoc, len(a.codeNodes))
	for i := range locs {
		locs[i] = nodeRenderLoc{firstLine: -1, lastLine: -1, spanCol: -1, spanColEnd: -1}
	}
	placeNodesFromSource(a.srcMap, a.codeNodes, a.sectionLines, a.renderedLines, locs)
	if len(a.codeNodes) != 3 {
		t.Fatalf("%d nodes, want 3", len(a.codeNodes))
	}
	for i, n := range a.codeNodes {
		if locs[i].firstLine < 0 {
			t.Errorf("%q not placed", n.content)
		}
	}
}

func TestTableArtInCodeIsNotReadByColumn(t *testing.T) {
	// A code block drawing a table pairs with no markdown table: the
	// document is aligned in source order.
	md := "```\n a │ b\n───┼───\n c │ d\n```\n\nThen `span` here.\n"
	cfg := config.Default()
	cfg.Images.Enabled = false
	a := NewApp(parser.ParseMarkdown(md), "f.md", "", cfg)
	a.Update(tea.WindowSizeMsg{Width: 60, Height: 30})
	a.ensureRenderedLines()
	if got := spansOn(a)["span"]; got != "span" {
		t.Fatalf("span highlighted on %q", got)
	}
	if loc := a.nodeRenderInfo[0]; a.codeNodes[0].kind != nodeCodeBlock || loc.lastLine-loc.firstLine != 2 {
		t.Fatalf("code block placed at %+v", loc)
	}
}

func TestTableBordersStayAlignedWithLinks(t *testing.T) {
	md := "| Command | Description |\n| ------- | ----------- |\n| [attach](attach.md) | Attach streams |\n| plain | No link here |\n"
	cfg := config.Default()
	cfg.Images.Enabled = false
	a := NewApp(parser.ParseMarkdown(md), "f.md", "", cfg)
	lines := a.renderGlamour(md, 70)
	cols := map[int]bool{}
	for _, l := range lines {
		s := ansi.Strip(l)
		if i := strings.IndexAny(s, "│┼"); i >= 0 {
			cols[ansi.StringWidth(s[:i])] = true
		}
	}
	if len(cols) != 1 {
		t.Fatalf("table borders at columns %v:\n%s", cols, ansi.Strip(strings.Join(lines, "\n")))
	}
}

func TestNoNodesOnLinkDefinitions(t *testing.T) {
	lines := strings.Split("See [#12][] and `code`.\n\n[#12]: https://example.com/pull/12\n[guide]: <https://example.com/g> \"The guide\"\n[x]: not a definition, `kept`\n", "\n")
	var got []string
	for _, n := range sectionNodes(lines) {
		got = append(got, n.content)
	}
	if want := "code|kept"; strings.Join(got, "|") != want {
		t.Fatalf("nodes %q, want %s", got, want)
	}
}

func TestExtractNodes(t *testing.T) {
	for _, tc := range []struct{ name, md, want string }{
		{"parentheses in a URL", "[a](https://x.org/A_(b)) end",
			`link 0:0-0:24 "https://x.org/A_(b)"`},
		{"reference link and image", "[a][r] ![i][r]\n\n[r]: https://x.org",
			`link 0:0-0:6 "https://x.org"|link 0:7-0:14 "https://x.org"`},
		{"link over two lines", "a [b\nc](https://x.org) d",
			`link 0:2-1:22 "https://x.org"`},
		{"nested brackets", "[[a](https://x.org)]",
			`link 0:1-0:19 "https://x.org"`},
		{"space before the destination", "[a]( https://x.org)",
			`link 0:0-0:19 "https://x.org"`},
		{"code span over two lines", "a `b\nc` d",
			`span 0:2-1:7 "b c"`},
		{"escaped pipe in a table cell", "| a |\n|---|\n| `x\\|y` |",
			`span 2:2-2:8 "x|y"`},
		{"no links in code", "`[a](b)` and `https://x.org`",
			`span 0:0-0:8 "[a](b)"|span 0:13-0:28 "https://x.org"`},
		{"code block without trailing blank lines", "```sh\nx\n\n```",
			`block 0-3 sh "x"`},
		{"unclosed fence", "```sh\nx",
			`block 0-2 sh "x"`},
		{"tags only in an HTML block", "<div>\n<a href=\"https://x.org\">a</a> `c` [m](https://y.org)\n</div>",
			`link 1:0-1:29 "https://x.org"`},
		{"image in a link", "[![i](a.png)](https://x.org)",
			`link 0:0-0:28 "https://x.org"|link 0:1-0:12 "a.png"`},
		{"bare URL and email", "see https://x.org/a. or a@b.org",
			`link 0:4-0:19 "https://x.org/a"|link 0:24-0:31 "a@b.org"`},
		{"HTML anchor in a paragraph", "x <a href=\"https://x.org\">t</a> y",
			`link 0:2-0:31 "https://x.org"`},
	} {
		var got []string
		for _, n := range extractNodes(strings.Split(tc.md, "\n")) {
			switch n.kind {
			case nodeCodeBlock:
				got = append(got, fmt.Sprintf("block %d-%d %s %q", n.startLine, n.endLine, n.lang, n.content))
			default:
				kind := map[nodeKind]string{nodeInlineCode: "span", nodeLink: "link"}[n.kind]
				got = append(got, fmt.Sprintf("%s %d:%d-%d:%d %q", kind, n.startLine, n.colStart, n.endLine, n.colEnd, n.content))
			}
		}
		if g := strings.Join(got, "|"); g != tc.want {
			t.Errorf("%s:\n got %s\nwant %s", tc.name, g, tc.want)
		}
	}
}
