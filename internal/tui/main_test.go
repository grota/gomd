package tui

import (
	"os"
	"testing"
)

// TestMain keeps copies made by tests away from the real clipboard and
// terminal.
func TestMain(m *testing.M) {
	tmuxLoadBuffer = func(string) error { return nil }
	osc5522Supported = func() bool { return false }
	os.Exit(m.Run())
}
