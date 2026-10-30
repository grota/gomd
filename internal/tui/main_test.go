package tui

import (
	"os"
	"testing"
)

// TestMain keeps copies made by tests away from the real clipboard and
// terminal, and downloaded images away from the user's cache.
func TestMain(m *testing.M) {
	tmuxLoadBuffer = func(string) error { return nil }
	osc5522Supported = func() bool { return false }
	cache, err := os.MkdirTemp("", "gomd-images-")
	if err != nil {
		panic(err)
	}
	remoteImagesDir = func() string { return cache }
	code := m.Run()
	os.RemoveAll(cache)
	os.Exit(code)
}
