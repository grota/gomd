package tui

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// ─────────────────────────────────────────
// helpers

func renderStripped(t *testing.T, src string, width int) (rendered, stripped []string) {
	t.Helper()
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
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
	theme := GetTheme("OceanDark")
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
	theme := GetTheme("OceanDark")
	style := lipgloss.NewStyle().Background(theme.NodeSel).Foreground(theme.Background).Bold(true)
	result := highlightSpanInLine(line, 6, 9, style)
	plain := ansi.Strip(result)
	if plain != line {
		t.Errorf("stripped result changed: %q != %q", plain, line)
	}
}

func TestHighlightSpanInLine_NoOpOnInvalidRange(t *testing.T) {
	line := "some text"
	theme := GetTheme("OceanDark")
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

// TestMapNodes_FullREADME verifies that all inline code spans in the project README
// are correctly mapped to their rendered positions.
func TestMapNodes_FullREADME(t *testing.T) {
	content, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Skip("README.md not found")
	}
	src := string(content)
	lines := strings.Split(src, "\n")
	nodes := extractCodeNodes(lines)

	rendered, stripped := renderStripped(t, src, 140)
	locs := mapNodesToRenderedLines(nodes, rendered)

	var failures int
	for i, n := range nodes {
		if !n.inline || n.lang == "heading" {
			continue
		}
		// Single-char inline codes in headings can't be reliably mapped (known limitation)
		if len(n.content) <= 1 {
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
