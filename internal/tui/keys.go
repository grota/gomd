// Package tui provides the interactive terminal user interface for gomd.
package tui

import (
	"fmt"
	"regexp"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/grota/gomd/internal/config"
)

// ─────────────────────────────────────────────
// Key handling — dispatched by mode then focus
// ─────────────────────────────────────────────

func (a *App) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	if a.sel != nil && !a.sel.dragging && a.mode == ModeNormal && config.KeyMatches(k, a.cfg.Keys.CopyMarkdown) {
		text := a.selectedMarkdown()
		if text == "" {
			a.statusMsg = "No markdown to copy"
			return a, nil
		}
		return a, copyCmd(text, fmt.Sprintf("Copied %d characters of markdown", len([]rune(text))), "", "")
	}
	a.sel = nil

	// Global keys that work in any mode
	switch k {
	case "ctrl+c":
		if a.watcher != nil {
			a.watcher.Close()
		}
		return a, tea.Quit
	}

	switch a.mode {
	case ModeSearch:
		return a.handleSearchKey(msg)
	case ModeContentSearch:
		return a.handleContentSearchKey(msg)
	case ModeHelp:
		a.mode = ModeNormal
		return a, nil
	case ModeThemePicker:
		return a.handleThemeKey(msg)
	case ModeNodeSelect:
		return a.handleNodeSelectKey(msg)
	case ModeJump:
		return a.handleJumpKey(msg)
	}

	// Normal mode — shared keys regardless of focus
	// If a shared key matches, cancel any pending key sequence.
	km := a.cfg.Keys
	switch {
	case config.KeyMatches(k, km.Quit):
		a.pendingKey = ""
		if a.watcher != nil {
			a.watcher.Close()
		}
		return a, tea.Quit
	case config.KeyMatches(k, km.Help):
		a.mode = ModeHelp
		return a, nil
	case config.KeyMatches(k, km.ThemePicker):
		return a, a.openThemePicker()
	case config.KeyMatches(k, km.ToggleFocus):
		a.toggleFocus()
		return a, nil
	case config.KeyMatches(k, km.ToggleSidebar):
		a.sidebarHidden = !a.sidebarHidden
		if a.sidebarHidden {
			a.focus = FocusContent
		}
		return a, nil
	case config.KeyMatches(k, km.Reload):
		a.reloadFromDisk("Reloaded")
		return a, nil
	case config.KeyMatches(k, km.Edit):
		if a.filepath != "" {
			return a, openEditorCmd(a.filepath)
		}
		a.statusMsg = "No file to edit"
		return a, nil
	case config.KeyMatches(k, km.NavBack):
		a.navGoBack()
		return a, nil
	case config.KeyMatches(k, km.NavForward):
		a.navGoForward()
		return a, nil
	}

	// Focus-specific keys
	if a.focus == FocusSidebar && !a.sidebarHidden {
		return a.handleSidebarKey(msg)
	}
	return a.handleContentKey(msg)
}

func (a *App) toggleFocus() {
	if a.sidebarHidden {
		return
	}
	if a.focus == FocusSidebar {
		a.focus = FocusContent
	} else {
		a.focus = FocusSidebar
	}
}

// handleMouseWheel handles mouse wheel events for content scrolling.
func (a *App) handleMouseWheel(msg tea.MouseWheelMsg) (tea.Model, tea.Cmd) {
	if a.mode != ModeNormal {
		return a, nil
	}

	switch msg.Button {
	case tea.MouseWheelUp:
		if a.focus == FocusContent || a.sidebarHidden {
			a.contentOffset -= 3
			if a.contentOffset < 0 {
				a.contentOffset = 0
			}
		}
	case tea.MouseWheelDown:
		if a.focus == FocusContent || a.sidebarHidden {
			a.contentOffset += 3
			a.clampContentOffset()
		}
	}
	return a, nil
}

// handleMouseClick selects a clicked sidebar heading, or starts a selection
// in the content pane (see selection.go).
func (a *App) handleMouseClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if a.mode != ModeNormal {
		return a, nil
	}

	if msg.Button == tea.MouseLeft {
		mouse := msg.Mouse()
		a.sel = nil
		if a.inContent(mouse.X, mouse.Y) {
			p := a.contentPosAt(mouse.X, mouse.Y)
			a.sel = &selection{anchor: p, head: p, dragging: true}
		}
		// Check if click is in the sidebar
		sw := a.sidebarWidth()
		if !a.sidebarHidden && mouse.X < sw && mouse.Y >= 2 && mouse.Y < a.height-1 {
			// Click in sidebar — determine which entry was clicked
			row := mouse.Y - 2 // subtract title(1) + border-top(1)
			entryIdx := a.outlineOffset + row
			if entryIdx >= 0 && entryIdx < a.totalEntries() {
				newIdx := entryIdx - 1 // entryIdx 0 = root (-1)
				a.navigateTo(newIdx)
				a.focus = FocusSidebar
			}
		} else if mouse.X >= sw {
			a.focus = FocusContent
		}
	}
	return a, nil
}

// ─────────────────────────────────────────────
// Sidebar keys
// ─────────────────────────────────────────────

func (a *App) handleSidebarKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	km := a.cfg.Keys

	// Handle pending key sequences (gg, zz, zt, zb)
	if a.pendingKey != "" {
		combo := a.pendingKey + k
		a.pendingKey = ""
		switch {
		case config.KeyMatches(combo, km.SidebarTop):
			a.selectedIdx = -1
			a.scrollOutlineToSelected()
			a.rebuildSection()
		case config.KeyMatches(combo, km.ViewCenter):
			a.sidebarViewCenter()
		case config.KeyMatches(combo, km.ViewTop):
			a.sidebarViewTop()
		case config.KeyMatches(combo, km.ViewBottom):
			a.sidebarViewBottom()
		}
		return a, nil
	}

	if config.IsSequencePrefix(k, km.SidebarTop, km.ViewCenter, km.ViewTop, km.ViewBottom) {
		a.pendingKey = k
		return a, nil
	}

	switch {
	case config.KeyMatches(k, km.SidebarDown):
		a.moveSidebarDown(1)
	case config.KeyMatches(k, km.SidebarUp):
		a.moveSidebarUp(1)
	case config.KeyMatches(k, km.ScrollHalfDown):
		a.moveSidebarDown(max(a.outlineHeight()/2, 1))
	case config.KeyMatches(k, km.ScrollHalfUp):
		a.moveSidebarUp(max(a.outlineHeight()/2, 1))
	case config.KeyMatches(k, km.SidebarBottom):
		if len(a.doc.Headings) > 0 {
			a.selectedIdx = len(a.doc.Headings) - 1
		}
		a.scrollOutlineToSelected()
		a.rebuildSection()
	case config.KeyMatches(k, km.JumpHigh):
		a.sidebarJumpHigh()
	case config.KeyMatches(k, km.JumpMid):
		a.sidebarJumpMid()
	case config.KeyMatches(k, km.JumpLow):
		a.sidebarJumpLow()
	case config.KeyMatches(k, km.SidebarSearch):
		a.mode = ModeSearch
		a.searchQuery = ""
		a.searchMatches = nil
	case config.KeyMatches(k, km.NextMatch):
		a.nextSearchMatch()
	case config.KeyMatches(k, km.PrevMatch):
		a.prevSearchMatch()
	}
	return a, nil
}

func (a *App) moveSidebarDown(n int) {
	prev := a.selectedIdx
	a.selectedIdx += n
	if a.selectedIdx >= len(a.doc.Headings) {
		a.selectedIdx = len(a.doc.Headings) - 1
	}
	if a.selectedIdx != prev {
		a.scrollOutlineToSelected()
		a.rebuildSection()
	}
}

func (a *App) moveSidebarUp(n int) {
	prev := a.selectedIdx
	a.selectedIdx -= n
	if a.selectedIdx < -1 {
		a.selectedIdx = -1
	}
	if a.selectedIdx != prev {
		a.scrollOutlineToSelected()
		a.rebuildSection()
	}
}

// Sidebar rows are display rows: row 0 is the "(Document)" root, row r is
// heading r-1. outlineOffset is the first visible row.

// sidebarSelectRow selects display row r, clamped to the visible window.
func (a *App) sidebarSelectRow(r int) {
	r = min(r, a.outlineOffset+a.outlineHeight()-1, a.totalEntries()-1)
	r = max(r, a.outlineOffset, 0)
	a.selectedIdx = r - 1
	a.rebuildSection()
}

// sidebarJumpHigh jumps to the topmost visible sidebar entry.
func (a *App) sidebarJumpHigh() { a.sidebarSelectRow(a.outlineOffset) }

// sidebarJumpMid jumps to the middle visible sidebar entry.
func (a *App) sidebarJumpMid() {
	bot := min(a.outlineOffset+a.outlineHeight()-1, a.totalEntries()-1)
	a.sidebarSelectRow((a.outlineOffset + bot) / 2)
}

// sidebarJumpLow jumps to the bottommost visible sidebar entry.
func (a *App) sidebarJumpLow() { a.sidebarSelectRow(a.outlineOffset + a.outlineHeight() - 1) }

// sidebarViewCenter scrolls the sidebar so the selected entry is centered.
func (a *App) sidebarViewCenter() {
	a.outlineOffset = a.displayIdx() - a.outlineHeight()/2
	a.clampOutlineOffset()
}

// sidebarViewTop scrolls the sidebar so the selected entry is at the top.
func (a *App) sidebarViewTop() {
	a.outlineOffset = a.displayIdx()
	a.clampOutlineOffset()
}

// sidebarViewBottom scrolls the sidebar so the selected entry is at the bottom.
func (a *App) sidebarViewBottom() {
	a.outlineOffset = a.displayIdx() - a.outlineHeight() + 1
	a.clampOutlineOffset()
}

// clampOutlineOffset keeps outlineOffset within the scrollable range.
func (a *App) clampOutlineOffset() {
	a.outlineOffset = max(min(a.outlineOffset, a.totalEntries()-a.outlineHeight()), 0)
}

// navigateTo changes the selected heading with history tracking.
func (a *App) navigateTo(idx int) {
	if idx == a.selectedIdx {
		return
	}
	a.navHistory = append(a.navHistory, a.selectedIdx)
	a.navFuture = nil
	a.selectedIdx = idx
	a.scrollOutlineToSelected()
	a.rebuildSection()
}

// navGoBack navigates to the previous heading in history.
func (a *App) navGoBack() {
	if len(a.navHistory) == 0 {
		return
	}
	a.navFuture = append(a.navFuture, a.selectedIdx)
	a.selectedIdx = a.navHistory[len(a.navHistory)-1]
	a.navHistory = a.navHistory[:len(a.navHistory)-1]
	a.scrollOutlineToSelected()
	a.rebuildSection()
}

// navGoForward navigates forward in history.
func (a *App) navGoForward() {
	if len(a.navFuture) == 0 {
		return
	}
	a.navHistory = append(a.navHistory, a.selectedIdx)
	a.selectedIdx = a.navFuture[len(a.navFuture)-1]
	a.navFuture = a.navFuture[:len(a.navFuture)-1]
	a.scrollOutlineToSelected()
	a.rebuildSection()
}

// findHeadingByAnchor finds a heading index by GitHub-style anchor slug.
func (a *App) findHeadingByAnchor(anchor string) int {
	anchor = strings.ToLower(anchor)
	for i, h := range a.doc.Headings {
		if headingToAnchor(h.Text) == anchor {
			return i
		}
	}
	return -1
}

// headingToAnchor converts heading text to a GitHub-style anchor slug.
func headingToAnchor(text string) string {
	text = strings.ToLower(text)
	var sb strings.Builder
	for _, r := range text {
		if r == ' ' || r == '-' {
			sb.WriteByte('-')
		} else if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// displayIdx returns the sidebar display index for the current selection.
// Root (selectedIdx==-1) → 0; heading i → i+1.
func (a *App) displayIdx() int {
	return a.selectedIdx + 1
}

// totalEntries returns the total number of sidebar entries (root + headings).
func (a *App) totalEntries() int {
	return len(a.doc.Headings) + 1
}

// scrollOutlineToSelected scrolls the outline viewport so the selected item is
// always visible, but does NOT move the viewport unless the selection has left it.
func (a *App) scrollOutlineToSelected() {
	h := a.outlineHeight()
	if h <= 0 {
		return
	}
	di := a.displayIdx()
	if di < a.outlineOffset {
		a.outlineOffset = di
	}
	if di >= a.outlineOffset+h {
		a.outlineOffset = di - h + 1
	}
	if a.outlineOffset < 0 {
		a.outlineOffset = 0
	}
}

// ─────────────────────────────────────────────
// Content keys
// ─────────────────────────────────────────────

func (a *App) handleContentKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	km := a.cfg.Keys

	// Handle pending key sequences (gg, zz, zt, zb)
	if a.pendingKey != "" {
		combo := a.pendingKey + k
		a.pendingKey = ""
		switch {
		case config.KeyMatches(combo, km.ContentTop):
			a.contentOffset = 0
		}
		return a, nil
	}

	if config.IsSequencePrefix(k, km.ContentTop) {
		a.pendingKey = k
		return a, nil
	}

	switch {
	case config.KeyMatches(k, km.ScrollDown):
		a.scrollContent(1)
	case config.KeyMatches(k, km.ScrollUp):
		a.scrollContent(-1)
	case config.KeyMatches(k, km.ScrollHalfDown):
		a.scrollContent(a.contentHeight() / 2)
	case config.KeyMatches(k, km.ScrollHalfUp):
		a.scrollContent(-a.contentHeight() / 2)
	case config.KeyMatches(k, km.ContentBottom):
		a.contentOffset = len(a.activeLines())
		a.clampContentOffset()
	case config.KeyMatches(k, km.NodeSelect):
		a.enterNodeSelectMode()
	case config.KeyMatches(k, km.ContentSearch):
		a.mode = ModeContentSearch
		a.contentSearchQuery = ""
		a.contentSearchMatches = nil
		a.contentSearchIdx = 0
	case config.KeyMatches(k, km.ContentNextMatch):
		a.nextContentSearchMatch()
	case config.KeyMatches(k, km.ContentPrevMatch):
		a.prevContentSearchMatch()
	case config.KeyMatches(k, km.Jump):
		a.enterJumpMode()
	}
	return a, nil
}

func (a *App) scrollContent(delta int) {
	a.contentOffset += delta
	a.clampContentOffset()
}

func (a *App) activeLines() []string {
	if len(a.renderedLines) == 0 {
		return a.sectionLines
	}
	return a.renderedLines
}

func (a *App) clampContentOffset() {
	if a.contentOffset < 0 {
		a.contentOffset = 0
	}
	max := len(a.activeLines()) - a.contentHeight()
	if max < 0 {
		max = 0
	}
	if a.contentOffset > max {
		a.contentOffset = max
	}
}

// ─────────────────────────────────────────────
// Node selection mode
// ─────────────────────────────────────────────

// enterNodeSelectMode transitions to ModeNodeSelect, picking the first sub-mode
// that has nodes and scrolling to the first visible node at the current offset.
func (a *App) enterNodeSelectMode() {
	if len(a.codeNodes) == 0 {
		a.statusMsg = "No selectable nodes in this section"
		return
	}

	a.mode = ModeNodeSelect
	a.nodeSubMode = nodeAll
	a.nodeSelIdx = 0

	// Find the first sub-mode that has nodes
	if len(a.filteredNodeIndices()) == 0 {
		for _, mode := range []nodeKind{nodeCodeBlock, nodeInlineCode, nodeLink} {
			a.nodeSubMode = mode
			if len(a.filteredNodeIndices()) > 0 {
				break
			}
		}
	}

	indices := a.filteredNodeIndices()
	if len(indices) == 0 {
		a.mode = ModeNormal
		a.statusMsg = "No selectable nodes in this section"
		return
	}

	// Find the first filtered node visible at the current scroll position
	a.ensureRenderedLines()
	for i, idx := range indices {
		if a.nodeRenderInfo != nil && idx < len(a.nodeRenderInfo) && a.nodeRenderInfo[idx].firstLine >= a.contentOffset {
			a.nodeSelIdx = i
			break
		}
	}

	a.scrollToFilteredNode(a.nodeSelIdx)
	a.statusMsg = ""
}

// enterJumpMode activates EasyMotion-style jump labels for visible selectable nodes.
func (a *App) enterJumpMode() {
	if len(a.codeNodes) == 0 {
		a.statusMsg = "No selectable nodes in this section"
		return
	}
	a.ensureRenderedLines()

	// Find nodes visible on screen
	visibleStart := a.contentOffset
	visibleEnd := a.contentOffset + a.contentHeight()

	var visibleNodes []int
	for i, info := range a.nodeRenderInfo {
		if info.firstLine >= visibleStart && info.firstLine < visibleEnd {
			visibleNodes = append(visibleNodes, i)
		}
	}

	if len(visibleNodes) == 0 {
		a.statusMsg = "No selectable nodes visible"
		return
	}

	// Generate single-char labels only for visible nodes (max 24)
	labels := generateLabels(len(visibleNodes))
	a.jumpLabels = make(map[string]int, len(labels))
	for i, lbl := range labels {
		a.jumpLabels[lbl] = visibleNodes[i]
	}
	a.jumpInput = ""
	a.mode = ModeJump
}

// generateLabels produces unique single-character labels (a-z, skipping f and q).
func generateLabels(n int) []string {
	chars := "abcdeghijklmnoprstuvwxyz" // 24 chars, skip f and q
	if n > len(chars) {
		n = len(chars)
	}
	labels := make([]string, n)
	for i := 0; i < n; i++ {
		labels[i] = string(chars[i])
	}
	return labels
}

// handleJumpKey processes input during jump mode.
func (a *App) handleJumpKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	switch k {
	case "esc":
		a.mode = ModeNormal
		a.jumpLabels = nil
		return a, nil
	}

	if len(msg.Text) == 0 {
		return a, nil
	}

	a.jumpInput += string(msg.Text)

	// Check for exact match
	if idx, ok := a.jumpLabels[a.jumpInput]; ok {
		// Jump to the node — enter node-select mode with "all" sub-mode
		a.mode = ModeNodeSelect
		a.nodeSubMode = nodeAll
		a.jumpLabels = nil
		a.jumpInput = ""

		// Find the position in filtered list
		indices := a.filteredNodeIndices()
		for i, nodeIdx := range indices {
			if nodeIdx == idx {
				a.nodeSelIdx = i
				break
			}
		}
		a.scrollToFilteredNode(a.nodeSelIdx)
		return a, nil
	}

	// Check if any label starts with current input (partial match)
	hasPrefix := false
	for lbl := range a.jumpLabels {
		if strings.HasPrefix(lbl, a.jumpInput) {
			hasPrefix = true
			break
		}
	}
	if !hasPrefix {
		// No possible match — cancel
		a.mode = ModeNormal
		a.jumpLabels = nil
		a.jumpInput = ""
	}

	return a, nil
}

// ─────────────────────────────────────────────
// Node selection mode keys
// ─────────────────────────────────────────────

func (a *App) handleNodeSelectKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	km := a.cfg.Keys
	filtered := a.filteredNodes()
	a.statusMsg = "" // feedback from the previous action lasts until the next key

	// Handle pending key sequences (gg, zz, zt, zb)
	if a.pendingKey != "" {
		combo := a.pendingKey + k
		a.pendingKey = ""
		switch {
		case config.KeyMatches(combo, km.ViewCenter):
			a.nodeSelectViewAt(a.contentHeight() / 2)
		case config.KeyMatches(combo, km.ViewTop):
			a.nodeSelectViewAt(0)
		case config.KeyMatches(combo, km.ViewBottom):
			a.nodeSelectViewAt(a.contentHeight() - 1)
		}
		return a, nil
	}

	if config.IsSequencePrefix(k, km.ViewCenter, km.ViewTop, km.ViewBottom) {
		a.pendingKey = k
		return a, nil
	}

	switch {
	case config.KeyMatches(k, km.NodeExit):
		a.mode = ModeNormal
	case config.KeyMatches(k, km.NodeMode):
		a.nodeSubMode = (a.nodeSubMode + 1) % nodeKindCount
		a.nodeSelIdx = 0
		a.scrollToFilteredNode(0)
	case config.KeyMatches(k, km.NodeNext):
		if len(filtered) > 0 {
			a.nodeSelIdx = (a.nodeSelIdx + 1) % len(filtered)
			a.scrollToFilteredNode(a.nodeSelIdx)
		}
	case config.KeyMatches(k, km.NodePrev):
		if len(filtered) > 0 {
			a.nodeSelIdx = (a.nodeSelIdx - 1 + len(filtered)) % len(filtered)
			a.scrollToFilteredNode(a.nodeSelIdx)
		}
	case config.KeyMatches(k, km.JumpHigh):
		a.nodeSelectJumpHigh()
	case config.KeyMatches(k, km.JumpMid):
		a.nodeSelectJumpMid()
	case config.KeyMatches(k, km.JumpLow):
		a.nodeSelectJumpLow()
	case config.KeyMatches(k, km.NodeCopy):
		if len(filtered) > 0 {
			node := filtered[a.nodeSelIdx]
			lang := node.lang
			if lang == "" {
				lang = "span"
			}
			status := fmt.Sprintf("✓ Copied %s block (%d lines)", lang, strings.Count(node.content, "\n")+1)
			if node.kind == nodeLink {
				status = fmt.Sprintf("✓ Copied link: %s", node.content)
			}
			if a.nodeSubMode != nodeAll && a.nodeSubMode != nodeLink {
				a.mode = ModeNormal
			}
			return a, copyCmd(node.content, status, "", "")
		}
	case config.KeyMatches(k, km.NodeOpen) && (a.nodeSubMode == nodeLink || (a.nodeSubMode == nodeAll && len(filtered) > 0 && filtered[a.nodeSelIdx].kind == nodeLink)):
		if len(filtered) > 0 {
			node := filtered[a.nodeSelIdx]
			url := node.content
			if url == "" {
				break
			}
			// Internal anchor link — jump to heading
			if strings.HasPrefix(url, "#") {
				anchor := url[1:]
				if idx := a.findHeadingByAnchor(anchor); idx >= 0 {
					a.mode = ModeNormal
					a.navigateTo(idx)
					a.statusMsg = fmt.Sprintf("Jumped to: %s", a.doc.Headings[idx].Text)
				} else {
					a.statusMsg = fmt.Sprintf("Heading not found: %s", anchor)
				}
			} else {
				// External link — open with configured opener
				a.statusMsg = a.openURL(url)
			}
		}
	case config.KeyMatches(k, km.ScrollHalfDown):
		a.nodeSelectScroll(a.contentHeight() / 2)
	case config.KeyMatches(k, km.ScrollHalfUp):
		a.nodeSelectScroll(-a.contentHeight() / 2)
	}
	return a, nil
}

// visibleFilteredNode returns the position in filteredNodeIndices of the
// first (or, with fromBottom, last) node whose first line is in the viewport.
func (a *App) visibleFilteredNode(fromBottom bool) (int, bool) {
	indices := a.filteredNodeIndices()
	start, end := a.contentOffset, a.contentOffset+a.contentHeight()
	for n := range indices {
		i := n
		if fromBottom {
			i = len(indices) - 1 - n
		}
		if idx := indices[i]; idx < len(a.nodeRenderInfo) && a.nodeRenderInfo[idx].firstLine >= start && a.nodeRenderInfo[idx].firstLine < end {
			return i, true
		}
	}
	return 0, false
}

func (a *App) nodeSelectScroll(delta int) {
	a.contentOffset += delta
	a.clampContentOffset()
	n := len(a.filteredNodeIndices())
	if n == 0 {
		return
	}
	// Scrolling down picks the topmost visible node, up the bottommost; with
	// none visible, fall back to the last or first node.
	if i, ok := a.visibleFilteredNode(delta < 0); ok {
		a.nodeSelIdx = i
	} else if delta > 0 {
		a.nodeSelIdx = n - 1
	} else {
		a.nodeSelIdx = 0
	}
}

func (a *App) nodeSelectJumpHigh() {
	if i, ok := a.visibleFilteredNode(false); ok {
		a.nodeSelIdx = i
	}
}

// nodeSelectJumpMid selects the node closest to the middle of the viewport.
func (a *App) nodeSelectJumpMid() {
	indices := a.filteredNodeIndices()
	if len(indices) == 0 {
		return
	}
	midLine := a.contentOffset + a.contentHeight()/2
	bestDist := -1
	bestIdx := 0
	for i, idx := range indices {
		if idx < len(a.nodeRenderInfo) && a.nodeRenderInfo[idx].firstLine >= 0 {
			dist := a.nodeRenderInfo[idx].firstLine - midLine
			if dist < 0 {
				dist = -dist
			}
			if bestDist < 0 || dist < bestDist {
				bestDist = dist
				bestIdx = i
			}
		}
	}
	a.nodeSelIdx = bestIdx
	a.scrollToFilteredNode(a.nodeSelIdx)
}

func (a *App) nodeSelectJumpLow() {
	if i, ok := a.visibleFilteredNode(true); ok {
		a.nodeSelIdx = i
	}
}

// nodeSelectViewAt scrolls so the selected node's first line sits at row
// `row` of the viewport.
func (a *App) nodeSelectViewAt(row int) {
	indices := a.filteredNodeIndices()
	if a.nodeSelIdx >= len(indices) {
		return
	}
	idx := indices[a.nodeSelIdx]
	if idx >= len(a.nodeRenderInfo) || a.nodeRenderInfo[idx].firstLine < 0 {
		return
	}
	a.contentOffset = a.nodeRenderInfo[idx].firstLine - row
	a.clampContentOffset()
}

// ensureRenderedLines builds renderedLines and nodeRenderInfo if they are
// stale or missing. Called eagerly before any operation that needs them
// (e.g. scrollContentToNode) so that the data is ready before View().
func (a *App) ensureRenderedLines() {
	w := a.contentWidth() - 2
	if w < 1 {
		w = 1
	}
	if a.renderedLinesIdx == a.selectedIdx && a.renderedLinesWidth == w && a.renderedLines != nil {
		return // already up to date
	}
	markdown := strings.Join(a.sectionLines, "\n")
	a.renderedLines = a.renderGlamour(markdown, w)
	a.renderedLinesIdx = a.selectedIdx
	a.renderedLinesWidth = w
	a.sel = nil // its positions index the old lines
	a.srcMap = nil
	a.nodeRenderInfo = mapNodesToRenderedLines(a.codeNodes, a.renderedLines)
	// Match positions index into renderedLines, so recompute them for the new render.
	a.findContentSearchMatches()
	a.clampContentOffset()
}

// scrollContentToNode scrolls the content so the selected code node is visible.
func (a *App) scrollContentToNode(nodeIdx int) {
	if nodeIdx < 0 || nodeIdx >= len(a.codeNodes) {
		return
	}
	// Ensure rendered lines and node mapping are built before consulting them.
	a.ensureRenderedLines()
	// Use the rendered-line position if available, fall back to source line.
	target := a.codeNodes[nodeIdx].startLine
	if a.nodeRenderInfo != nil && nodeIdx < len(a.nodeRenderInfo) && a.nodeRenderInfo[nodeIdx].firstLine >= 0 {
		target = a.nodeRenderInfo[nodeIdx].firstLine
	}
	if target < a.contentOffset || target >= a.contentOffset+a.contentHeight() {
		a.contentOffset = target
		a.clampContentOffset()
	}
}

// scrollToFilteredNode scrolls to the node at filteredIdx within the current sub-mode.
func (a *App) scrollToFilteredNode(filteredIdx int) {
	indices := a.filteredNodeIndices()
	if filteredIdx >= 0 && filteredIdx < len(indices) {
		a.scrollContentToNode(indices[filteredIdx])
	}
}

// ─────────────────────────────────────────────
// Search mode
// ─────────────────────────────────────────────

func (a *App) handleSearchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		a.searchMatches = nil
		q := strings.ToLower(a.searchQuery)
		for i, h := range a.doc.Headings {
			if fuzzyMatch(strings.ToLower(h.Text), q) {
				a.searchMatches = append(a.searchMatches, i)
			}
		}
		a.searchIdx = 0
		if len(a.searchMatches) > 0 {
			a.navigateTo(a.searchMatches[0])
		}
		a.mode = ModeNormal

	case "esc":
		a.mode = ModeNormal
		a.searchQuery = ""

	case "backspace", "ctrl+h":
		if r := []rune(a.searchQuery); len(r) > 0 {
			a.searchQuery = string(r[:len(r)-1])
		}

	default:
		if len(msg.Text) > 0 {
			a.searchQuery += string(msg.Text)
		}
	}
	return a, nil
}

// fuzzyMatch checks if all characters in pattern appear in text in order.
func fuzzyMatch(text, pattern string) bool {
	ti := 0
	for _, pc := range pattern {
		found := false
		for ti < len([]rune(text)) {
			if []rune(text)[ti] == pc {
				ti++
				found = true
				break
			}
			ti++
		}
		if !found {
			return false
		}
	}
	return true
}

func (a *App) nextSearchMatch() {
	if len(a.searchMatches) == 0 {
		return
	}
	a.searchIdx = (a.searchIdx + 1) % len(a.searchMatches)
	a.selectedIdx = a.searchMatches[a.searchIdx]
	a.scrollOutlineToSelected()
	a.rebuildSection()
}

func (a *App) prevSearchMatch() {
	if len(a.searchMatches) == 0 {
		return
	}
	a.searchIdx = (a.searchIdx - 1 + len(a.searchMatches)) % len(a.searchMatches)
	a.selectedIdx = a.searchMatches[a.searchIdx]
	a.scrollOutlineToSelected()
	a.rebuildSection()
}

// ─────────────────────────────────────────────
// Content search
// ─────────────────────────────────────────────

func (a *App) handleContentSearchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		a.mode = ModeNormal
		if len(a.contentSearchMatches) > 0 {
			a.scrollToContentMatch(a.contentSearchIdx)
		}
	case "esc":
		a.mode = ModeNormal
		a.contentSearchQuery = ""
		a.contentSearchMatches = nil
	case "backspace", "ctrl+h":
		if r := []rune(a.contentSearchQuery); len(r) > 0 {
			a.contentSearchQuery = string(r[:len(r)-1])
		}
		a.updateContentSearchMatches()
	default:
		if len(msg.Text) > 0 {
			a.contentSearchQuery += string(msg.Text)
		}
		a.updateContentSearchMatches()
	}
	return a, nil
}

func (a *App) updateContentSearchMatches() {
	a.contentSearchIdx = 0
	a.findContentSearchMatches()
	if len(a.contentSearchMatches) > 0 {
		a.scrollToContentMatch(0)
	}
}

func (a *App) findContentSearchMatches() {
	a.contentSearchMatches = nil
	if a.contentSearchQuery == "" {
		return
	}
	// (?i) matches on the original string, so offsets stay valid even when
	// lowercasing would change a character's byte length.
	re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(a.contentSearchQuery))
	for i, l := range a.renderedLines {
		s := ansi.Strip(l)
		for _, m := range re.FindAllStringIndex(s, -1) {
			a.contentSearchMatches = append(a.contentSearchMatches, contentMatch{
				line:     i,
				colStart: byteColToDisplayCol(s, m[0]),
				colEnd:   byteColToDisplayCol(s, m[1]),
			})
		}
	}
	if a.contentSearchIdx >= len(a.contentSearchMatches) {
		a.contentSearchIdx = 0
	}
}

func (a *App) scrollToContentMatch(idx int) {
	if idx < 0 || idx >= len(a.contentSearchMatches) {
		return
	}
	m := a.contentSearchMatches[idx]
	h := a.contentHeight()
	// Ensure the match line is visible
	if m.line < a.contentOffset {
		a.contentOffset = m.line
	} else if m.line >= a.contentOffset+h {
		a.contentOffset = m.line - h/2
	}
	a.clampContentOffset()
}

func (a *App) nextContentSearchMatch() {
	if len(a.contentSearchMatches) == 0 {
		return
	}
	a.contentSearchIdx = (a.contentSearchIdx + 1) % len(a.contentSearchMatches)
	a.scrollToContentMatch(a.contentSearchIdx)
}

func (a *App) prevContentSearchMatch() {
	if len(a.contentSearchMatches) == 0 {
		return
	}
	a.contentSearchIdx = (a.contentSearchIdx - 1 + len(a.contentSearchMatches)) % len(a.contentSearchMatches)
	a.scrollToContentMatch(a.contentSearchIdx)
}

// ─────────────────────────────────────────────
// Theme picker
// ─────────────────────────────────────────────
