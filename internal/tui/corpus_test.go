package tui

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/grota/gomd/internal/config"
	"github.com/grota/gomd/internal/parser"
)

// TestCorpus measures node placement on real documents: every .md file
// (2 KB to 300 KB) under the folder in GOMD_CORPUS, for example
//
//	GOMD_CORPUS=$(go env GOMODCACHE) go test ./internal/tui -run TestCorpus -v
//
// It reports how many nodes the source map cannot place, how many of those
// the text search places instead, and how many nodes the two place on
// different lines, with the files that miss the most, and how many one-line
// code spans the source map highlights on text other than their code. It checks nothing and
// skips without GOMD_CORPUS: run it before and after a source map change and
// compare.
func TestCorpus(t *testing.T) {
	root := os.Getenv("GOMD_CORPUS")
	if root == "" {
		t.Skip("set GOMD_CORPUS to a folder of markdown files")
	}
	type result struct {
		path            string
		misses, rescued int
	}
	var results []result
	var docs, nodes, misses, rescued, disagree, spans, wrong int
	walkCorpus(root, func(path string, src []byte) {
		cfg := config.Default()
		cfg.Images.Enabled = false
		a := NewApp(parser.ParseMarkdown(string(src)), "x.md", "", cfg)
		a.Update(tea.WindowSizeMsg{Width: 82, Height: 40})
		a.ensureRenderedLines()
		search := mapNodesToRenderedLines(a.codeNodes, a.renderedLines)
		fromMap := make([]nodeRenderLoc, len(a.codeNodes))
		for i := range fromMap {
			fromMap[i] = nodeRenderLoc{firstLine: -1, lastLine: -1, spanCol: -1, spanColEnd: -1}
		}
		placeNodesFromSource(a.srcMap, a.codeNodes, a.sectionLines, a.renderedLines, fromMap)
		r := result{path: path}
		for i, n := range a.codeNodes {
			// A code span on one line must be highlighted on its own code.
			if loc := fromMap[i]; n.kind == nodeInlineCode && loc.firstLine >= 0 && loc.firstLine == loc.lastLine {
				spans++
				if ansi.Cut(ansi.Strip(a.renderedLines[loc.firstLine]), loc.spanCol, loc.spanColEnd) != n.content {
					wrong++
				}
			}
			switch {
			case fromMap[i].firstLine < 0:
				r.misses++
				if search[i].firstLine >= 0 {
					r.rescued++
				}
			case search[i].firstLine >= 0 && search[i].firstLine != fromMap[i].firstLine:
				disagree++
			}
		}
		docs++
		nodes += len(a.codeNodes)
		misses += r.misses
		rescued += r.rescued
		results = append(results, r)
	})
	sort.Slice(results, func(i, j int) bool { return results[i].misses > results[j].misses })
	t.Logf("%d documents, %d nodes: the source map misses %d (the text search places %d of them); %d placed on different lines",
		docs, nodes, misses, rescued, disagree)
	t.Logf("of %d one-line code spans the source map placed, %d are not highlighted on their own code", spans, wrong)
	for _, r := range results[:min(10, len(results))] {
		if r.misses > 0 {
			t.Log(fmt.Sprintf("%5d misses (%d placed by the text search)  %s", r.misses, r.rescued, r.path))
		}
	}
}

// walkCorpus calls f with every .md file of 2 KB to 300 KB under root.
func walkCorpus(root string, f func(path string, src []byte)) {
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".md") {
			return nil
		}
		if fi, err := d.Info(); err != nil || fi.Size() < 2<<10 || fi.Size() > 300<<10 {
			return nil
		}
		if src, err := os.ReadFile(path); err == nil {
			f(path, src)
		}
		return nil
	})
}

// TestCorpusExtractors compares the nodes of the line scanner
// (extractCodeNodes) and of goldmark (extractNodes) on the documents under
// GOMD_CORPUS, while the second replaces the first. It counts the nodes
// alike, the nodes found by one only, and the nodes differing in a field,
// by kind; GOMD_SHOW=<text> prints examples of the counts whose name holds
// it. It checks nothing and skips without GOMD_CORPUS.
func TestCorpusExtractors(t *testing.T) {
	root := os.Getenv("GOMD_CORPUS")
	if root == "" {
		t.Skip("set GOMD_CORPUS to a folder of markdown files")
	}
	kinds := map[nodeKind]string{nodeCodeBlock: "block", nodeInlineCode: "span", nodeLink: "link"}
	counts, examples := map[string]int{}, map[string][]string{}
	note := func(bucket, example string) {
		counts[bucket]++
		if len(examples[bucket]) < 8 {
			examples[bucket] = append(examples[bucket], example)
		}
	}
	walkCorpus(root, func(path string, src []byte) {
		lines := strings.Split(strings.TrimRight(string(src), "\n"), "\n")
		key := func(n codeNode) string { return fmt.Sprint(n.kind, n.startLine, n.colStart) }
		at := func(n codeNode) string {
			return fmt.Sprintf("%s:%d %q", strings.TrimPrefix(path, root), n.startLine+1, ansi.Truncate(lines[n.startLine], 100, "…"))
		}
		byKey := map[string]codeNode{}
		for _, n := range extractNodes(lines) {
			byKey[key(n)] = n
		}
		for _, o := range extractCodeNodes(lines) {
			k := kinds[o.kind]
			n, ok := byKey[key(o)]
			if !ok {
				note(k+" scanner only", fmt.Sprintf("%s %q", at(o), o.content))
				continue
			}
			delete(byKey, key(o))
			same := true
			for _, f := range []struct {
				name     string
				old, new any
			}{
				{"content", o.content, n.content}, {"display", o.display, n.display}, {"lang", o.lang, n.lang},
				{"endLine", o.endLine, n.endLine}, {"colEnd", o.colEnd, n.colEnd},
				{"heading", o.heading, n.heading}, {"indented", o.indented, n.indented},
			} {
				if f.old != f.new && (o.kind != nodeCodeBlock || f.name != "colEnd") {
					same = false
					note(k+" "+f.name, fmt.Sprintf("%s\n      scanner  %q\n      goldmark %q", at(o), fmt.Sprint(f.old), fmt.Sprint(f.new)))
				}
			}
			if same {
				counts[k+" alike"]++
			}
		}
		for _, n := range byKey {
			note(kinds[n.kind]+" goldmark only", fmt.Sprintf("%s %q", at(n), n.content))
		}
	})
	names := make([]string, 0, len(counts))
	for b := range counts {
		names = append(names, b)
	}
	sort.Strings(names)
	show := os.Getenv("GOMD_SHOW")
	for _, b := range names {
		t.Logf("%-22s %6d", b, counts[b])
		if show != "" && strings.Contains(b, show) && !strings.HasSuffix(b, "alike") {
			for _, e := range examples[b] {
				t.Log("    " + e)
			}
		}
	}
}
