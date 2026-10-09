package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/grota/gomd/internal/config"
)

// ─────────────────────────────────────────────
// Theme picker: Dark | Light tabs with live preview
// ─────────────────────────────────────────────

const (
	tabDark = iota
	tabLight
)

// pickerEntry is one theme in a picker tab.
type pickerEntry struct {
	name    string
	builtin bool
	theme   Theme // with the config's colour overrides, for the preview
}

// themePicker is the state of the open picker.
type themePicker struct {
	tab         int
	filter      [2]string
	sel         [2]int // position in the filtered list
	offset      [2]int // first visible row of the filtered list
	downloading bool
	err         string
	switched    bool // the system mode changed while the picker was open
}

// themesDownloadedMsg reports the end of a background theme download.
type themesDownloadedMsg struct{ err error }

// themeLists builds the Dark and Light lists: built-ins first, then the
// downloaded themes, darkest first in Dark and lightest first in Light.
func themeLists(ov config.ThemeOverrideConfig) [2][]pickerEntry {
	type item struct {
		pickerEntry
		lum float64
	}
	var items [2][]item
	add := func(name string, raw Theme, builtin bool) {
		l := luminance(raw.Background)
		tab := tabLight
		if l < 0.5 {
			tab = tabDark
		}
		items[tab] = append(items[tab], item{pickerEntry{name, builtin, applyOverrides(raw, ov)}, l})
	}
	for _, name := range builtinNames {
		add(name, themes[name], true)
	}
	names, _ := remoteThemeNames() // none until downloaded
	for _, name := range names {
		if t, err := loadRemoteTheme(name); err == nil {
			add(name, t, false)
		}
	}
	var lists [2][]pickerEntry
	for tab := range items {
		sort.SliceStable(items[tab], func(i, j int) bool {
			a, b := items[tab][i], items[tab][j]
			if a.builtin != b.builtin {
				return a.builtin
			}
			if a.lum != b.lum {
				return (a.lum < b.lum) == (tab == tabDark)
			}
			return a.name < b.name
		})
		for _, it := range items[tab] {
			lists[tab] = append(lists[tab], it.pickerEntry)
		}
	}
	return lists
}

// openThemePicker opens the picker on the tab of the current system mode,
// highlighting the theme in use.
func (a *App) openThemePicker() tea.Cmd {
	p := &themePicker{tab: tabLight}
	if a.dark {
		p.tab = tabDark
	}
	a.picker = p
	a.mode = ModeThemePicker
	a.themeLists = themeLists(a.cfg.UI.ThemeOverride)
	for tab, slot := range [2]string{a.themeSetting.Dark, a.themeSetting.Light} {
		for i, e := range a.themeLists[tab] {
			if e.name == slot {
				p.sel[tab] = i
			}
		}
	}
	a.clampPicker()
	if RemoteThemesCached() {
		return nil
	}
	p.downloading = true
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return themesDownloadedMsg{EnsureRemoteThemes(ctx)}
	}
}

// filteredEntries returns the entries of tab matching its filter: names
// containing the filter first, then the other fuzzy matches, each group in
// list order.
func (a *App) filteredEntries(tab int) []pickerEntry {
	q := strings.ToLower(a.picker.filter[tab])
	if q == "" {
		return a.themeLists[tab]
	}
	var exact, fuzzy []pickerEntry
	for _, e := range a.themeLists[tab] {
		switch name := strings.ToLower(e.name); {
		case strings.Contains(name, q):
			exact = append(exact, e)
		case fuzzyMatch(name, q):
			fuzzy = append(fuzzy, e)
		}
	}
	return append(exact, fuzzy...)
}

// pickerRows is how many list rows fit in the picker.
func (a *App) pickerRows() int {
	return max(min(a.height-12, 20), 3)
}

// clampPicker keeps the selection inside the filtered list and visible.
func (a *App) clampPicker() {
	p := a.picker
	n := len(a.filteredEntries(p.tab))
	p.sel[p.tab] = max(min(p.sel[p.tab], n-1), 0)
	rows := a.pickerRows()
	if p.sel[p.tab] < p.offset[p.tab] {
		p.offset[p.tab] = p.sel[p.tab]
	}
	if p.sel[p.tab] >= p.offset[p.tab]+rows {
		p.offset[p.tab] = p.sel[p.tab] - rows + 1
	}
}

// previewSelected shows the highlighted theme; with no match the last
// previewed theme stays.
func (a *App) previewSelected() {
	if list := a.filteredEntries(a.picker.tab); len(list) > 0 {
		a.setTheme(list[a.picker.sel[a.picker.tab]].theme)
	}
}

// closeThemePicker leaves the picker and shows the theme for the current mode.
func (a *App) closeThemePicker() {
	a.picker = nil
	a.mode = ModeNormal
	a.applyTheme()
}

// Picker keys are fixed: every printable key goes into the filter.
func (a *App) handleThemeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	p := a.picker
	switch k := msg.String(); k {
	case "esc":
		a.closeThemePicker()
		return a, nil
	case "enter":
		list := a.filteredEntries(p.tab)
		if len(list) == 0 {
			return a, nil
		}
		a.saveThemeSlot(p.tab == tabDark, list[p.sel[p.tab]].name)
		return a, nil
	case "tab", "shift+tab":
		p.tab = 1 - p.tab
	case "up", "ctrl+p":
		p.sel[p.tab]--
	case "down", "ctrl+n":
		p.sel[p.tab]++
	case "pgup":
		p.sel[p.tab] -= a.pickerRows()
	case "pgdown":
		p.sel[p.tab] += a.pickerRows()
	case "backspace":
		if r := []rune(p.filter[p.tab]); len(r) > 0 {
			p.filter[p.tab] = string(r[:len(r)-1])
			p.sel[p.tab], p.offset[p.tab] = 0, 0
		}
	case "ctrl+u":
		p.filter[p.tab], p.sel[p.tab], p.offset[p.tab] = "", 0, 0
	default:
		if msg.Text == "" {
			return a, nil
		}
		p.filter[p.tab] += msg.Text
		p.sel[p.tab], p.offset[p.tab] = 0, 0
	}
	a.clampPicker()
	a.previewSelected()
	return a, nil
}

// saveThemeSlot stores name in the dark or light slot, saves it and closes
// the picker.
func (a *App) saveThemeSlot(dark bool, name string) {
	if dark {
		a.themeSetting.Dark = name
	} else {
		a.themeSetting.Light = name
	}
	a.cfg.UI.Theme = a.themeSetting.String()
	a.closeThemePicker()
	switch err := config.SaveTheme(a.cfg.UI.Theme); {
	case err != nil:
		a.statusMsg = "Theme not saved: " + err.Error()
	case dark != a.dark:
		mode := modeName(dark)
		a.statusMsg = fmt.Sprintf("Saved as %s theme; shown when the system is %s", mode, mode)
	}
}

func modeName(dark bool) string {
	if dark {
		return "dark"
	}
	return "light"
}

// handleThemesDownloaded refreshes the picker lists after a download.
func (a *App) handleThemesDownloaded(msg themesDownloadedMsg) {
	if a.picker == nil {
		return
	}
	a.picker.downloading = false
	if msg.err != nil {
		a.picker.err = msg.err.Error()
		return
	}
	a.themeLists = themeLists(a.cfg.UI.ThemeOverride)
	a.clampPicker()
}

// overlayThemePicker draws the picker over the view.
func (a *App) overlayThemePicker(background string) string {
	p := a.picker
	modalW := min(60, a.width-4)
	innerW := max(modalW-4, 10)
	active := lipgloss.NewStyle().Bold(true).Foreground(a.theme.Highlight)
	dim := lipgloss.NewStyle().Foreground(a.theme.HeadingN)

	// ● marks the tab matching the current system mode.
	tabLabel := func(tab int, label string) string {
		if (tab == tabDark) == a.dark {
			label += " ●"
		}
		if tab == p.tab {
			return active.Render("[" + label + "]")
		}
		return dim.Render(" " + label + " ")
	}
	lines := []string{tabLabel(tabDark, "Dark") + "  " + tabLabel(tabLight, "Light")}
	if p.switched {
		lines = append(lines, lipgloss.NewStyle().Foreground(a.theme.Highlight).Render(
			ansi.Truncate("System is now "+modeName(a.dark)+": applied when you close this", innerW, "…")))
	}
	lines = append(lines, "Search: "+p.filter[p.tab]+"█", "")

	list := a.filteredEntries(p.tab)
	inUse := [2]string{a.themeSetting.Dark, a.themeSetting.Light}[p.tab]
	rows := a.pickerRows()
	for i := p.offset[p.tab]; i < min(p.offset[p.tab]+rows, len(list)); i++ {
		e := list[i]
		var tags []string
		if e.builtin {
			tags = append(tags, "built-in")
		}
		if e.name == inUse {
			tags = append(tags, "in use")
		}
		tag := ""
		if len(tags) > 0 {
			tag = " " + strings.Join(tags, ", ")
		}
		name := ansi.Truncate(e.name, innerW-2-len(tag), "…")
		pad := strings.Repeat(" ", max(innerW-2-ansi.StringWidth(name)-len(tag), 0))
		marker := "  "
		if i == p.sel[p.tab] {
			marker, name = active.Render("→ "), lipgloss.NewStyle().Bold(true).Render(name)
		}
		lines = append(lines, marker+name+pad+dim.Render(tag))
	}
	switch {
	case p.downloading:
		lines = append(lines, dim.Render("Downloading Ghostty themes…"))
	case p.err != "":
		lines = append(lines, lipgloss.NewStyle().Foreground(a.theme.Search).Render(ansi.Truncate("Download failed: "+p.err, innerW, "…")))
	case len(list) == 0:
		lines = append(lines, dim.Render("No match"))
	}
	lines = append(lines, "",
		dim.Render(fmt.Sprintf("%d of %d", len(list), len(a.themeLists[p.tab]))),
		dim.Render("Tab: dark/light  ↑↓: move  Enter: save  Esc: cancel"))
	return a.overlayModal(background, strings.Join(lines, "\n"), modalW)
}
