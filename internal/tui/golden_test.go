package tui

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/grota/gomd/internal/config"
	"github.com/grota/gomd/internal/parser"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata/golden")

// TestGolden pins what gomd shows for the documents in testdata: the
// rendered text, its colours, where each selectable node is found, and the
// markdown copied for each rendered line. Refactors of parsing, rendering or
// node mapping must keep these unchanged; after an intended change, rerun
// with -update and review the diff.
func TestGolden(t *testing.T) {
	for _, tc := range []struct {
		file  string
		width int // content width: the window is 2 columns wider
	}{
		{"readme.md", 80}, {"sample.md", 80}, {"edge.md", 80}, {"cases.md", 80}, {"cases.md", 40}, {"nodes.md", 80},
	} {
		name := tc.file
		if tc.width != 80 {
			name = fmt.Sprintf("%s@%d", tc.file, tc.width)
		}
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(filepath.Join("testdata", tc.file))
			if err != nil {
				t.Fatal(err)
			}
			cfg := config.Default()
			cfg.Images.Enabled = false
			a := NewApp(parser.ParseMarkdown(string(src)), tc.file, "", cfg)
			a.Update(tea.WindowSizeMsg{Width: tc.width + 2, Height: 40})
			var text, styled strings.Builder
			for _, idx := range append([]int{-1}, headingIndexes(a)...) {
				a.selectedIdx = idx
				a.rebuildSection()
				a.ensureRenderedLines()
				fmt.Fprintf(&text, "######## section %d\n", idx)
				fmt.Fprintf(&styled, "######## section %d\n", idx)
				writeSection(&text, &styled, a)
			}
			checkGolden(t, name+".txt", text.String())
			checkGolden(t, name+".ansi", styled.String())
		})
	}
}

func headingIndexes(a *App) []int {
	idx := make([]int, len(a.doc.Headings))
	for i := range idx {
		idx[i] = i
	}
	return idx
}

// writeSection writes the rendered section as text and nodes to text, and
// its styled lines to styled.
func writeSection(text, styled *strings.Builder, a *App) {
	text.WriteString("== rendered\n")
	for _, l := range a.renderedLines {
		text.WriteString(strings.TrimRight(ansi.Strip(l), " ") + "\n")
		fmt.Fprintf(styled, "%q\n", l)
	}
	text.WriteString("== nodes\n")
	kinds := map[nodeKind]string{nodeCodeBlock: "block", nodeInlineCode: "inline", nodeLink: "link"}
	for i, n := range a.codeNodes {
		loc := a.nodeRenderInfo[i]
		label := n.content
		if n.kind == nodeLink {
			label = n.display + " -> " + n.content
		}
		fmt.Fprintf(text, "%-6s src %d-%d col %d-%d  shown %d-%d col %d-%d  %q", kinds[n.kind],
			n.startLine, n.endLine, n.colStart, n.colEnd, loc.firstLine, loc.lastLine, loc.spanCol, loc.spanColEnd, label)
		if n.inline && loc.firstLine >= 0 && loc.spanCol >= 0 { // the text highlighted on its first line
			fmt.Fprintf(text, " on %q", ansi.Cut(ansi.Strip(a.renderedLines[loc.firstLine]), loc.spanCol, loc.spanColEnd))
		}
		text.WriteString("\n")
	}
	text.WriteString("== copy per line\n")
	for i, l := range a.renderedLines {
		w := ansi.StringWidth(ansi.Strip(l))
		if strings.TrimSpace(ansi.Strip(l)) == "" {
			continue
		}
		a.sel = &selection{anchor: textPos{i, 0}, head: textPos{i, w}}
		fmt.Fprintf(text, "%d: %q | %q\n", i, a.selectedText(), a.selectedMarkdown())
	}
	a.sel = nil
}

func checkGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", "golden", name)
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run go test -run TestGolden -update to create it)", err)
	}
	if got != string(want) {
		gl, wl := strings.Split(got, "\n"), strings.Split(string(want), "\n")
		for i := 0; i < max(len(gl), len(wl)); i++ {
			g, w := "", ""
			if i < len(gl) {
				g = gl[i]
			}
			if i < len(wl) {
				w = wl[i]
			}
			if g != w {
				t.Fatalf("%s differs at line %d:\n got  %s\n want %s\n(rerun with -update if the change is intended)", path, i+1, g, w)
			}
		}
	}
}
