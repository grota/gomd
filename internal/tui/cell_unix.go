//go:build unix

package tui

import (
	"os"

	"golang.org/x/sys/unix"
)

// cellAspect returns the height of a terminal cell over its width, from the
// window size in cells and pixels that the terminal reports (Kitty and
// Ghostty do), or defaultCellAspect when it reports no pixels.
func cellAspect() float64 {
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return defaultCellAspect
	}
	return aspectOf(ws.Col, ws.Row, ws.Xpixel, ws.Ypixel)
}
