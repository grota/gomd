package tui

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ghosttyTheme holds the parsed colors from a Ghostty theme file.
type ghosttyTheme struct {
	Palette             [16]string // palette indices 0–15
	Background          string
	Foreground          string
	SelectionBackground string
	SelectionForeground string
}

// parseGhosttyFile reads a Ghostty theme file and extracts colors.
func parseGhosttyFile(path string) (ghosttyTheme, error) {
	f, err := os.Open(path)
	if err != nil {
		return ghosttyTheme{}, err
	}
	defer f.Close()

	var gt ghosttyTheme
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch {
		case key == "background":
			gt.Background = val
		case key == "foreground":
			gt.Foreground = val
		case key == "selection-background":
			gt.SelectionBackground = val
		case key == "selection-foreground":
			gt.SelectionForeground = val
		case key == "palette":
			// "palette = N=#rrggbb"
			pparts := strings.SplitN(val, "=", 2)
			if len(pparts) == 2 {
				idx, err := strconv.Atoi(strings.TrimSpace(pparts[0]))
				if err == nil && idx >= 0 && idx < 16 {
					gt.Palette[idx] = strings.TrimSpace(pparts[1])
				}
			}
		}
	}
	return gt, scanner.Err()
}

// ghosttyToTheme converts a parsed Ghostty theme into a gomd Theme.
//
// Mapping:
//   - Background  → ghostty background
//   - Foreground  → ghostty foreground
//   - Border      → palette 8  (bright black / dark gray)
//   - Selected    → selection-background (fallback: palette 0)
//   - StatusBar   → palette 8
//   - Heading1    → palette 3  (yellow)
//   - Heading2    → palette 2  (green)
//   - Heading3    → palette 4  (blue)
//   - HeadingN    → palette 8  (bright black)
//   - Highlight   → palette 11 (bright yellow)
//   - Code        → palette 0  (black)
//   - Search      → palette 1  (red)
//   - NodeSel     → palette 6  (cyan)
func ghosttyToTheme(name string, gt ghosttyTheme) Theme {
	c := func(hex string, fallback string) lipgloss.Color {
		if hex != "" {
			return lipgloss.Color(hex)
		}
		return lipgloss.Color(fallback)
	}

	// Selected: prefer selection-background, fallback to palette 0.
	selectedBg := gt.SelectionBackground
	if selectedBg == "" {
		selectedBg = gt.Palette[0]
	}

	return Theme{
		Name:       name,
		Background: c(gt.Background, "#000000"),
		Foreground: c(gt.Foreground, "#ffffff"),
		Border:     c(gt.Palette[8], "#4c4c4c"),
		Selected:   c(selectedBg, "#000000"),
		StatusBar:  c(gt.Palette[8], "#4c4c4c"),
		Heading1:   c(gt.Palette[3], "#b8b87a"),
		Heading2:   c(gt.Palette[2], "#7ab87a"),
		Heading3:   c(gt.Palette[4], "#7a7ab8"),
		HeadingN:   c(gt.Palette[8], "#4c4c4c"),
		Highlight:  c(gt.Palette[11], "#dbdbbd"),
		Code:       c(gt.Palette[0], "#000000"),
		Search:     c(gt.Palette[1], "#b87a7a"),
		NodeSel:    c(gt.Palette[6], "#7ab8b8"),
	}
}

// LoadGhosttyTheme loads a named Ghostty theme from a directory and converts it
// to a gomd Theme. The theme name should match the filename in the directory.
func LoadGhosttyTheme(dir, name string) (Theme, error) {
	path := filepath.Join(dir, name)
	gt, err := parseGhosttyFile(path)
	if err != nil {
		return Theme{}, fmt.Errorf("loading ghostty theme %q: %w", name, err)
	}
	return ghosttyToTheme(name, gt), nil
}
