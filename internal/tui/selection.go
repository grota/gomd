package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
)

// Mouse selection in the content pane: drag to select, release to copy the
// plain text to the clipboard with OSC 52 (tea.SetClipboard).

// textPos is a cell of the rendered content: a line of renderedLines and a
// display column within it.
type textPos struct{ line, col int }

// selection spans anchor (where the drag started) to head (the pointer),
// both cells included.
type selection struct {
	anchor, head textPos
	dragging     bool
}

func (s *selection) ordered() (from, to textPos) {
	from, to = s.anchor, s.head
	if to.line < from.line || to.line == from.line && to.col < from.col {
		from, to = to, from
	}
	return from, to
}

// contentOrigin returns the screen cell of the content pane's first text cell.
func (a *App) contentOrigin() (x, y int) {
	x = a.sidebarWidth() + 1 // left border or blank column
	y = 1                    // title bar
	if a.contentBordered() {
		y++
	}
	return x, y
}

// inContent reports whether the screen cell lies in the content text area.
func (a *App) inContent(mx, my int) bool {
	x0, y0 := a.contentOrigin()
	return mx >= x0 && mx < x0+a.contentWidth()-2 && my >= y0 && my < y0+a.contentHeight()
}

// contentPosAt maps a screen cell to a content cell, clamped to the pane.
func (a *App) contentPosAt(mx, my int) textPos {
	x0, y0 := a.contentOrigin()
	row := min(max(my-y0, 0), max(a.contentHeight()-1, 0))
	col := min(max(mx-x0, 0), max(a.contentWidth()-3, 0))
	return textPos{a.contentOffset + row, col}
}

// selectionCols returns the selected column range [left, right) of line i,
// where w is the width of a full line.
func (a *App) selectionCols(i, w int) (left, right int) {
	from, to := a.sel.ordered()
	left, right = 0, w
	if i == from.line {
		left = from.col
	}
	if i == to.line {
		right = to.col + 1
	}
	return left, right
}

func (a *App) handleMouseMotion(msg tea.MouseMotionMsg) (tea.Model, tea.Cmd) {
	if a.sel == nil || !a.sel.dragging {
		return a, nil
	}
	m := msg.Mouse()
	// Dragging past the top or bottom edge scrolls one line per motion event.
	_, y0 := a.contentOrigin()
	if m.Y < y0 && a.contentOffset > 0 {
		a.contentOffset--
	} else if m.Y >= y0+a.contentHeight() {
		a.contentOffset++
		a.clampContentOffset()
	}
	a.sel.head = a.contentPosAt(m.X, m.Y)
	return a, nil
}

func (a *App) handleMouseRelease(msg tea.MouseReleaseMsg) (tea.Model, tea.Cmd) {
	if a.sel == nil || !a.sel.dragging {
		return a, nil
	}
	m := msg.Mouse()
	a.sel.head = a.contentPosAt(m.X, m.Y)
	a.sel.dragging = false
	if a.sel.anchor == a.sel.head {
		a.sel = nil // a plain click
		return a, nil
	}
	text := a.selectedText()
	a.statusMsg = fmt.Sprintf("Copied %d characters", len([]rune(text)))
	return a, tea.SetClipboard(text)
}

// selectedText returns the selection as plain text, without image rows and
// without the left margin all its lines share.
func (a *App) selectedText() string {
	from, to := a.sel.ordered()
	var lines []string
	partial := false // lines[0] starts mid-line, so its margin is already cut
	for i := from.line; i <= to.line && i < len(a.renderedLines); i++ {
		s := ansi.Strip(a.renderedLines[i])
		if strings.ContainsRune(s, kitty.Placeholder) {
			continue
		}
		left, right := a.selectionCols(i, lipgloss.Width(s))
		partial = partial || len(lines) == 0 && left > 0
		lines = append(lines, strings.TrimRight(ansi.Cut(s, left, right), " "))
	}
	margin := -1
	for i, l := range lines {
		if i == 0 && partial {
			continue
		}
		if t := strings.TrimLeft(l, " "); t != "" && (margin < 0 || len(l)-len(t) < margin) {
			margin = len(l) - len(t)
		}
	}
	for i, l := range lines {
		if i == 0 && partial {
			continue
		}
		lines[i] = l[min(max(margin, 0), len(l)):]
	}
	return strings.Join(lines, "\n")
}

// applySelectionHighlight paints the selection on lines [from, to), the ones
// on screen. Image rows are left alone: their colour carries the image id.
func (a *App) applySelectionHighlight(lines []string, from, to, w int) []string {
	sf, st := a.sel.ordered()
	out := make([]string, len(lines))
	copy(out, lines)
	style := lipgloss.NewStyle().Background(a.theme.Selected).Foreground(a.theme.textOn(a.theme.Selected))
	for i := max(from, sf.line); i < min(to, st.line+1, len(out)); i++ {
		if strings.ContainsRune(out[i], kitty.Placeholder) {
			continue
		}
		left, right := a.selectionCols(i, w)
		out[i] = highlightSpans(out[i], []highlightSpan{{left, right, style}})
	}
	return out
}
