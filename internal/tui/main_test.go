package tui

import (
	"os"
	"testing"
)

// TestMain keeps copies made by tests away from the real clipboard.
func TestMain(m *testing.M) {
	tmuxLoadBuffer = func(string) error { return nil }
	os.Exit(m.Run())
}
