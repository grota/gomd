package tui

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"

	tea "charm.land/bubbletea/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

// Every copy carries plain text, sent through OSC 52. Inside tmux it goes
// through "tmux load-buffer -w", which forwards it to the terminal with
// set-clipboard on or external, where a plain OSC 52 from gomd works only
// with on. A copy may also carry markdown to offer as HTML: on a terminal
// with the Kitty clipboard protocol (OSC 5522) the text and the HTML go
// into one clipboard entry instead, and the application pasting picks one.

// clipboardDoneMsg reports a finished copy. raw, when set, is sent to the
// terminal; osc52, when set, is the text still to send through OSC 52.
type clipboardDoneMsg struct {
	status string
	raw    string
	osc52  string
}

// tmuxLoadBuffer copies text into a tmux paste buffer and the clipboard.
// Tests replace it.
var tmuxLoadBuffer = func(text string) error {
	cmd := exec.Command("tmux", "load-buffer", "-w", "-")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// copyCmd copies text to the clipboard and reports status when done. When
// md is set and the terminal supports OSC 5522, md rendered as HTML, with
// local images inlined, is copied too; mdPath resolves those images.
func copyCmd(text, status, md, mdPath string) tea.Cmd {
	return func() tea.Msg {
		if md != "" && osc5522Supported() {
			return clipboardDoneMsg{status: status + " with formatting", raw: osc5522Write(text, markdownHTML(md, mdPath))}
		}
		if inTmux() && tmuxLoadBuffer(text) == nil {
			return clipboardDoneMsg{status: status}
		}
		return clipboardDoneMsg{status: status, osc52: text}
	}
}

func (a *App) handleClipboardDone(msg clipboardDoneMsg) (tea.Model, tea.Cmd) {
	a.statusMsg = msg.status
	switch {
	case msg.raw != "":
		return a, tea.Raw(msg.raw)
	case msg.osc52 != "":
		return a, tea.SetClipboard(msg.osc52)
	}
	return a, nil
}

// maxInlineImage is the largest image file inlined in copied HTML; larger
// ones stay links.
const maxInlineImage = 5 << 20

// markdownHTML renders md as HTML, as a document fragment. Local images up
// to maxInlineImage are inlined as data URIs, so the HTML pastes with them.
func markdownHTML(md, mdPath string) string {
	md = PreProcessMarkdown(md)
	gm := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.DefinitionList),
		goldmark.WithRendererOptions(html.WithUnsafe()), // keep the document's own HTML
	)
	src := []byte(md)
	doc := gm.Parser().Parse(text.NewReader(src))
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if img, ok := n.(*ast.Image); ok && entering {
			if uri := dataURI(string(img.Destination), mdPath); uri != "" {
				img.Destination = []byte(uri)
			}
		}
		return ast.WalkContinue, nil
	})
	var buf bytes.Buffer
	if err := gm.Renderer().Render(&buf, src, doc); err != nil {
		return ""
	}
	return buf.String()
}

// dataURI returns the local image src as a data URI, or "" when it is not a
// readable local image of at most maxInlineImage bytes.
func dataURI(src, mdPath string) string {
	path := ResolveImagePath(src, mdPath)
	if path == "" {
		return ""
	}
	if fi, err := os.Stat(path); err != nil || fi.Size() > maxInlineImage {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	mime := http.DetectContentType(data)
	if !strings.HasPrefix(mime, "image/") {
		return ""
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
}

// osc5522Write builds a Kitty clipboard protocol write of text and, when
// set, html.
func osc5522Write(text, html string) string {
	var sb strings.Builder
	packet := func(meta, payload string) {
		seq := "\x1b]5522;" + meta
		if payload != "" {
			seq += ";" + payload
		}
		seq += "\x1b\\"
		if inTmux() {
			seq = wrapTmuxPassthrough(seq)
		}
		sb.WriteString(seq)
	}
	data := func(mime, s string) {
		meta := "type=wdata:mime=" + base64.StdEncoding.EncodeToString([]byte(mime))
		const chunkSize = 4096 // bytes before encoding, the spec's maximum
		for i := 0; i < len(s); i += chunkSize {
			packet(meta, base64.StdEncoding.EncodeToString([]byte(s[i:min(i+chunkSize, len(s))])))
		}
	}
	packet("type=write", "")
	data("text/plain", text)
	if html != "" {
		data("text/html", html)
	}
	packet("type=wdata", "")
	return sb.String()
}

// osc5522Supported reports whether the terminal supports the Kitty clipboard
// protocol, from its name and version. Inside tmux it needs passthrough.
var osc5522Supported = sync.OnceValue(func() bool {
	if inTmux() {
		out, err := exec.Command("tmux", "display", "-p", "#{allow-passthrough} #{client_termtype}").Output()
		f := strings.SplitN(strings.TrimSpace(string(out)), " ", 2)
		return err == nil && len(f) == 2 && (f[0] == "on" || f[0] == "all") && termHasOSC5522(f[1])
	}
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return true
	}
	return termHasOSC5522(os.Getenv("TERM_PROGRAM") + " " + os.Getenv("TERM_PROGRAM_VERSION"))
})

var ghosttyVersionRe = regexp.MustCompile(`^ghostty (\d+)\.(\d+)`)

// termHasOSC5522 reports whether a terminal, named with its version as tmux's
// client_termtype shows it ("ghostty 1.4.0", "kitty(0.45.0)"), supports the
// Kitty clipboard protocol: every kitty that reports a version, Ghostty 1.4+.
func termHasOSC5522(nameVersion string) bool {
	s := strings.ToLower(nameVersion)
	if strings.HasPrefix(s, "kitty") {
		return true
	}
	m := ghosttyVersionRe.FindStringSubmatch(s)
	if m == nil {
		return false
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	return major > 1 || major == 1 && minor >= 4
}
