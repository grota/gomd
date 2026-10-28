package tui

import (
	"encoding/base64"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/grota/gomd/internal/config"
	"github.com/grota/gomd/internal/parser"
)

// cellOf returns the cell of byte off in the first rendered line holding sub;
// off -1 means the line's last column.
func cellOf(t *testing.T, a *App, sub string, off int) textPos {
	t.Helper()
	for i, l := range a.renderedLines {
		s := ansi.Strip(l)
		if j := strings.Index(s, sub); j >= 0 {
			if off < 0 {
				return textPos{i, ansi.StringWidth(s) + 5}
			}
			if off == 0 && j == 0 {
				return textPos{i, 0}
			}
			return textPos{i, ansi.StringWidth(s[:j+off])}
		}
	}
	t.Fatalf("%q not rendered", sub)
	return textPos{}
}

func TestSelectionMarkdown(t *testing.T) {
	type end struct {
		sub string
		off int // byte offset in sub; -1 is past the line's end
	}
	lineStart := func(sub string) end { return end{sub, -2} }
	for _, tc := range []struct {
		name, md      string
		from, to      end
		rendered, md2 string
	}{
		{"plain", "alpha beta gamma\n", end{"alpha", 2}, end{"beta", 1}, "pha be", "pha be"},
		{"bold", "Some **bold** text\n", end{"bold", 2}, end{"text", 3}, "bold text", "**bold** text"},
		{"link", "See [docs](http://x.io) now\n", end{"docs", 1}, end{"docs", 2}, "", "[docs](http://x.io)"},
		{"inline code", "run `go test` now\n", end{"test", 1}, end{"now", 0}, "", "`go test` n"},
		{"list", "- one\n- two\n", lineStart("one"), end{"two", -1}, "", "- one\n- two"},
		{"heading", "## Title\n\ntext\n", lineStart("Title"), end{"Title", -1}, "", "## Title"},
		{"code whole", "```go\nx := 1\ny := 2\n```\n", lineStart("x := 1"), end{"y := 2", -1}, "", "```go\nx := 1\ny := 2\n```"},
		{"code part", "```go\nx := 1\ny := 2\n```\n", end{"x := 1", 0}, end{"x := 1", 5}, "", "x := 1"},
		{"table", "| a | b |\n|---|---|\n| 1 | 2 |\n\nafter\n", end{"1", 0}, end{"1", 0}, "", "| a | b |\n|---|---|\n| 1 | 2 |"},
		{"image", "![pic](./a.png)\n", end{"pic", 1}, end{"pic", 1}, "", "![pic](./a.png)"},
		{"wrapped", strings.Repeat("word ", 40) + "end\n", end{"end", 0}, end{"end", 2}, "end", "end"},
		{"html anchor", "See <a href=\"http://x.io\">docs</a> now\n", end{"docs", 1}, end{"docs", 1}, "", "<a href=\"http://x.io\">docs</a>"},
		{"html in code", "Use `<a href=\"u\">t</a>` here\n", end{"here", 0}, end{"here", 3}, "here", "here"},
		{"after html block", "<div align=\"center\">\n  <h1>T</h1>\n</div>\nIntro text.\n\n- **one** item\n\nfinal marker\n", end{"marker", 0}, end{"marker", 5}, "marker", "marker"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := NewApp(parser.ParseMarkdown(tc.md), "f.md", "", config.Default())
			a.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
			a.View()
			pos := func(e end) textPos {
				if e.off == -2 {
					return textPos{cellOf(t, a, e.sub, 0).line, 0}
				}
				return cellOf(t, a, e.sub, e.off)
			}
			a.sel = &selection{anchor: pos(tc.from), head: pos(tc.to)}
			got, gotMD := a.selectedText(), a.selectedMarkdown()
			if tc.rendered != "" && got != tc.rendered {
				t.Errorf("rendered copy %q, want %q", got, tc.rendered)
			}
			if gotMD != tc.md2 {
				t.Errorf("markdown copy %q, want %q", gotMD, tc.md2)
			}
		})
	}
}

func TestKeyCopiesSelectionMarkdown(t *testing.T) {
	var copied string
	defer func(f func(string) error) { tmuxLoadBuffer = f }(tmuxLoadBuffer)
	tmuxLoadBuffer = func(s string) error { copied = s; return nil }
	t.Setenv("TMUX", "test")

	a := NewApp(parser.ParseMarkdown("Some **bold** text\n"), "f.md", "", config.Default())
	a.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	a.View()
	a.sel = &selection{anchor: cellOf(t, a, "bold", 1), head: cellOf(t, a, "bold", 2)}
	_, cmd := a.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if cmd == nil {
		t.Fatal("no copy command")
	}
	a.Update(cmd())
	if copied != "**bold**" {
		t.Errorf("copied %q, want %q", copied, "**bold**")
	}
	if a.sel == nil {
		t.Errorf("the selection was cleared")
	}
}

func TestReleaseCopiesHTMLWithOSC5522(t *testing.T) {
	defer func(f func() bool) { osc5522Supported = f }(osc5522Supported)
	osc5522Supported = func() bool { return true }
	t.Setenv("TMUX", "")
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "a.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatal(err)
	}
	f.Close()

	md := "Some **bold** text\n\n![pic](a.png)\n\nafter\n"
	a := NewApp(parser.ParseMarkdown(md), "d.md", filepath.Join(dir, "d.md"), config.Default())
	a.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	a.View()
	from, to := cellOf(t, a, "bold", 1), cellOf(t, a, "after", 4)
	a.sel = &selection{anchor: from, head: to, dragging: true}
	x0, y0 := a.contentOrigin()
	_, cmd := a.Update(tea.MouseReleaseMsg{X: x0 + to.col, Y: y0 + to.line - a.contentOffset, Button: tea.MouseLeft})
	msg, ok := cmd().(clipboardDoneMsg)
	if !ok || msg.raw == "" {
		t.Fatalf("no OSC 5522 write: %+v", msg)
	}
	var html string
	b64 := base64.StdEncoding
	htmlMime := "mime=" + b64.EncodeToString([]byte("text/html")) + ";"
	for _, part := range strings.Split(msg.raw, "\x1b\\") {
		if i := strings.Index(part, htmlMime); i >= 0 {
			chunk, _ := b64.DecodeString(part[i+len(htmlMime):])
			html += string(chunk)
		}
	}
	for _, want := range []string{"<strong>bold</strong>", `<img src="data:image/png;base64,`, `alt="pic"`, "after"} {
		if !strings.Contains(html, want) {
			t.Errorf("HTML %q lacks %q", html, want)
		}
	}
}

func TestOSC5522Write(t *testing.T) {
	t.Setenv("TMUX", "")
	html := strings.Repeat("x", 5000) // two chunks
	seq := osc5522Write("hi", html)
	b64 := base64.StdEncoding.EncodeToString
	text := "\x1b]5522;type=wdata:mime=" + b64([]byte("text/plain")) + ";" + b64([]byte("hi")) + "\x1b\\"
	if !strings.HasPrefix(seq, "\x1b]5522;type=write\x1b\\"+text) || !strings.HasSuffix(seq, "\x1b]5522;type=wdata\x1b\\") {
		t.Errorf("bad framing: %q", seq[:min(len(seq), 120)])
	}
	if n := strings.Count(seq, "mime="+b64([]byte("text/html"))); n != 2 {
		t.Errorf("HTML sent in %d chunks, want 2", n)
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
