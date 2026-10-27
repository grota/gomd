package tui

import (
	"bytes"
	"encoding/base64"
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

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
	nodes := extractCodeNodes(lines)
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
	nodes := extractCodeNodes(lines)
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
	nodes := extractCodeNodes(lines)
	if len(nodes) != 1 {
		t.Fatalf("expected 1 inline code node from heading, got %d", len(nodes))
	}
	if nodes[0].content != "useCallback" || nodes[0].kind != nodeInlineCode {
		t.Errorf("unexpected node: %+v", nodes[0])
	}
}

func TestExtractCodeNodes_NoNodes(t *testing.T) {
	lines := []string{"Just plain prose.", "", "No code here."}
	nodes := extractCodeNodes(lines)
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
	nodes := extractCodeNodes(lines)
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
	nodes := extractCodeNodes(lines)
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
	nodes := extractCodeNodes(lines)
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
	nodes := extractCodeNodes(lines)
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
	nodes := extractCodeNodes(lines)

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
	if a.selectedIdx != -1 || a.navHistory != nil {
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
		cfg.UI.Clipboard = config.ClipboardOSC52 // leave the real clipboard alone
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
		if _, cmd = a.Update(cmd()); cmd == nil || a.statusMsg != "Copied 11 characters" {
			t.Errorf("%s: no OSC 52 copy (status %q)", border, a.statusMsg)
		}
		if !strings.Contains(a.View().Content, "beta") {
			t.Errorf("%s: selection highlight lost the text", border)
		}
	}
}

func TestOSC5522Write(t *testing.T) {
	img := bytes.Repeat([]byte{7}, 5000) // two chunks
	seq := osc5522Write("hi", img)
	b64 := base64.StdEncoding.EncodeToString
	text := "\x1b]5522;type=wdata:mime=" + b64([]byte("text/plain")) + ";" + b64([]byte("hi")) + "\x1b\\"
	if inTmux() {
		t.Skip("packets are wrapped for tmux passthrough")
	}
	if !strings.HasPrefix(seq, "\x1b]5522;type=write\x1b\\"+text) || !strings.HasSuffix(seq, "\x1b]5522;type=wdata\x1b\\") {
		t.Errorf("bad framing: %q", seq[:min(len(seq), 120)])
	}
	if n := strings.Count(seq, "mime="+b64([]byte("image/png"))); n != 2 {
		t.Errorf("image sent in %d chunks, want 2", n)
	}
}

func TestTermHasOSC5522(t *testing.T) {
	for s, want := range map[string]bool{
		"ghostty 1.4.0": true, "ghostty 1.3.1-arch2": false, "ghostty 2.0.0": true,
		"kitty(0.45.0)": true, "xterm 390": false, " ": false,
	} {
		if got := termHasOSC5522(s); got != want {
			t.Errorf("termHasOSC5522(%q) = %v, want %v", s, got, want)
		}
	}
}
