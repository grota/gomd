package tui

import (
	"bytes"
	"os"
	"time"

	"golang.org/x/term"
)

// QuerySystemDark asks the terminal, once, whether the system uses a dark or
// light appearance (DSR 996, answered as CSI ? 997 ; 1|2 n by terminals and
// tmux 3.6+ that support mode 2031). ok is false when no answer arrived.
func QuerySystemDark(timeout time.Duration) (dark, ok bool) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false, false
	}
	defer tty.Close()
	// Without read deadlines (e.g. a non-pollable tty), a read could block on
	// a terminal that never answers, so skip the query entirely.
	if tty.SetReadDeadline(time.Now().Add(timeout)) != nil {
		return false, false
	}
	// Fd() would switch the file to blocking mode and disable the deadline.
	raw, err := tty.SyscallConn()
	if err != nil {
		return false, false
	}
	var state *term.State
	if raw.Control(func(fd uintptr) { state, err = term.MakeRaw(int(fd)) }) != nil || err != nil {
		return false, false
	}
	defer raw.Control(func(fd uintptr) { _ = term.Restore(int(fd), state) })

	// Primary device attributes (CSI c) is answered by every terminal, so its
	// reply ends the wait even where DSR 996 is unsupported.
	if _, err := tty.WriteString("\x1b[?996n\x1b[c"); err != nil {
		return false, false
	}
	var got []byte
	buf := make([]byte, 256)
	for {
		n, err := tty.Read(buf)
		got = append(got, buf[:n]...)
		if dark, ok, done := parseSchemeReply(got); done || err != nil {
			return dark, ok
		}
	}
}

// parseSchemeReply scans terminal replies for the colour-scheme report; done
// is true once the device-attributes reply (which follows it) has arrived.
func parseSchemeReply(b []byte) (dark, ok, done bool) {
	switch {
	case bytes.Contains(b, []byte("\x1b[?997;1n")):
		dark, ok = true, true
	case bytes.Contains(b, []byte("\x1b[?997;2n")):
		dark, ok = false, true
	}
	// DA1 reply: CSI ? <params> c, possibly after the 997 reply.
	for rest := b; ; {
		i := bytes.Index(rest, []byte("\x1b[?"))
		if i < 0 {
			return dark, ok, false
		}
		rest = rest[i+3:]
		j := 0
		for j < len(rest) && (rest[j] >= '0' && rest[j] <= '9' || rest[j] == ';') {
			j++
		}
		if j < len(rest) && rest[j] == 'c' {
			return dark, ok, true
		}
	}
}
