package tui

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	tea "charm.land/bubbletea/v2"

	"github.com/grota/gomd/internal/config"
)

// Copying a selection, with ui.clipboard = "auto", tries in order:
//  1. OSC 5522 (Kitty clipboard protocol): text and image in one clipboard
//     entry. Kitty, Ghostty 1.4+; inside tmux through passthrough.
//  2. A local tool (wl-copy, xclip, pbcopy/osascript) when not over SSH. It
//     holds one format, so a covered image replaces the text.
//  3. OSC 52: text only. Inside tmux it needs set-clipboard on.

// clipboardDoneMsg reports a finished copy. raw, when set, is sent to the
// terminal; osc52 asks for the text to be sent through OSC 52.
type clipboardDoneMsg struct {
	status string
	raw    string
	osc52  string
}

// copyCmd copies text and, when imgPath is set, that image.
func copyCmd(method, text, imgPath string) tea.Cmd {
	chars := fmt.Sprintf("Copied %d characters", len([]rune(text)))
	if method == config.ClipboardOSC52 || imgPath == "" && !osc5522Supported() && !localClipboard() {
		return func() tea.Msg { return clipboardDoneMsg{status: chars, osc52: text} }
	}
	return func() tea.Msg {
		var img []byte
		if imgPath != "" {
			var err error
			if img, err = pngBytes(imgPath); err != nil {
				return clipboardDoneMsg{status: "Image not copied: " + err.Error(), osc52: text}
			}
		}
		withImg := chars
		if img != nil {
			withImg += " and an image"
		}
		switch {
		case osc5522Supported():
			return clipboardDoneMsg{status: withImg, raw: osc5522Write(text, img)}
		case localClipboard():
			if img != nil {
				err := copyWithTool(img, true)
				if err != nil {
					return clipboardDoneMsg{status: "Copy failed: " + err.Error()}
				}
				return clipboardDoneMsg{status: "Copied the image (a clipboard tool holds one format)"}
			}
			if err := copyWithTool([]byte(text), false); err != nil {
				return clipboardDoneMsg{status: "Copy failed: " + err.Error()}
			}
			return clipboardDoneMsg{status: chars}
		}
		return clipboardDoneMsg{status: chars + " (image needs OSC 5522)", osc52: text}
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

// osc5522Write builds a Kitty clipboard protocol write of text and, when
// img is not nil, a PNG image.
func osc5522Write(text string, img []byte) string {
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
	data := func(mime string, b []byte) {
		meta := "type=wdata:mime=" + base64.StdEncoding.EncodeToString([]byte(mime))
		const chunkSize = 4096 // bytes before encoding, the spec's maximum
		for i := 0; i < len(b); i += chunkSize {
			packet(meta, base64.StdEncoding.EncodeToString(b[i:min(i+chunkSize, len(b))]))
		}
	}
	packet("type=write", "")
	if text != "" {
		data("text/plain", []byte(text))
	}
	if img != nil {
		data("image/png", img)
	}
	packet("type=wdata", "")
	return sb.String()
}

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

// localClipboard reports whether a clipboard tool can reach the user's
// clipboard: not over SSH, and the platform's tool is installed.
var localClipboard = sync.OnceValue(func() bool {
	if os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_TTY") != "" {
		return false
	}
	name, _ := clipboardTool(false)
	_, err := exec.LookPath(name)
	return err == nil
})

// clipboardTool returns the command that reads data from stdin into the
// clipboard, as text or as a PNG image.
func clipboardTool(image bool) (string, []string) {
	switch {
	case runtime.GOOS == "darwin" && image:
		return "osascript", nil // see copyWithTool
	case runtime.GOOS == "darwin":
		return "pbcopy", nil
	case os.Getenv("WAYLAND_DISPLAY") != "" && image:
		return "wl-copy", []string{"--type", "image/png"}
	case os.Getenv("WAYLAND_DISPLAY") != "":
		return "wl-copy", nil
	case image:
		return "xclip", []string{"-selection", "clipboard", "-t", "image/png"}
	}
	return "xclip", []string{"-selection", "clipboard"}
}

func copyWithTool(data []byte, image bool) error {
	name, args := clipboardTool(image)
	if name == "osascript" {
		// osascript cannot read stdin as an image, so go through a file.
		f, err := os.CreateTemp("", "gomd-*.png")
		if err != nil {
			return err
		}
		defer os.Remove(f.Name())
		if _, err := f.Write(data); err != nil {
			f.Close()
			return err
		}
		f.Close()
		args = []string{"-e", fmt.Sprintf("set the clipboard to (read (POSIX file %q) as «class PNGf»)", f.Name())}
		data = nil
	}
	// No output pipes: wl-copy and xclip leave a child serving the clipboard,
	// and a pipe it inherits would keep Run waiting until it exits.
	cmd := exec.Command(name, args...)
	cmd.Stdin = bytes.NewReader(data)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// pngBytes returns the image at path as PNG, converting other formats.
func pngBytes(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if http.DetectContentType(data) == "image/png" {
		return data, nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
