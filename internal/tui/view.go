// Package tui provides the interactive terminal user interface for gomd.
package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/grota/gomd/internal/config"
)

// ─────────────────────────────────────────────
// Layout helpers
// ─────────────────────────────────────────────

func (a *App) sidebarWidth() int {
	if a.sidebarHidden {
		return 0
	}

	// Compute based on longest heading (with indent)
	maxLen := len("(Document)")
	for _, h := range a.doc.Headings {
		// indent(2*level) + marker(level) + space(1) + text
		entryLen := 2*h.Level + h.Level + 1 + len([]rune(h.Text))
		if entryLen > maxLen {
			maxLen = entryLen
		}
	}

	// Add 2 for borders, 1 for padding
	ideal := maxLen + 3

	// Clamp: minimum 20, maximum 40% of width
	minW := 20
	maxW := a.width * 2 / 5
	if ideal < minW {
		ideal = minW
	}
	if ideal > maxW {
		ideal = maxW
	}
	if ideal > a.width/2 {
		ideal = a.width / 2
	}
	return ideal
}

func (a *App) contentWidth() int {
	sw := a.sidebarWidth()
	if sw == 0 {
		return a.width
	}
	return a.width - sw
}

// paneInnerHeight returns the number of content lines that fit inside a bordered pane.
// Layout: title(1) + border-top(1) + inner(N) + border-bottom(1) + status(1) = height
// So inner = height - 4.
func (a *App) paneInnerHeight() int {
	h := a.height - 4
	if h < 0 {
		return 0
	}
	return h
}

// outlineHeight is an alias kept for scrollOutlineToSelected.
func (a *App) outlineHeight() int { return a.paneInnerHeight() }
func (a *App) contentHeight() int { return a.paneInnerHeight() }

// ─────────────────────────────────────────────
// View
// ─────────────────────────────────────────────

func (a *App) View() tea.View {
	if a.width == 0 {
		v := tea.NewView("Loading...")
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	title := a.renderTitle()
	status := a.renderStatus()

	var body string
	if a.sidebarHidden {
		// Single bordered content pane, full width
		body = a.renderBorderedContent(a.width, a.focus == FocusContent)
	} else {
		sw := a.sidebarWidth()
		cw := a.contentWidth()
		outline := a.renderBorderedOutline(sw, a.focus == FocusSidebar)
		content := a.renderBorderedContent(cw, a.focus == FocusContent)
		body = lipgloss.JoinHorizontal(lipgloss.Top, outline, content)
	}

	view := lipgloss.JoinVertical(lipgloss.Left, title, body, status)

	if a.mode == ModeHelp {
		view = a.overlayHelp(view)
	}
	if a.mode == ModeThemePicker {
		view = a.overlayThemePicker(view)
	}

	v := tea.NewView(view)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// ─────────────────────────────────────────────
// Individual pane renderers
// ─────────────────────────────────────────────

func (a *App) renderTitle() string {
	name := filepath.Base(a.filename)
	if name == "" || name == "." {
		name = "gomd"
	}

	title := "gomd — " + name
	if bc := a.breadcrumb(); bc != "" {
		title += " > " + bc
	}

	return lipgloss.NewStyle().
		Background(a.theme.Border).
		Foreground(a.theme.Foreground).
		Bold(true).
		Width(a.width).
		Padding(0, 1).
		Render(title)
}

// breadcrumb returns the heading path from root to the current selection.
func (a *App) breadcrumb() string {
	if a.selectedIdx < 0 {
		return ""
	}
	h := a.doc.Headings[a.selectedIdx]
	var parts []string
	parts = append(parts, h.Text)

	// Walk backwards to find ancestor headings (lower level)
	for i := a.selectedIdx - 1; i >= 0; i-- {
		if a.doc.Headings[i].Level < h.Level {
			parts = append([]string{a.doc.Headings[i].Text}, parts...)
			h = a.doc.Headings[i]
		}
	}
	return strings.Join(parts, " > ")
}

// drawBox draws a bordered box around content lines.
// outerW is the full column width including border characters (2 cols).
// lines must have exactly paneInnerHeight() entries.
// focused controls border color.
func (a *App) drawBox(lines []string, outerW int, focused bool, title string) string {
	innerW := outerW - 2
	if innerW < 1 {
		innerW = 1
	}

	borderColor := a.theme.Border
	if focused {
		borderColor = a.theme.Highlight
	}
	bc := lipgloss.NewStyle().Foreground(borderColor).Background(a.theme.Background)

	// Top border: ┌─ title ──────┐
	titleRunes := []rune(title)
	topInner := innerW
	if len(titleRunes)+4 > topInner {
		topInner = len(titleRunes) + 4
	}
	var topMid string
	if title != "" {
		dashes := topInner - len(titleRunes) - 2 // 2 for spaces
		if dashes < 0 {
			dashes = 0
		}
		topMid = " " + title + " " + strings.Repeat("─", dashes)
	} else {
		topMid = strings.Repeat("─", topInner)
	}
	// Trim/pad topMid to innerW
	topMidRunes := []rune(topMid)
	if len(topMidRunes) > innerW {
		topMidRunes = topMidRunes[:innerW]
	}
	for len(topMidRunes) < innerW {
		topMidRunes = append(topMidRunes, '─')
	}
	topMid = string(topMidRunes)

	top := bc.Render("┌" + topMid + "┐")
	bottom := bc.Render("└" + strings.Repeat("─", innerW) + "┘")

	var sb strings.Builder
	sb.WriteString(top)
	sb.WriteByte('\n')
	borderL := bc.Render("│")
	borderR := bc.Render("│")
	bgStyle := lipgloss.NewStyle().
		Background(a.theme.Background).
		Foreground(a.theme.Foreground)
	// ANSI opening sequence for the theme colors, injected after every reset
	// in glamour output so the pane background survives.
	bgOpen := ansi.Style{}.ForegroundColor(a.theme.Foreground).BackgroundColor(a.theme.Background).String()

	for _, line := range lines {
		// Normalize line to exactly innerW display columns so the right
		// border always appears in the correct column regardless of whether
		// the line contains ANSI sequences (e.g. from glamour).
		line = ansi.Truncate(line, innerW, "")
		visW := lipgloss.Width(line)

		// Replace ANSI resets with resets that re-apply our background,
		// so glamour's internal resets don't clear the theme background.
		// Glamour v2 uses \x1b[m (short reset); normalise to \x1b[0m first.
		line = strings.ReplaceAll(line, "\x1b[m", "\x1b[0m")
		line = strings.ReplaceAll(line, "\x1b[0m", "\x1b[0m"+bgOpen)

		// Paint the entire line with the theme background.
		line = bgOpen + line + "\x1b[0m"
		if visW < innerW {
			line += bgStyle.Render(strings.Repeat(" ", innerW-visW))
		}
		sb.WriteString(borderL + line + borderR)
		sb.WriteByte('\n')
	}
	sb.WriteString(bottom)
	return sb.String()
}

// renderBorderedOutline renders the outline pane with a manually drawn border.
func (a *App) renderBorderedOutline(outerW int, focused bool) string {
	innerW := outerW - 2
	if innerW < 1 {
		innerW = 1
	}
	h := a.paneInnerHeight()

	// Total entries: root + headings. outlineOffset is a display index.
	total := a.totalEntries()
	endDisplay := a.outlineOffset + h
	if endDisplay > total {
		endDisplay = total
	}

	lines := make([]string, 0, h)
	for di := a.outlineOffset; di < endDisplay; di++ {
		// di==0 is root; di>0 maps to heading di-1
		var text string
		var isSelected bool
		var headingLevel int

		if di == 0 {
			text = "(Document)"
			isSelected = a.selectedIdx == -1
			headingLevel = 0
		} else {
			hd := a.doc.Headings[di-1]
			indent := strings.Repeat("  ", hd.Level)
			marker := strings.Repeat("#", hd.Level)
			text = indent + marker + " " + hd.Text
			isSelected = a.selectedIdx == di-1
			headingLevel = hd.Level
		}

		maxRunes := innerW
		if maxRunes < 1 {
			maxRunes = 1
		}
		if len([]rune(text)) > maxRunes {
			text = string([]rune(text)[:maxRunes-1]) + "…"
		}

		if isSelected {
			lines = append(lines, lipgloss.NewStyle().
				Background(a.theme.Selected).
				Foreground(a.theme.Foreground).
				Bold(true).
				Width(innerW).
				Render(text))
		} else {
			var fg Color
			switch headingLevel {
			case 0:
				fg = a.theme.Foreground
			case 1:
				fg = a.theme.Heading1
			case 2:
				fg = a.theme.Heading2
			case 3:
				fg = a.theme.Heading3
			default:
				fg = a.theme.HeadingN
			}
			lines = append(lines, lipgloss.NewStyle().
				Background(a.theme.Background).
				Foreground(fg).
				Width(innerW).
				Render(text))
		}
	}
	for len(lines) < h {
		lines = append(lines, lipgloss.NewStyle().Background(a.theme.Background).Width(innerW).Render(""))
	}

	return a.drawBox(lines, outerW, focused, "Outline")
}

// renderBorderedContent renders the section content pane with a manually drawn border.
func (a *App) renderBorderedContent(outerW int, focused bool) string {
	innerW := outerW - 2
	if innerW < 1 {
		innerW = 1
	}
	inner := a.renderContentWidth(innerW)
	lines := strings.Split(inner, "\n")
	h := a.paneInnerHeight()
	// Ensure exactly h lines
	for len(lines) < h {
		lines = append(lines, "")
	}
	lines = lines[:h]

	return a.drawBox(lines, outerW, focused, "")
}

func (a *App) renderContentWidth(w int) string {
	h := a.contentHeight()
	if w <= 0 || h <= 0 {
		return strings.Repeat("\n", max(h-1, 0))
	}

	return a.renderContentGlamour(w, h)
}

// renderContentGlamour renders the current section using glamour and returns a
// newline-joined string of exactly h lines, each at most w columns wide.
// In ModeNodeSelect it overlays background highlights on selectable nodes.
func (a *App) renderContentGlamour(w, h int) string {
	a.ensureRenderedLines()

	lines := a.renderedLines

	// In node-select mode, highlight only the filtered nodes.
	if a.mode == ModeNodeSelect {
		indices := a.filteredNodeIndices()
		if len(indices) > 0 {
			filtered := make([]codeNode, len(indices))
			filteredInfo := make([]nodeRenderLoc, len(indices))
			for i, idx := range indices {
				filtered[i] = a.codeNodes[idx]
				if idx < len(a.nodeRenderInfo) {
					filteredInfo[i] = a.nodeRenderInfo[idx]
				}
			}
			lines = applyNodeHighlights(lines, filtered, a.nodeSelIdx, filteredInfo, a.theme, w)
		}
	}

	// In jump mode, dim content and overlay labels.
	if a.mode == ModeJump && a.jumpLabels != nil {
		lines = a.applyJumpLabels(lines, w)
	}

	// Apply content search highlights
	if len(a.contentSearchMatches) > 0 && a.contentSearchQuery != "" {
		lines = a.applyContentSearchHighlights(lines)
	}

	start := a.contentOffset
	if start > len(lines) {
		start = len(lines)
	}
	end := start + h
	if end > len(lines) {
		end = len(lines)
	}
	visible := lines[start:end]

	// Pad to exactly h lines
	out := make([]string, h)
	copy(out, visible)
	return strings.Join(out, "\n")
}

// mapNodesToRenderedLines maps each codeNode to its location in the glamour-rendered
// output. Returns a slice of nodeRenderLoc parallel to nodes.
//
// For fenced blocks: finds the first rendered line whose stripped text contains
// the first non-empty line of the block body, then extends to cover all body lines.
// For inline nodes: finds the rendered line containing the span text, and records
// the display-column range of that span within the stripped line.
// byteColToDisplayCol converts a byte offset within a stripped (ANSI-free) string
// to a display column count. Needed because strings.Index returns byte offsets
// but ansi.Truncate / lipgloss.Width operate in display columns (Unicode-aware).
func byteColToDisplayCol(s string, byteOff int) int {
	if byteOff <= 0 {
		return 0
	}
	if byteOff >= len(s) {
		return lipgloss.Width(s)
	}
	return lipgloss.Width(s[:byteOff])
}

func mapNodesToRenderedLines(nodes []codeNode, rendered []string) []nodeRenderLoc {
	result := make([]nodeRenderLoc, len(nodes))
	// Pre-strip ANSI from all rendered lines once.
	stripped := make([]string, len(rendered))
	for i, l := range rendered {
		stripped[i] = ansi.Strip(l)
	}

	// Nodes are ordered by source line. Their rendered positions are also
	// monotonically non-decreasing, so we scan forward from the previous
	// node's rendered position to avoid matching an earlier duplicate.
	//
	// Two separate forward cursors are maintained:
	//
	//   inlineSearchFromLine / inlineSearchFromCol
	//     Used by inline nodes (headings, backtick spans).
	//     After a fenced block this advances to lastLine+1 so that inline
	//     nodes cannot accidentally match text inside a rendered code block.
	//
	//   blockSearchFromLine
	//     Used by fenced block nodes only.
	//     Advances to firstLine (not lastLine) of the previous block so that
	//     consecutive source blocks whose rendered output overlaps (glamour
	//     can merge adjacent blocks) can each be located.
	inlineSearchFromLine := 0
	inlineSearchFromCol := -1
	blockSearchFromLine := 0

	for ni, n := range nodes {
		loc := nodeRenderLoc{firstLine: -1, lastLine: -1, spanCol: -1, spanColEnd: -1}

		if n.inline {
			if n.kind == nodeLink {
				// Link nodes: use n.display (the visible text glamour renders)
				// as the search needle. Links don't have the double-space padding
				// that inline code spans get.
				needle := n.display
				if needle == "" {
					needle = n.content // fallback
				}
				for ri := inlineSearchFromLine; ri < len(stripped); ri++ {
					s := stripped[ri]
					colOffset := 0
					if ri == inlineSearchFromLine && inlineSearchFromCol >= 0 {
						colOffset = inlineSearchFromCol + 1
						if colOffset > len(s) {
							continue
						}
						s = s[colOffset:]
					}
					idx, found := strings.Index(s, needle), needle
					if idx < 0 {
						// A long image caption may wrap: fall back to "alt 📷".
						if cut := strings.Index(needle, " 📷 "); cut > 0 {
							found = needle[:cut+len(" 📷")]
							idx = strings.Index(s, found)
						}
					}
					if idx >= 0 {
						bestCol := colOffset + idx
						bestColEnd := bestCol + len(found)
						loc.firstLine = ri
						loc.lastLine = ri
						loc.spanColByte = bestCol
						loc.spanColEndByte = bestColEnd
						fullLine := stripped[ri]
						loc.spanCol = byteColToDisplayCol(fullLine, bestCol)
						loc.spanColEnd = loc.spanCol + lipgloss.Width(fullLine[bestCol:bestColEnd])
						break
					}
				}
			} else {
				// when mid-sentence, e.g. `useCallback` → "  useCallback  ".
				// When at end-of-sentence before punctuation only one trailing
				// space may appear, e.g. `Object.is`. → "  Object.is .".
				// Strategy: in a single forward pass from searchFromLine, on each
				// rendered line find BOTH the padded form ("  content  ") and the
				// bare form, then take the EARLIEST (minimum column) match.  This
				// prevents a later padded occurrence from pre-empting an earlier bare
				// occurrence on the same line.
				//
				// On inlineSearchFromLine, respect inlineSearchFromCol to handle multiple spans
				// on the same rendered line (same source line).
				padded := "  " + n.content + "  "
				paddedNBSP := "\u00a0" + n.content + "\u00a0"
				needle := n.content
				for ri := inlineSearchFromLine; ri < len(stripped); ri++ {
					s := stripped[ri]
					colOffset := 0
					if ri == inlineSearchFromLine && inlineSearchFromCol >= 0 {
						colOffset = inlineSearchFromCol + 1
						if colOffset > len(s) {
							continue
						}
						s = s[colOffset:]
					}
					bestCol := -1    // byte offset
					bestColEnd := -1 // byte offset

					// Check padded form (double-space in glamour v1, NBSP in v2)
					if paddedIdx := strings.Index(s, padded); paddedIdx != -1 {
						col := colOffset + paddedIdx + 2
						bestCol = col
						bestColEnd = col + len(n.content)
					} else if nbspIdx := strings.Index(s, paddedNBSP); nbspIdx != -1 {
						col := colOffset + nbspIdx + len("\u00a0")
						bestCol = col
						bestColEnd = col + len(n.content)
					}
					// Check alternative forms for inline code spans that may not have
					// full double-space padding (e.g., adjacent spans: `j`/`k` renders
					// as "j / k" with single spaces). Require at least 1 space before
					// AND 1 space after (or end of trimmed line) to confirm it's a span.
					if bestCol < 0 && !strings.Contains(stripped[ri], "##") {
						for pos := 0; pos < len(s); pos++ {
							idx := strings.Index(s[pos:], needle)
							if idx == -1 {
								break
							}
							absIdx := colOffset + pos + idx
							endIdx := absIdx + len(needle)
							fullLine := stripped[ri]
							// For longer needles (>=3 chars), require 2 spaces before to
							// distinguish code spans from prose. For short needles (1-2 chars),
							// 1 space suffices since they can't be confused with prose words.
							// Exception: also accept 1 space before when preceded by punctuation
							// (glamour places single space after opening parens/commas/pipes).
							var hasPre bool
							if len(needle) >= 3 {
								hasPre = absIdx > 1 && fullLine[absIdx-1] == ' ' && fullLine[absIdx-2] == ' '
								if !hasPre && absIdx > 0 && fullLine[absIdx-1] == ' ' {
									// Accept single space when preceded by punctuation or at position 1
									if absIdx == 1 {
										hasPre = true
									} else if absIdx > 1 {
										pre := fullLine[absIdx-2]
										hasPre = pre == '(' || pre == ',' || pre == '|' || pre == '/' || pre == ';'
									}
								}
							} else {
								hasPre = absIdx > 0 && fullLine[absIdx-1] == ' '
							}
							// Must have space after (or be at end of trimmed content)
							hasPost := endIdx >= len(strings.TrimRight(fullLine, " ")) || (endIdx < len(fullLine) && fullLine[endIdx] == ' ')
							if hasPre && hasPost {
								bestCol = absIdx
								bestColEnd = endIdx
								break
							}
							pos += idx + len(needle)
						}
					}
					if bestCol >= 0 {
						loc.firstLine = ri
						loc.lastLine = ri
						loc.spanColByte = bestCol
						loc.spanColEndByte = bestColEnd
						// Convert to display columns for highlightSpanInLine
						fullLine := stripped[ri]
						loc.spanCol = byteColToDisplayCol(fullLine, bestCol)
						loc.spanColEnd = loc.spanCol + lipgloss.Width(fullLine[bestCol:bestColEnd])
						break
					}
				}
			}
		} else {
			// Fenced block: find the first non-empty body line as a search anchor.
			needle := ""
			for _, bodyLine := range strings.Split(n.content, "\n") {
				t := strings.TrimSpace(bodyLine)
				if t != "" {
					needle = t
					break
				}
			}
			firstLine := -1
			if needle != "" {
				for ri := blockSearchFromLine; ri < len(stripped); ri++ {
					if strings.Contains(stripped[ri], needle) {
						firstLine = ri
						break
					}
				}
			}
			if firstLine == -1 {
				// Try fence marker as fallback
				for ri := blockSearchFromLine; ri < len(stripped); ri++ {
					if strings.Contains(stripped[ri], "```") || strings.Contains(stripped[ri], "~~~") {
						firstLine = ri
						break
					}
				}
			}
			if firstLine >= 0 {
				// Glamour renders fenced code blocks with 4-space indentation (vs
				// 2-space for prose). Scan forward from firstLine while lines are
				// either blank or start with at least 4 spaces to find the last
				// line of the block.
				lastLine := firstLine
				for ri := firstLine + 1; ri < len(stripped); ri++ {
					l := stripped[ri]
					trimmed := strings.TrimSpace(l)
					if trimmed == "" || strings.HasPrefix(l, "    ") {
						lastLine = ri
						continue
					}
					break
				}
				// Trim trailing blank lines.
				for lastLine > firstLine && strings.TrimSpace(stripped[lastLine]) == "" {
					lastLine--
				}
				loc.firstLine = firstLine
				loc.lastLine = lastLine
			}
		}

		result[ni] = loc
		// Advance search cursors.
		//
		// inlineSearchFromLine / inlineSearchFromCol: used by inline nodes.
		//   After a fenced block advances to lastLine+1 so inline nodes cannot
		//   match text that appears inside the rendered code block.
		//   After an inline node stays on loc.firstLine with col bumped past
		//   the span, allowing subsequent nodes on the same rendered line.
		//
		// blockSearchFromLine: used by fenced block nodes.
		//   Advances to firstLine only (not lastLine) so that consecutive
		//   source blocks whose rendered output overlaps can each be located.
		if loc.firstLine >= 0 {
			if !n.inline && loc.lastLine > loc.firstLine {
				// Fenced block: inline cursor jumps past the block.
				newInline := loc.lastLine + 1
				if newInline > inlineSearchFromLine {
					inlineSearchFromLine = newInline
					inlineSearchFromCol = -1
				}
				// Block cursor: only advance to firstLine.
				if loc.firstLine > blockSearchFromLine {
					blockSearchFromLine = loc.firstLine
				}
			} else {
				// Inline node: bump both inline and block cursors.
				if loc.firstLine > inlineSearchFromLine {
					inlineSearchFromLine = loc.firstLine
					inlineSearchFromCol = -1
				}
				if loc.spanColEndByte > 0 {
					inlineSearchFromCol = loc.spanColEndByte - 1
				}
				if loc.firstLine > blockSearchFromLine {
					blockSearchFromLine = loc.firstLine
				}
			}
		}
	}
	return result
}

// highlightSpanInLine applies a highlight style to the substring of a glamour-rendered
// line at display columns [colStart, colEnd). The rest of the line retains its original
// ANSI styling. Returns the modified line.
func highlightSpanInLine(line string, colStart, colEnd int, style lipgloss.Style) string {
	if colStart < 0 || colEnd <= colStart {
		return line
	}
	// Split the rendered string at display column boundaries using ansi.Truncate.
	// prefix = first colStart columns
	// middle = next (colEnd-colStart) columns
	// suffix = everything after colEnd
	prefix := ansi.Truncate(line, colStart, "")
	withMid := ansi.Truncate(line, colEnd, "")
	// middle is withMid minus prefix — we can get it by stripping the prefix bytes
	// from withMid. Since ansi.Truncate preserves ANSI codes, we use TruncateLeft.
	middle := ansi.TruncateLeft(withMid, colStart, "")
	suffix := ansi.TruncateLeft(line, colEnd, "")
	// Re-render the middle span with the highlight style.
	plainMiddle := ansi.Strip(middle)
	return prefix + style.Render(plainMiddle) + suffix
}

// applyContentSearchHighlights highlights all content search matches in the rendered lines.
func (a *App) applyContentSearchHighlights(lines []string) []string {
	if len(a.contentSearchMatches) == 0 {
		return lines
	}
	out := make([]string, len(lines))
	copy(out, lines)

	matchStyle := lipgloss.NewStyle().Background(lipgloss.Color("#5f5f00")).Foreground(lipgloss.Color("#ffffff"))
	currentStyle := lipgloss.NewStyle().Background(lipgloss.Color("#af8700")).Foreground(lipgloss.Color("#000000")).Bold(true)

	// Group matches by line for efficient processing
	// Apply from right to left to preserve column positions
	lineMatches := make(map[int][]int) // line -> list of match indices
	for i, m := range a.contentSearchMatches {
		lineMatches[m.line] = append(lineMatches[m.line], i)
	}

	for lineIdx, matchIdxs := range lineMatches {
		if lineIdx >= len(out) {
			continue
		}
		// Apply in reverse order (right to left) to preserve positions
		for i := len(matchIdxs) - 1; i >= 0; i-- {
			mi := matchIdxs[i]
			m := a.contentSearchMatches[mi]
			style := matchStyle
			if mi == a.contentSearchIdx {
				style = currentStyle
			}
			out[lineIdx] = highlightSpanInLine(out[lineIdx], m.colStart, m.colEnd, style)
		}
	}
	return out
}

// applyNodeHighlights returns a copy of rendered lines with highlights applied
// for node-select mode:
//   - All selectable inline/block nodes get a dim "available" background tint.
//   - The currently selected node gets a bright "selected" background.
//
// applyJumpLabels overlays labels next to selectable nodes with subtle highlights.
// Content remains unchanged; nodes get a background tint and labels appear beside them.
func (a *App) applyJumpLabels(lines []string, w int) []string {
	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ffffff")).
		Background(lipgloss.Color("#ff5f00"))

	// Copy lines
	out := make([]string, len(lines))
	copy(out, lines)

	nodeBg := lipgloss.NewStyle().Background(a.theme.Selected)

	// Group inline labels by line so we can apply right-to-left (preserving positions).
	type inlineLabel struct {
		lbl     string
		col     int // spanCol
		colEnd  int // spanColEnd
		nodeIdx int
	}
	lineLabels := make(map[int][]inlineLabel)

	for lbl, nodeIdx := range a.jumpLabels {
		if nodeIdx >= len(a.nodeRenderInfo) {
			continue
		}
		info := a.nodeRenderInfo[nodeIdx]
		if info.firstLine < 0 || info.firstLine >= len(out) {
			continue
		}

		node := a.codeNodes[nodeIdx]

		if node.inline && info.spanCol >= 0 {
			lineLabels[info.firstLine] = append(lineLabels[info.firstLine], inlineLabel{
				lbl: lbl, col: info.spanCol, colEnd: info.spanColEnd, nodeIdx: nodeIdx,
			})
		} else {
			// Block node: highlight all lines of the block and prepend label on first
			labelRendered := labelStyle.Render(lbl)
			firstLine := info.firstLine
			lastLine := info.lastLine
			if lastLine >= len(out) {
				lastLine = len(out) - 1
			}
			for li := firstLine; li <= lastLine; li++ {
				stripped := ansi.Strip(out[li])
				out[li] = nodeBg.Render(stripped)
			}
			out[firstLine] = labelRendered + " " + out[firstLine]
		}
	}

	// Apply inline labels per line, sorted right-to-left to preserve column positions.
	for lineIdx, labels := range lineLabels {
		// Sort by column descending
		sort.Slice(labels, func(i, j int) bool {
			return labels[i].col > labels[j].col
		})
		// Columns are display cells, so cut by width, not by rune index.
		line := ansi.Strip(lines[lineIdx])
		for _, il := range labels {
			w := ansi.StringWidth(line)
			col, end := min(il.col, w), min(il.colEnd, w)
			line = ansi.Cut(line, 0, col) + labelStyle.Render(il.lbl) +
				nodeBg.Render(ansi.Cut(line, col, end)) + ansi.TruncateLeft(line, end, "")
		}
		out[lineIdx] = line
	}

	return out
}

// For inline nodes the highlight is applied only to the span, preserving surrounding
// glamour styling. For fenced blocks the whole-line background is applied (appropriate
// since the block is the selectable unit).
func applyNodeHighlights(lines []string, nodes []codeNode, selIdx int, info []nodeRenderLoc, theme Theme, w int) []string {
	out := make([]string, len(lines))
	copy(out, lines)

	dimStyle := lipgloss.NewStyle().Background(theme.Code)
	selStyle := lipgloss.NewStyle().Background(theme.NodeSel).Foreground(theme.Background).Bold(true)

	// Apply available highlights first (lower priority), then selected (higher).
	for pass := 0; pass < 2; pass++ {
		for ni, n := range nodes {
			isSelected := ni == selIdx
			if pass == 0 && isSelected {
				continue
			}
			if pass == 1 && !isSelected {
				continue
			}
			loc := info[ni]
			if loc.firstLine < 0 {
				continue
			}
			style := dimStyle
			if isSelected {
				style = selStyle
			}

			if n.inline {
				// Surgical span highlight — only colour the backtick content.
				li := loc.firstLine
				if li < len(out) {
					out[li] = highlightSpanInLine(out[li], loc.spanCol, loc.spanColEnd, style)
				}
			} else {
				// Full-line background for all lines of the block.
				for li := loc.firstLine; li <= loc.lastLine && li < len(out); li++ {
					plain := ansi.Strip(out[li])
					visW := lipgloss.Width(plain)
					if visW < w {
						plain += strings.Repeat(" ", w-visW)
					}
					out[li] = style.Render(plain)
				}
			}
		}
	}
	return out
}

func (a *App) renderStatus() string {
	km := a.cfg.Keys
	// Build plain (unstyled) left/right text first so we can measure widths correctly.
	var leftPlain string
	switch a.mode {
	case ModeSearch:
		leftPlain = "/ " + a.searchQuery + "█"
	case ModeContentSearch:
		matchInfo := ""
		if len(a.contentSearchMatches) > 0 {
			matchInfo = fmt.Sprintf(" [%d/%d]", a.contentSearchIdx+1, len(a.contentSearchMatches))
		} else if a.contentSearchQuery != "" {
			matchInfo = " [0/0]"
		}
		leftPlain = "/ " + a.contentSearchQuery + "█" + matchInfo
	case ModeNodeSelect:
		filtered := a.filteredNodes()
		modeName := "CODE"
		switch a.nodeSubMode {
		case nodeInlineCode:
			modeName = "INLINE"
		case nodeLink:
			modeName = "LINKS"
		case nodeAll:
			modeName = "ALL"
		}
		if len(filtered) > 0 {
			leftPlain = fmt.Sprintf("%s [%d/%d]  %s:mode  %s:copy  %s/%s:nav  %s:exit",
				modeName, a.nodeSelIdx+1, len(filtered), config.FirstKey(km.NodeMode), config.FirstKey(km.NodeCopy),
				config.FirstKey(km.NodeNext), config.FirstKey(km.NodePrev), config.FirstKey(km.NodeExit))
		} else {
			leftPlain = fmt.Sprintf("%s [0/0]  %s:mode  %s:exit", modeName, config.FirstKey(km.NodeMode), config.FirstKey(km.NodeExit))
		}
		if a.statusMsg != "" {
			leftPlain += "  " + a.statusMsg
		}
	case ModeJump:
		leftPlain = "JUMP: type label to jump (Esc to cancel)"
		if a.jumpInput != "" {
			leftPlain = fmt.Sprintf("JUMP: %s…", a.jumpInput)
		}
	default:
		if a.selectedIdx < 0 {
			leftPlain = fmt.Sprintf("[0/%d] (Document)", len(a.doc.Headings))
		} else if a.selectedIdx < len(a.doc.Headings) {
			h := a.doc.Headings[a.selectedIdx]
			pos := fmt.Sprintf("[%d/%d]", a.selectedIdx+1, len(a.doc.Headings))
			heading := strings.Repeat("#", h.Level) + " " + h.Text
			leftPlain = pos + " " + heading
		}
		if a.statusMsg != "" {
			leftPlain += "  " + a.statusMsg
		}
	}

	var rightPlain string
	switch a.mode {
	case ModeNodeSelect:
		rightPlain = ""
	case ModeJump:
		rightPlain = ""
	default:
		if a.sidebarHidden {
			rightPlain = "w:sidebar  Tab:focus  e:edit  i:interactive  /:search  T:theme  ?:help  q:quit"
		} else {
			rightPlain = "Tab:focus  w:hide sidebar  e:edit  i:interactive  /:search  T:theme  ?:help  q:quit"
		}
	}

	// Use rune counts (no ANSI codes yet) for padding calculation.
	// Subtract 2 for the 1-cell padding on each side added by lipgloss.
	innerWidth := a.width - 2
	usedWidth := len([]rune(leftPlain)) + len([]rune(rightPlain))
	pad := innerWidth - usedWidth
	if pad < 1 {
		pad = 1
	}

	// Now apply colour styling.
	var content string
	switch a.mode {
	case ModeSearch:
		content = lipgloss.NewStyle().Foreground(a.theme.Search).Bold(true).Render(leftPlain) +
			strings.Repeat(" ", pad) + rightPlain
	case ModeNodeSelect:
		content = lipgloss.NewStyle().Foreground(a.theme.NodeSel).Bold(true).Render(leftPlain)
	case ModeJump:
		content = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5f00")).Bold(true).Render(leftPlain)
	default:
		content = leftPlain + strings.Repeat(" ", pad) + rightPlain
	}

	return lipgloss.NewStyle().
		Background(a.theme.StatusBar).
		Foreground(a.theme.Foreground).
		Width(a.width).
		Padding(0, 1).
		Render(content)
}

func (a *App) overlayHelp(background string) string {
	km := a.cfg.Keys
	var focusState string
	if a.sidebarHidden {
		focusState = "sidebar hidden (" + config.FirstKey(km.ToggleSidebar) + " to restore)"
	} else if a.focus == FocusSidebar {
		focusState = "sidebar focused"
	} else {
		focusState = "content focused"
	}

	// fmtKeys formats a key binding string for display:
	// "j,down" → "j / ↓", "ctrl+d,pgdown,space" → "Ctrl+D / PgDn / Space"
	fmtKeys := func(binding string) string {
		var parts []string
		for _, k := range strings.Split(binding, ",") {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			switch k {
			case "up":
				k = "↑"
			case "down":
				k = "↓"
			case "left":
				k = "←"
			case "right":
				k = "→"
			case "tab":
				k = "Tab"
			case "shift+tab":
				k = "Shift+Tab"
			case "enter":
				k = "Enter"
			case "esc":
				k = "Esc"
			case "pgup":
				k = "PgUp"
			case "pgdown":
				k = "PgDn"
			case "space":
				k = "Space"
			default:
				if strings.HasPrefix(k, "ctrl+") {
					k = "Ctrl+" + strings.ToUpper(k[5:])
				}
			}
			parts = append(parts, k)
		}
		return strings.Join(parts, " / ")
	}

	type helpEntry struct {
		keys string
		desc string
	}

	// Build sections from config.
	global := []helpEntry{
		{fmtKeys(km.ToggleFocus), "Toggle focus sidebar ↔ content"},
		{fmtKeys(km.ToggleSidebar), "Hide / show sidebar"},
		{fmtKeys(km.Edit), "Open file in $EDITOR"},
		{fmtKeys(km.Reload), "Reload file"},
		{fmtKeys(km.ThemePicker), "Open theme picker"},
		{fmtKeys(km.NavBack) + " / " + fmtKeys(km.NavForward), "Navigate back / forward"},
		{fmtKeys(km.Help), "Toggle this help"},
		{fmtKeys(km.Quit), "Quit"},
	}

	sidebar := []helpEntry{
		{fmtKeys(km.SidebarDown), "Select next heading"},
		{fmtKeys(km.SidebarUp), "Select previous heading"},
		{fmtKeys(km.ScrollHalfDown), "Half page down"},
		{fmtKeys(km.ScrollHalfUp), "Half page up"},
		{fmtKeys(km.SidebarTop) + " / " + fmtKeys(km.SidebarBottom), "Jump to root / last heading"},
		{fmtKeys(km.JumpHigh) + " / " + fmtKeys(km.JumpMid) + " / " + fmtKeys(km.JumpLow), "Top / mid / bottom of visible area"},
		{fmtKeys(km.ViewCenter) + " / " + fmtKeys(km.ViewTop) + " / " + fmtKeys(km.ViewBottom), "Center / top / bottom in viewport"},
		{fmtKeys(km.SidebarSearch), "Search headings"},
		{fmtKeys(km.NextMatch) + " / " + fmtKeys(km.PrevMatch), "Next / previous match"},
	}

	content := []helpEntry{
		{fmtKeys(km.ScrollDown), "Scroll down one line"},
		{fmtKeys(km.ScrollUp), "Scroll up one line"},
		{fmtKeys(km.ScrollHalfDown), "Half page down"},
		{fmtKeys(km.ScrollHalfUp), "Half page up"},
		{fmtKeys(km.ContentTop) + " / " + fmtKeys(km.ContentBottom), "Jump to top / bottom"},
		{fmtKeys(km.ContentSearch), "Search content"},
		{fmtKeys(km.ContentNextMatch) + " / " + fmtKeys(km.ContentPrevMatch), "Next / previous match"},
		{fmtKeys(km.NodeSelect), "Enter interactive mode"},
		{fmtKeys(km.Jump), "Jump mode (EasyMotion labels)"},
	}

	interactive := []helpEntry{
		{fmtKeys(km.NodeNext), "Next node"},
		{fmtKeys(km.NodePrev), "Previous node"},
		{fmtKeys(km.NodeMode), "Cycle sub-mode (all/code/inline/links)"},
		{fmtKeys(km.ScrollHalfDown) + " / " + fmtKeys(km.ScrollHalfUp), "Half page down / up"},
		{fmtKeys(km.JumpHigh) + " / " + fmtKeys(km.JumpMid) + " / " + fmtKeys(km.JumpLow), "Top / mid / bottom visible node"},
		{fmtKeys(km.ViewCenter) + " / " + fmtKeys(km.ViewTop) + " / " + fmtKeys(km.ViewBottom), "Center / top / bottom in viewport"},
		{fmtKeys(km.NodeCopy), "Copy to clipboard"},
		{fmtKeys(km.NodeOpen), "Open link"},
		{fmtKeys(km.NodeExit), "Exit interactive mode"},
	}

	// Find max key width across all sections for alignment.
	maxKeyW := 0
	for _, entries := range [][]helpEntry{global, sidebar, content, interactive} {
		for _, e := range entries {
			if w := len([]rune(e.keys)); w > maxKeyW {
				maxKeyW = w
			}
		}
	}

	renderSection := func(entries []helpEntry) string {
		var lines []string
		for _, e := range entries {
			pad := maxKeyW - len([]rune(e.keys)) + 2
			if pad < 2 {
				pad = 2
			}
			lines = append(lines, "    "+e.keys+strings.Repeat(" ", pad)+e.desc)
		}
		return strings.Join(lines, "\n")
	}

	text := fmt.Sprintf("gomd — Keyboard Shortcuts   [%s]\n\n", focusState)
	text += "  GLOBAL\n" + renderSection(global) + "\n\n"
	text += "  SIDEBAR  (when focused)\n" + renderSection(sidebar) + "\n\n"
	text += "  CONTENT  (when focused)\n" + renderSection(content) + "\n\n"
	text += "  INTERACTIVE  (press " + fmtKeys(km.NodeSelect) + " from content)\n" + renderSection(interactive) + "\n\n"
	text += "  Press any key to dismiss"

	// Compute modal width from content.
	modalW := 0
	for _, line := range strings.Split(text, "\n") {
		if w := lipgloss.Width(line); w > modalW {
			modalW = w
		}
	}
	modalW += 6 // padding (1 each side) + border (1 each side) + margin
	if modalW > a.width-4 {
		modalW = a.width - 4
	}

	return a.overlayModal(background, text, modalW)
}

// overlayModal draws text in a bordered box of width modalW centered over background.
func (a *App) overlayModal(background, text string, modalW int) string {
	// Styled segments end with a reset, which would show the terminal's own
	// background; re-apply the theme colours after each one.
	bgOpen := ansi.Style{}.ForegroundColor(a.theme.Foreground).BackgroundColor(a.theme.Background).String()
	text = strings.NewReplacer("\x1b[m", "\x1b[m"+bgOpen, "\x1b[0m", "\x1b[0m"+bgOpen).Replace(text)
	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(a.theme.Border).
		BorderBackground(a.theme.Background).
		Background(a.theme.Background).
		Foreground(a.theme.Foreground).
		Width(modalW). // lipgloss v2: width includes the border
		Padding(0, 1).
		Render(text)
	modalLines := strings.Split(modal, "\n")
	startRow := max((a.height-len(modalLines))/2, 0)
	startCol := max((a.width-modalW)/2, 0)

	// ANSI-aware overlay of modal onto background
	bgLines := strings.Split(background, "\n")
	for i, mLine := range modalLines {
		row := startRow + i
		if row >= len(bgLines) {
			break
		}
		left := ansi.Truncate(bgLines[row], startCol, "")
		right := ansi.TruncateLeft(bgLines[row], startCol+lipgloss.Width(mLine), "")
		bgLines[row] = left + mLine + right
	}
	return strings.Join(bgLines, "\n")
}
