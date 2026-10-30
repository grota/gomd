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

// contentBordered reports whether the content pane is drawn inside a box.
// Without one it keeps a blank column on each side, so its inner width is
// unchanged and only the two border rows go to content.
func (a *App) contentBordered() bool {
	return !a.sidebarHidden || a.cfg.UI.ContentBorder != config.ContentBorderWithSidebar
}

// outlineHeight is an alias kept for scrollOutlineToSelected.
func (a *App) outlineHeight() int { return a.paneInnerHeight() }

func (a *App) contentHeight() int {
	if a.contentBordered() {
		return a.paneInnerHeight()
	}
	return max(a.height-2, 0) // title(1) + status(1)
}

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

	// File name centred on the bar, breadcrumb on the left cut short of it.
	inner := max(a.width-2, 0) // Padding(0, 1)
	name = ansi.Truncate(name, inner, "…")
	start := (inner - lipgloss.Width(name)) / 2
	left := ""
	if bc := a.breadcrumb(); bc != "" && start > 1 {
		left = ansi.Truncate(bc, start-1, "…")
	}
	title := left + strings.Repeat(" ", start-lipgloss.Width(left)) + name

	return lipgloss.NewStyle().
		Background(a.theme.Border).
		Foreground(a.theme.textOn(a.theme.Border)).
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

	border := bc.Render("│")
	return top + "\n" + a.paintPaneLines(lines, innerW, border) + "\n" + bottom
}

// paintPaneLines fits each line to innerW columns on the theme background,
// wraps it in edge (a border or a blank cell) and joins the lines.
func (a *App) paintPaneLines(lines []string, innerW int, edge string) string {
	var sb strings.Builder
	bgStyle := lipgloss.NewStyle().
		Background(a.theme.Background).
		Foreground(a.theme.Foreground)
	// ANSI opening sequence for the theme colors, injected after every reset
	// in glamour output so the pane background survives.
	bgOpen := ansi.Style{}.ForegroundColor(a.theme.Foreground).BackgroundColor(a.theme.Background).String()

	for i, line := range lines {
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
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(edge + line + edge)
	}
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
	h := a.contentHeight()
	// Ensure exactly h lines
	for len(lines) < h {
		lines = append(lines, "")
	}
	lines = lines[:h]

	if !a.contentBordered() {
		pad := lipgloss.NewStyle().Background(a.theme.Background).Render(" ")
		return a.paintPaneLines(lines, innerW, pad)
	}
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
		lines = a.applyContentSearchHighlights(lines, a.contentOffset, a.contentOffset+h)
	}
	if a.sel != nil {
		lines = a.applySelectionHighlight(lines, a.contentOffset, a.contentOffset+h, w)
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

// placeNodesFromSource replaces the placement in locs, found by searching
// the rendered text, with the one the source map gives: the screen cells
// matched to the node's visible source characters. It keeps the search's
// placement for a node with no matched cell, and for HTML links, whose text
// and URL all map to the start of the tag. rendered is the section's
// rendered lines.
func placeNodesFromSource(m *sourceMap, nodes []codeNode, lines, rendered []string, locs []nodeRenderLoc) {
	off := make([]int, len(lines)+1) // byte offset of each line in the section
	for i, l := range lines {
		off[i+1] = off[i] + len(l) + 1
	}
	for i, n := range nodes {
		var start, end int
		switch line := lines[n.startLine]; {
		case n.kind == nodeCodeBlock && n.indented:
			start, end = off[n.startLine], off[n.endLine+1] // no fence lines
		case n.kind == nodeCodeBlock:
			start, end = off[n.startLine+1], off[n.endLine] // the body
		case n.kind == nodeInlineCode:
			start, end = off[n.startLine]+n.colStart, off[n.startLine]+n.colEnd
		case strings.HasPrefix(line[n.colStart:], "!["): // the caption: alt and path
			start, end = off[n.startLine]+n.colStart+2, off[n.startLine]+n.colEnd
		case line[n.colStart] == '[': // the text, not the URL
			start = off[n.startLine] + n.colStart + 1
			end = start + len(n.display)
		case len(line) > n.colStart+2 && strings.EqualFold(line[n.colStart:n.colStart+2], "<a") &&
			(line[n.colStart+2] == ' ' || line[n.colStart+2] == '\t'): // the text between the tags
			m := aTagRe.FindStringSubmatchIndex(line[n.colStart:n.colEnd])
			if m == nil {
				continue
			}
			start, end = off[n.startLine]+n.colStart+m[4], off[n.startLine]+n.colStart+m[5]
		default: // <img>, autolinks and bare URLs
			start, end = off[n.startLine]+n.colStart, off[n.startLine]+n.colEnd
		}
		first, last, col0, col1, ok := m.place(start, end)
		if !ok {
			continue
		}
		locs[i].firstLine, locs[i].lastLine = first, last
		if n.inline {
			// Glamour shows a relative image path rooted ("a.png" as
			// "/a.png"); that "/" has no source, so take it in.
			if strings.HasPrefix(n.display, "/") && col0 > 0 && ansi.Cut(ansi.Strip(rendered[first]), col0-1, col0) == "/" {
				col0--
			}
			locs[i].spanCol, locs[i].spanColEnd = col0, col1
		}
	}
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
					// Rendered headings keep their "##", and their words must not
					// match spans of later lines; a span on the heading itself may.
					if bestCol < 0 && (n.heading || !strings.Contains(stripped[ri], "##")) {
						// Short needles (1-2 chars) may also follow a single space,
						// but so do prose words ("x" before `x`): on each line, a
						// match with the full padding wins over a single space.
						strictPasses := []bool{true}
						if len(needle) < 3 {
							strictPasses = append(strictPasses, false)
						}
						for _, strict := range strictPasses {
							for pos := 0; pos < len(s); pos++ {
								idx := strings.Index(s[pos:], needle)
								if idx == -1 {
									break
								}
								absIdx := colOffset + pos + idx
								endIdx := absIdx + len(needle)
								fullLine := stripped[ri]
								// Require 2 spaces before to distinguish code spans from
								// prose. Exception: also accept 1 space before when preceded
								// by punctuation (glamour places single space after opening
								// parens/commas/pipes).
								var hasPre bool
								if strict {
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
							if bestCol >= 0 {
								break
							}
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
			bodyLeft := 0 // body lines from the anchor to the end
			bodyLines := strings.Split(n.content, "\n")
			for i, bodyLine := range bodyLines {
				t := strings.TrimSpace(bodyLine)
				if t != "" {
					needle = t
					bodyLeft = len(bodyLines) - i
					break
				}
			}
			// Every block has text (empty ones are not nodes), so needle is
			// set. Glamour hides the fences, so there is nothing else to find.
			firstLine := -1
			for ri := blockSearchFromLine; ri < len(stripped); ri++ {
				if strings.Contains(stripped[ri], needle) {
					firstLine = ri
					break
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
				// The next code block is indented the same way, so the scan
				// runs into it. A block shows at least one line per body line,
				// so it ends no later than that count: this only shortens the
				// range. (A wrapped long line loses the indent and already ends
				// the scan early; the source map would place blocks exactly.)
				if bodyLeft > 0 {
					lastLine = min(lastLine, firstLine+bodyLeft-1)
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
	return highlightSpans(line, []highlightSpan{{colStart, colEnd, style}})
}

// highlightSpan is a display-column range [colStart, colEnd) of a rendered
// line and the style to paint it with.
type highlightSpan struct {
	colStart, colEnd int
	style            lipgloss.Style
}

// highlightSpans paints spans, sorted by colStart and not overlapping, over a
// rendered line in one pass. Text inside a span is drawn plain in the span's
// style; after it the SGR state in effect at that point is restored. Other
// sequences inside a span (OSC 8 links) are moved to just after it.
func highlightSpans(line string, spans []highlightSpan) string {
	var out, sgr, mid, held strings.Builder // sgr: SGR sequences since the last reset
	var state byte
	col, si, in := 0, 0, false
	closeSpan := func() {
		out.WriteString(spans[si].style.Render(mid.String()))
		out.WriteString(held.String())
		out.WriteString(sgr.String())
		mid.Reset()
		held.Reset()
		in = false
		si++
	}
	for len(line) > 0 {
		seq, w, n, newState := ansi.DecodeSequence(line, state, nil)
		state = newState
		line = line[n:]
		for !in && si < len(spans) && (spans[si].colStart < 0 || spans[si].colEnd <= max(spans[si].colStart, col)) {
			si++ // empty, invalid, or already behind
		}
		if !in && si < len(spans) && w > 0 && col >= spans[si].colStart {
			in = true
		}
		switch {
		case !strings.HasPrefix(seq, "\x1b"):
			if in {
				mid.WriteString(seq)
			} else {
				out.WriteString(seq)
			}
			col += w
			if in && col >= spans[si].colEnd {
				closeSpan()
			}
		case strings.HasPrefix(seq, "\x1b[") && strings.HasSuffix(seq, "m"):
			if seq == "\x1b[m" || seq == "\x1b[0m" || strings.HasPrefix(seq, "\x1b[0;") {
				sgr.Reset()
			}
			sgr.WriteString(seq)
			if !in {
				out.WriteString(seq)
			}
		default:
			if in {
				held.WriteString(seq)
			} else {
				out.WriteString(seq)
			}
		}
	}
	if in {
		closeSpan()
	}
	return out.String()
}

// applyContentSearchHighlights highlights the content search matches on
// lines [from, to), the ones on screen.
func (a *App) applyContentSearchHighlights(lines []string, from, to int) []string {
	out := make([]string, len(lines))
	copy(out, lines)

	// The current match is filled with the theme's highlight colour, the
	// others with a dimmer blend of it into the background.
	t := a.theme
	dim := mix(t.Highlight, t.Background, 0.6)
	matchStyle := lipgloss.NewStyle().Background(dim).Foreground(t.textOn(dim))
	currentStyle := lipgloss.NewStyle().Background(t.Highlight).Foreground(t.textOn(t.Highlight)).Bold(true)

	// Matches are ordered by line, then column, so each line's spans are
	// collected in order and painted in one pass.
	var spans []highlightSpan
	for i, m := range a.contentSearchMatches {
		if m.line < from || m.line >= to || m.line >= len(out) {
			continue
		}
		style := matchStyle
		if i == a.contentSearchIdx {
			style = currentStyle
		}
		spans = append(spans, highlightSpan{m.colStart, m.colEnd, style})
		if i+1 == len(a.contentSearchMatches) || a.contentSearchMatches[i+1].line != m.line {
			out[m.line] = highlightSpans(out[m.line], spans)
			spans = spans[:0]
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

	dimStyle := lipgloss.NewStyle().Background(theme.Code).Foreground(theme.textOn(theme.Code))
	selStyle := lipgloss.NewStyle().Background(theme.NodeSel).Foreground(theme.textOn(theme.NodeSel)).Bold(true)

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
		hint := func(binding, label string) string { return config.FirstKey(binding) + ":" + label }
		hints := []string{hint(km.ToggleSidebar, "sidebar"), hint(km.ToggleFocus, "focus")}
		if !a.sidebarHidden {
			hints = []string{hint(km.ToggleFocus, "focus"), hint(km.ToggleSidebar, "hide sidebar")}
		}
		hints = append(hints, hint(km.Edit, "edit"), hint(km.NodeSelect, "interactive"),
			hint(km.ContentSearch, "search"), hint(km.ThemePicker, "theme"), hint(km.Help, "help"), hint(km.Quit, "quit"))
		rightPlain = strings.Join(hints, "  ")
	}

	// Measure before styling. Subtract 2 for the 1-cell padding on each side
	// added by lipgloss.
	innerWidth := a.width - 2
	usedWidth := lipgloss.Width(leftPlain) + lipgloss.Width(rightPlain)
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
	// Cut what does not fit: a wrapped status bar would push the view past
	// the bottom of the terminal.
	content = ansi.Truncate(content, max(innerWidth, 0), "…")

	return lipgloss.NewStyle().
		Background(a.theme.StatusBar).
		Foreground(a.theme.textOn(a.theme.StatusBar)).
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
		{fmtKeys(km.Edit), "Open file in $EDITOR at the shown line or search match"},
		{fmtKeys(km.Reload), "Reload file"},
		{fmtKeys(km.ThemePicker), "Open theme picker"},
		{fmtKeys(km.NavBack) + " / " + fmtKeys(km.NavForward), "Navigate back / forward"},
		{"Mouse drag", "Select content text and copy it"},
		{fmtKeys(km.CopyMarkdown), "Copy the selection's markdown source"},
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
