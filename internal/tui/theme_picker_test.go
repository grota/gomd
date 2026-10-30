package tui

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	gansi "charm.land/glamour/v2/ansi"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"github.com/grota/gomd/internal/config"
	"github.com/grota/gomd/internal/parser"
)

func TestParseThemeSetting(t *testing.T) {
	good := map[string]ThemeSetting{
		"Dracula":                               {"Dracula", "Dracula"},
		"light:Gruvbox Light Hard,dark:Dracula": {"Gruvbox Light Hard", "Dracula"},
		"dark: Nord , light: Ayu Light":         {"Ayu Light", "Nord"},
		"":                                      {"OceanLight", "OceanDark"},
	}
	for in, want := range good {
		got, err := ParseThemeSetting(in)
		if err != nil || got != want {
			t.Errorf("%q: got %+v, %v; want %+v", in, got, err, want)
		}
	}
	for _, in := range []string{"light:A", "light:,dark:B", "dusk:A,dark:B", "light:A,dark"} {
		if _, err := ParseThemeSetting(in); err == nil {
			t.Errorf("%q: want an error", in)
		}
	}
	if s := (ThemeSetting{"A", "B"}).String(); s != "light:A,dark:B" {
		t.Errorf("String() = %q", s)
	}
	if s := (ThemeSetting{"A", "A"}).String(); s != "A" {
		t.Errorf("String() of a single theme = %q", s)
	}
}

// fakeThemes serves a theme archive and isolates the config and cache dirs.
func fakeThemes(t *testing.T, files map[string]string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, "cache"))

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		tw.Write([]byte(body))
	}
	tw.Close()
	gz.Close()
	archive := buf.Bytes()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(archive) }))
	t.Cleanup(srv.Close)

	sum := sha256.Sum256(archive)
	oldURL, oldSum := remoteThemesURL, remoteThemesWantSHA256
	remoteThemesURL, remoteThemesWantSHA256 = srv.URL, hex.EncodeToString(sum[:])
	t.Cleanup(func() { remoteThemesURL, remoteThemesWantSHA256 = oldURL, oldSum })
}

func ghosttyFile(bg string) string {
	return "palette = 0=#111111\npalette = 8=#888888\nbackground = " + bg + "\nforeground = #777777\n"
}

func TestEnsureRemoteThemes(t *testing.T) {
	fakeThemes(t, map[string]string{
		"ghostty/Deep Night": ghosttyFile("#000000"),
		"ghostty/Paper":      ghosttyFile("#ffffff"),
		"ghostty/../escape":  "x",
		"../outside":         "x",
		"other/NotATheme":    "x",
		"ghostty/sub/Nested": "x",
		"ghostty/.hidden":    "x",
	})
	if _, err := loadRemoteTheme("Paper"); !errors.Is(err, ErrThemesNotDownloaded) {
		t.Fatalf("before download: got %v", err)
	}
	if err := EnsureRemoteThemes(context.Background()); err != nil {
		t.Fatal(err)
	}
	names, err := remoteThemeNames()
	if err != nil || strings.Join(names, ",") != "Deep Night,Paper" {
		t.Fatalf("names = %q, %v", names, err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(remoteThemesDir()), "escape")); err == nil {
		t.Error("an archive entry escaped the cache dir")
	}
	if _, err := loadRemoteTheme("Nope"); !errors.Is(err, ErrUnknownTheme) {
		t.Errorf("unknown name: got %v", err)
	}
	if th, err := loadRemoteTheme("Paper"); err != nil || th.Background != "#ffffff" {
		t.Errorf("Paper: %+v, %v", th, err)
	}
}

func TestEnsureRemoteThemesRejectsBadChecksum(t *testing.T) {
	fakeThemes(t, map[string]string{"ghostty/A": ghosttyFile("#000000")})
	remoteThemesWantSHA256 = strings.Repeat("0", 64)
	if err := EnsureRemoteThemes(context.Background()); err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("want checksum error, got %v", err)
	}
	if RemoteThemesCached() {
		t.Error("a rejected archive was cached")
	}
}

func TestThemeForFallsBackWithoutDownload(t *testing.T) {
	fakeThemes(t, nil)
	cfg := config.Default()
	cfg.UI.Theme = "light:Paper,dark:Deep Night"
	th, err := ThemeFor(cfg, false)
	if !errors.Is(err, ErrThemesNotDownloaded) || th.Name != "OceanLight" {
		t.Errorf("got %s, %v", th.Name, err)
	}
}

func pickerApp(t *testing.T) *App {
	t.Helper()
	fakeThemes(t, map[string]string{
		"ghostty/Deep Night": ghosttyFile("#000000"),
		"ghostty/Dusk":       ghosttyFile("#303030"),
		"ghostty/Nu Sky":     ghosttyFile("#050505"),
		"ghostty/Paper":      ghosttyFile("#ffffff"),
		"ghostty/Cream":      ghosttyFile("#e0e0d0"),
	})
	if err := EnsureRemoteThemes(context.Background()); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.UI.Theme = "light:Cream,dark:Dusk"
	if err := config.SaveTheme(cfg.UI.Theme); err != nil {
		t.Fatal(err)
	}
	a := NewApp(parser.ParseMarkdown("# T\n\ntext\n"), "t.md", "", cfg)
	a.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	return a
}

func TestPickerDoesNotSaveTheThemeFlag(t *testing.T) {
	a := pickerApp(t) // the file says light:Cream,dark:Dusk
	a.cfg.UI.Theme = "Nu Sky"
	a.themeSetting, _ = ParseThemeSetting(a.cfg.UI.Theme) // as with --theme "Nu Sky"
	press(a, "T", "tab", "p", "a", "enter")
	if a.cfg.UI.Theme != "light:Paper,dark:Nu Sky" {
		t.Errorf("this run should keep the flag's dark theme: %q", a.cfg.UI.Theme)
	}
	saved, err := os.ReadFile(filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "gomd", "config.toml"))
	if err != nil || !strings.Contains(string(saved), `theme = "light:Paper,dark:Dusk"`) {
		t.Errorf("saved %q, %v; want the file's dark theme kept", saved, err)
	}
}

func names(entries []pickerEntry) string {
	var s []string
	for _, e := range entries {
		s = append(s, e.name)
	}
	return strings.Join(s, ",")
}

func press(a *App, keys ...string) {
	for _, k := range keys {
		var msg tea.KeyPressMsg
		switch k {
		case "esc":
			msg = tea.KeyPressMsg{Code: tea.KeyEscape}
		case "enter":
			msg = tea.KeyPressMsg{Code: tea.KeyEnter}
		case "tab":
			msg = tea.KeyPressMsg{Code: tea.KeyTab}
		case "down":
			msg = tea.KeyPressMsg{Code: tea.KeyDown}
		case "backspace":
			msg = tea.KeyPressMsg{Code: tea.KeyBackspace}
		default:
			msg = tea.KeyPressMsg{Code: []rune(k)[0], Text: k}
		}
		a.Update(msg)
	}
}

func TestPickerListsAndPreview(t *testing.T) {
	a := pickerApp(t)
	if a.theme.Name != "Dusk" {
		t.Fatalf("dark mode should show Dusk, got %s", a.theme.Name)
	}
	press(a, "T")
	if a.picker == nil || a.picker.tab != tabDark {
		t.Fatal("picker should open on the Dark tab in dark mode")
	}
	if got := names(a.themeLists[tabDark]); got != "OceanDark,Deep Night,Nu Sky,Dusk" {
		t.Errorf("dark order: %s", got)
	}
	if got := names(a.themeLists[tabLight]); got != "OceanLight,Paper,Cream" {
		t.Errorf("light order: %s", got)
	}
	if e := a.filteredEntries(tabDark)[a.picker.sel[tabDark]]; e.name != "Dusk" {
		t.Errorf("highlight should start on the theme in use, got %s", e.name)
	}

	press(a, "d", "e")
	if got := names(a.filteredEntries(tabDark)); got != "Deep Night" || a.theme.Name != "Deep Night" {
		t.Errorf("filter %q: list %s, preview %s", a.picker.filter[tabDark], got, a.theme.Name)
	}
	press(a, "backspace", "backspace", "u", "s")
	if got := names(a.filteredEntries(tabDark)); got != "Dusk,Nu Sky" {
		t.Errorf("filter %q: names containing it should come first, got %s", a.picker.filter[tabDark], got)
	}
	press(a, "esc")
	if a.picker != nil || a.theme.Name != "Dusk" {
		t.Errorf("Esc should close and restore Dusk, got %s", a.theme.Name)
	}
}

func TestPickerSavesOnlyTheActiveSlot(t *testing.T) {
	a := pickerApp(t)
	press(a, "T", "tab", "p", "a", "enter")
	if a.cfg.UI.Theme != "light:Paper,dark:Dusk" {
		t.Errorf("setting = %q", a.cfg.UI.Theme)
	}
	if a.theme.Name != "Dusk" || !strings.Contains(a.statusMsg, "light theme") {
		t.Errorf("dark mode should keep showing Dusk: %s, status %q", a.theme.Name, a.statusMsg)
	}
	saved, err := os.ReadFile(filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "gomd", "config.toml"))
	if err != nil || !strings.Contains(string(saved), `theme = "light:Paper,dark:Dusk"`) {
		t.Errorf("config not saved: %q, %v", saved, err)
	}
}

func TestSystemModeSwitchWaitsForPicker(t *testing.T) {
	a := pickerApp(t)
	a.Update(uv.LightColorSchemeEvent{})
	if a.theme.Name != "Cream" {
		t.Fatalf("light report should show Cream, got %s", a.theme.Name)
	}
	press(a, "T", "down") // Light tab opens on Cream; preview the next one
	previewed := a.theme.Name
	a.Update(uv.DarkColorSchemeEvent{})
	if a.theme.Name != previewed {
		t.Errorf("mode switch replaced the preview: %s", a.theme.Name)
	}
	press(a, "esc")
	if a.theme.Name != "Dusk" {
		t.Errorf("after closing, dark mode should show Dusk, got %s", a.theme.Name)
	}
}

// Every popup cell must carry the theme background: a reset not followed by
// a colour (or the end of the line) would show the terminal's own background.
func TestPopupPaintsItsOwnBackground(t *testing.T) {
	a := pickerApp(t)
	press(a, "T")
	popup := a.overlayThemePicker(strings.Repeat("\n", a.height-1))
	for i, line := range strings.Split(popup, "\n") {
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "\x1b[") {
			t.Errorf("line %d starts without colours: %q", i, line)
		}
		for _, reset := range []string{"\x1b[m", "\x1b[0m"} {
			for rest := line; ; {
				j := strings.Index(rest, reset)
				if j < 0 {
					break
				}
				rest = rest[j+len(reset):]
				if rest != "" && !strings.HasPrefix(rest, "\x1b[") {
					t.Errorf("line %d: text after a reset without colours: %q", i, rest)
					break
				}
			}
		}
	}
}

func TestPickerMarksSystemModeAndNotesSwitch(t *testing.T) {
	a := pickerApp(t)
	press(a, "T")
	view := ansi.Strip(a.overlayThemePicker(strings.Repeat("\n", a.height-1)))
	if !strings.Contains(view, "Dark ●") || strings.Contains(view, "Light ●") || strings.Contains(view, "will be used when you close") {
		t.Fatalf("dark mode: want only Dark marked and no note:\n%s", view)
	}
	a.Update(uv.LightColorSchemeEvent{})
	view = ansi.Strip(a.overlayThemePicker(strings.Repeat("\n", a.height-1)))
	if !strings.Contains(view, "Light ●") || strings.Contains(view, "Dark ●") ||
		!strings.Contains(view, "The light theme will be used when you close this") {
		t.Errorf("after switching to light:\n%s", view)
	}
}

func TestTextOnAvoidsInvisibleText(t *testing.T) {
	primary := Theme{Background: "#ffffff", Foreground: "#000000"}
	if got := primary.textOn("#000000"); got != "#ffffff" {
		t.Errorf("bar #000000 on Primary: got %s, want #ffffff", got)
	}
	fahrenheit := Theme{Background: "#000000", Foreground: "#ffffce"}
	if got := fahrenheit.textOn("#404040"); got != "#ffffce" {
		t.Errorf("bar #404040 on Fahrenheit: got %s, want #ffffce", got)
	}
	// Neither theme colour reaches 4.5:1 on mid grey: fall back to black.
	grey := Theme{Background: "#555555", Foreground: "#aaaaaa"}
	if got := grey.textOn("#999999"); got != "#000000" {
		t.Errorf("mid grey fill: got %s, want #000000", got)
	}
}

func TestMix(t *testing.T) {
	if got := mix("#000000", "#ffffff", 0.12); got != "#1f1f1f" {
		t.Errorf("mix black/white 12%%: got %s, want #1f1f1f", got)
	}
	if got := mix("203", "#ffffff", 0.5); got != "203" {
		t.Errorf("mix with a non-hex colour: got %s, want it unchanged", got)
	}
}

func TestCodeColoursReadableOnEveryTheme(t *testing.T) {
	// Pages that broke the old fixed or blended code background: dim
	// foregrounds, near-black, coloured and mid-grey backgrounds.
	cases := []Theme{
		{Name: "Fahrenheit", Background: "#000000", Foreground: "#ffffce"},
		{Name: "Primary", Background: "#ffffff", Foreground: "#000000"},
		{Name: "Batman", Background: "#1b1d1e", Foreground: "#6f6f6f"},
		{Name: "C64", Background: "#40318d", Foreground: "#7869c4"},
		{Name: "3024 Night", Background: "#090300", Foreground: "#a5a2a2"},
		{Name: "mid grey", Background: "#808080", Foreground: "#ffffff"},
		themes["OceanDark"], themes["OceanLight"],
	}
	for _, th := range cases {
		sc := StyleConfigFromTheme(th)
		bg := Color(*sc.Code.BackgroundColor)
		if r := contrast(bg, th.Background); r < 1.25 {
			t.Errorf("%s: code background %s is %.2f:1 from the page", th.Name, bg, r)
		}
		if r := contrast(Color(*sc.Code.Color), bg); r < 4.5 {
			t.Errorf("%s: inline code %s is %.2f:1 on %s", th.Name, *sc.Code.Color, r, bg)
		}
		v := reflect.ValueOf(*sc.CodeBlock.Chroma)
		for i := range v.NumField() {
			p, ok := v.Field(i).Interface().(gansi.StylePrimitive)
			if !ok || p.Color == nil || p.BackgroundColor != nil {
				continue // Error and Background bring their own fill
			}
			name, want := v.Type().Field(i).Name, 4.5
			if name == "Comment" || name == "GenericSubheading" {
				want = 3
			}
			if r := contrast(Color(*p.Color), bg); r < want {
				t.Errorf("%s: %s %s is %.2f:1 on %s, want %.1f", th.Name, name, *p.Color, r, bg, want)
			}
		}
	}
}
