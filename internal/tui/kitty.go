package tui

import (
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"math"
	"math/rand/v2"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/x/ansi/kitty"

	"github.com/grota/gomd/internal/parser"
)

// IsKittySupported reports whether the terminal can show Kitty Unicode
// placeholder images (Kitty, Ghostty). Inside tmux it asks tmux for the
// attached client's terminal and whether passthrough (needed to upload the
// image) is allowed.
func IsKittySupported() bool {
	if inTmux() {
		out, err := exec.Command("tmux", "display", "-p", "#{client_termname} #{allow-passthrough}").Output()
		f := strings.Fields(string(out))
		return err == nil && len(f) == 2 && isKittyTerm(f[0]) && (f[1] == "on" || f[1] == "all")
	}
	return os.Getenv("KITTY_WINDOW_ID") != "" || isKittyTerm(os.Getenv("TERM_PROGRAM")) || isKittyTerm(os.Getenv("TERM"))
}

func isKittyTerm(s string) bool {
	s = strings.ToLower(s)
	return strings.Contains(s, "kitty") || strings.Contains(s, "ghostty")
}

// ResolveImagePath resolves an image src relative to the markdown file's directory.
// Returns empty string if the image cannot be resolved (e.g. URLs).
func ResolveImagePath(src, mdFilePath string) string {
	// Skip URLs
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		return ""
	}
	// Skip data URIs
	if strings.HasPrefix(src, "data:") {
		return ""
	}
	// URL-decode the path
	if decoded, err := url.PathUnescape(src); err == nil {
		src = decoded
	}
	// Absolute path
	if filepath.IsAbs(src) {
		if _, err := os.Stat(src); err == nil {
			return src
		}
		return ""
	}
	// Relative path: resolve against markdown file directory
	if mdFilePath == "" {
		return ""
	}
	dir := filepath.Dir(mdFilePath)
	resolved := filepath.Join(dir, src)
	if _, err := os.Stat(resolved); err == nil {
		return resolved
	}
	return ""
}

// inTmux returns true if the current process is running inside tmux.
func inTmux() bool {
	return os.Getenv("TMUX") != ""
}

// wrapTmuxPassthrough wraps a sequence for tmux DCS passthrough.
// Inside tmux, escape sequences destined for the outer terminal must be
// wrapped as: \x1bPtmux;<seq with ESC doubled>\x1b\\
func wrapTmuxPassthrough(seq string) string {
	// Double every \x1b inside the payload
	doubled := strings.ReplaceAll(seq, "\x1b", "\x1b\x1b")
	return "\x1bPtmux;" + doubled + "\x1b\\"
}

// kittyImage is an image uploaded to the terminal once and then drawn as
// Unicode placeholder cells, which are ordinary text: bubbletea diffs them and
// tmux scrolls them like any other character.
type kittyImage struct {
	id         uint32 // 24-bit, encoded in the placeholders' foreground colour
	cols, rows int
	upload     string // escape sequences that transmit the image; send once
}

// loadKittyImage decodes path and prepares its upload, sized to at most
// maxCols cells wide.
func loadKittyImage(path string, maxCols int) (kittyImage, error) {
	f, err := os.Open(path)
	if err != nil {
		return kittyImage{}, fmt.Errorf("open image: %w", err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return kittyImage{}, fmt.Errorf("decode image: %w", err)
	}
	imgW, imgH := img.Bounds().Dx(), img.Bounds().Dy()
	if imgW == 0 || imgH == 0 {
		return kittyImage{}, fmt.Errorf("image has zero dimensions")
	}

	// shortcut: assumes ~8x16 px cells; query the cell size (CSI 16 t) if
	// images look stretched on unusual fonts.
	const cellW, cellH = 8, 16
	cols := max(min(int(math.Ceil(float64(imgW)/cellW)), maxCols), 1)
	rows := max(int(math.Ceil(float64(imgH)/cellH*float64(cols)*cellW/float64(imgW))), 1)

	var buf strings.Builder
	if err := png.Encode(&buf, img); err != nil {
		return kittyImage{}, fmt.Errorf("encode png: %w", err)
	}
	b64 := base64.StdEncoding.EncodeToString([]byte(buf.String()))

	// Random ids keep two gomd instances (e.g. tmux panes) from overwriting
	// each other's images in the shared outer terminal.
	id := rand.Uint32N(1<<24-1) + 1
	const chunkSize = 4096
	var sb strings.Builder
	for i := 0; i < len(b64); i += chunkSize {
		end := min(i+chunkSize, len(b64))
		more := 0
		if end < len(b64) {
			more = 1
		}
		// a=T,U=1: transmit and create a virtual placement for placeholders.
		// q=2: no replies, which would otherwise arrive as keyboard input.
		seq := fmt.Sprintf("\x1b_Gm=%d;%s\x1b\\", more, b64[i:end])
		if i == 0 {
			seq = fmt.Sprintf("\x1b_Ga=T,U=1,q=2,f=100,t=d,i=%d,c=%d,r=%d,m=%d;%s\x1b\\", id, cols, rows, more, b64[i:end])
		}
		if inTmux() {
			seq = wrapTmuxPassthrough(seq)
		}
		sb.WriteString(seq)
	}
	return kittyImage{id: id, cols: cols, rows: rows, upload: sb.String()}, nil
}

// placeholderRows returns the text rows that display the image.
func (k kittyImage) placeholderRows() []string {
	fg := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", k.id>>16&0xff, k.id>>8&0xff, k.id&0xff)
	rows := make([]string, k.rows)
	for r := range rows {
		var sb strings.Builder
		sb.WriteString(fg)
		for c := 0; c < k.cols; c++ {
			sb.WriteRune(kitty.Placeholder)
			sb.WriteRune(kitty.Diacritic(r))
			sb.WriteRune(kitty.Diacritic(c))
		}
		sb.WriteString("\x1b[39m")
		rows[r] = sb.String()
	}
	return rows
}

// indentedRows returns placeholderRows aligned with glamour's 2-column margin.
func (k kittyImage) indentedRows() []string {
	rows := k.placeholderRows()
	for i := range rows {
		rows[i] = "  " + rows[i]
	}
	return rows
}

// deleteSeq returns the sequence that frees the image in the terminal.
func (k kittyImage) deleteSeq() string {
	seq := fmt.Sprintf("\x1b_Ga=d,d=I,q=2,i=%d\x1b\\", k.id)
	if inTmux() {
		seq = wrapTmuxPassthrough(seq)
	}
	return seq
}

// ansiStripRe matches ANSI escape sequences for stripping when identifying image lines.
var ansiStripRe = regexp.MustCompile(`\x1b(?:\[[0-9;]*[a-zA-Z]|\].*?\x07|_.*?\x1b\\)`)

// imageCache maps a resolved path and width to an image already uploaded.
type imageCache map[string]kittyImage

// placeImages inserts placeholder rows after glamour's image captions
// ("alt 📷 /path") in rendered lines. With keepCaption false, a line holding
// only a caption is replaced by the image; a line with other text always
// keeps it. Images missing from cache are loaded and added, and their upload
// sequences are returned for the caller to send before the rows are shown.
func placeImages(lines []string, mdContent, mdPath string, maxCols int, keepCaption bool, cache imageCache) ([]string, []string) {
	images := parser.ExtractImages(mdContent)
	if len(images) == 0 || maxCols < 1 {
		return lines, nil
	}
	absMdPath := mdPath
	if mdPath != "" && !filepath.IsAbs(mdPath) {
		if abs, err := filepath.Abs(mdPath); err == nil {
			absMdPath = abs
		}
	}

	// Key by the path glamour displays, so "./a.png" and "a.png" share an entry.
	type entry struct {
		img      kittyImage
		captions []string
	}
	entries := map[string]*entry{}
	var order []string
	var uploads []string
	for _, im := range images {
		cp := imageCaptionPath(im.Src)
		if e, ok := entries[cp]; ok {
			e.captions = append(e.captions, imageCaption(im.Alt, im.Src))
			continue
		}
		resolved := ResolveImagePath(im.Src, absMdPath)
		if resolved == "" {
			continue
		}
		key := fmt.Sprintf("%s@%d", resolved, maxCols)
		k, ok := cache[key]
		if !ok {
			var err error
			if k, err = loadKittyImage(resolved, maxCols); err != nil {
				continue
			}
			cache[key] = k
			uploads = append(uploads, k.upload)
		}
		entries[cp] = &entry{img: k, captions: []string{imageCaption(im.Alt, im.Src)}}
		order = append(order, cp)
	}
	if len(entries) == 0 {
		return lines, uploads
	}

	var out []string
	for _, line := range lines {
		stripped := strings.TrimSpace(ansiStripRe.ReplaceAllString(line, ""))
		// A line holding only one image's caption.
		var standalone *entry
		for cp, e := range entries {
			if stripped == cp {
				standalone = e
			}
			for _, c := range e.captions {
				if stripped == c {
					standalone = e
				}
			}
		}
		if standalone != nil {
			if keepCaption {
				out = append(out, line)
			}
			out = append(out, standalone.img.indentedRows()...)
			continue
		}
		out = append(out, line)
		if !strings.Contains(stripped, "📷") {
			continue
		}
		// Inline images: keep the text, add each image below in reading order.
		type hit struct {
			at  int
			img kittyImage
		}
		var hits []hit
		for _, cp := range order {
			if at := strings.Index(stripped, "📷 "+cp); at >= 0 {
				hits = append(hits, hit{at, entries[cp].img})
			}
		}
		sort.Slice(hits, func(i, j int) bool { return hits[i].at < hits[j].at })
		for _, h := range hits {
			out = append(out, h.img.indentedRows()...)
		}
	}
	return out, uploads
}

// RenderImages places images into render-mode output and returns the text to
// print: the uploads first, then the rendered lines with placeholder rows.
func RenderImages(rendered, mdContent, mdPath string, maxCols int) string {
	lines, uploads := placeImages(strings.Split(rendered, "\n"), mdContent, mdPath, maxCols, false, imageCache{})
	return strings.Join(uploads, "") + strings.Join(lines, "\n")
}

// imageCaptionPath is the path glamour shows for an image src: URLs as
// written, local paths cleaned and rooted ("./a/../b.png" -> "/b.png"), URL
// escapes kept.
func imageCaptionPath(src string) string {
	if strings.Contains(src, "://") {
		return src
	}
	return path.Clean("/" + src)
}

// imageCaption is the full text glamour renders for an image: "alt 📷 /path",
// or the path alone when alt is empty.
func imageCaption(alt, src string) string {
	if alt == "" {
		return imageCaptionPath(src)
	}
	return alt + " 📷 " + imageCaptionPath(src)
}
