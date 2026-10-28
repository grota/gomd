package tui

import (
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// Every copy is plain text, sent through OSC 52. Inside tmux it goes through
// "tmux load-buffer -w", which forwards it to the terminal with set-clipboard
// on or external, where a plain OSC 52 from gomd works only with on.

// clipboardDoneMsg reports a finished copy. osc52, when set, is the text
// still to send through OSC 52.
type clipboardDoneMsg struct {
	status string
	osc52  string
}

// tmuxLoadBuffer copies text into a tmux paste buffer and the clipboard.
// Tests replace it.
var tmuxLoadBuffer = func(text string) error {
	cmd := exec.Command("tmux", "load-buffer", "-w", "-")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// copyCmd copies text to the clipboard and reports status when done.
func copyCmd(text, status string) tea.Cmd {
	return func() tea.Msg {
		if inTmux() && tmuxLoadBuffer(text) == nil {
			return clipboardDoneMsg{status: status}
		}
		return clipboardDoneMsg{status: status, osc52: text}
	}
}

func (a *App) handleClipboardDone(msg clipboardDoneMsg) (tea.Model, tea.Cmd) {
	a.statusMsg = msg.status
	if msg.osc52 != "" {
		return a, tea.SetClipboard(msg.osc52)
	}
	return a, nil
}
