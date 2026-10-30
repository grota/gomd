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
// different lines, with the files that miss the most. It checks nothing and
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
	var docs, nodes, misses, rescued, disagree int
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".md") {
			return nil
		}
		if fi, err := d.Info(); err != nil || fi.Size() < 2<<10 || fi.Size() > 300<<10 {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
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
		for i := range a.codeNodes {
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
		return nil
	})
	sort.Slice(results, func(i, j int) bool { return results[i].misses > results[j].misses })
	t.Logf("%d documents, %d nodes: the source map misses %d (the text search places %d of them); %d placed on different lines",
		docs, nodes, misses, rescued, disagree)
	for _, r := range results[:min(10, len(results))] {
		if r.misses > 0 {
			t.Log(fmt.Sprintf("%5d misses (%d placed by the text search)  %s", r.misses, r.rescued, r.path))
		}
	}
}
