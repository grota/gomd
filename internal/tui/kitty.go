package tui

import (
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/grota/gomd/internal/parser"
)

// IsKittySupported checks if the terminal supports the Kitty graphics protocol.
func IsKittySupported() bool {
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return true
	}
	tp := os.Getenv("TERM_PROGRAM")
	switch strings.ToLower(tp) {
	case "kitty", "ghostty", "wezterm":
		return true
	}
	term := os.Getenv("TERM")
	if strings.Contains(strings.ToLower(term), "kitty") {
		return true
	}
	return false
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

// RenderKittyImage loads an image file and returns Kitty graphics protocol escape
// sequences to display it. maxCols limits the display width in terminal cells.
func RenderKittyImage(path string, maxCols int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open image: %w", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return "", fmt.Errorf("decode image: %w", err)
	}

	bounds := img.Bounds()
	imgW := bounds.Dx()
	imgH := bounds.Dy()

	if imgW == 0 || imgH == 0 {
		return "", fmt.Errorf("image has zero dimensions")
	}

	// Calculate display columns. Assume ~8px per cell width, ~16px per cell height.
	// Limit to maxCols width.
	cellW := 8
	cellH := 16
	displayCols := int(math.Ceil(float64(imgW) / float64(cellW)))
	if displayCols > maxCols {
		displayCols = maxCols
	}
	// Calculate rows maintaining aspect ratio
	displayRows := int(math.Ceil(float64(imgH) / float64(cellH) * float64(displayCols) * float64(cellW) / float64(imgW)))
	if displayRows < 1 {
		displayRows = 1
	}

	// Re-encode to PNG for transmission
	var pngBuf strings.Builder
	if err := png.Encode(&pngBuf, img); err != nil {
		return "", fmt.Errorf("encode png: %w", err)
	}

	// Base64 encode
	b64Data := base64.StdEncoding.EncodeToString([]byte(pngBuf.String()))

	// Build Kitty escape sequences with chunking (max 4096 bytes per chunk)
	const chunkSize = 4096
	tmux := inTmux()
	var sb strings.Builder

	for i := 0; i < len(b64Data); i += chunkSize {
		end := i + chunkSize
		if end > len(b64Data) {
			end = len(b64Data)
		}
		chunk := b64Data[i:end]

		var seq string
		if i == 0 {
			// First chunk: include all parameters
			more := 1
			if end >= len(b64Data) {
				more = 0
			}
			seq = fmt.Sprintf("\x1b_Gf=100,a=T,t=d,c=%d,r=%d,m=%d;%s\x1b\\", displayCols, displayRows, more, chunk)
		} else {
			// Subsequent chunks
			more := 1
			if end >= len(b64Data) {
				more = 0
			}
			seq = fmt.Sprintf("\x1b_Gm=%d;%s\x1b\\", more, chunk)
		}

		if tmux {
			seq = wrapTmuxPassthrough(seq)
		}
		sb.WriteString(seq)
	}

	return sb.String(), nil
}

// ansiStripRe matches ANSI escape sequences for stripping when identifying image lines.
var ansiStripRe = regexp.MustCompile(`\x1b(?:\[[0-9;]*[a-zA-Z]|\].*?\x07|_.*?\x1b\\)`)

// ReplaceImagePlaceholders swaps glamour image placeholders ("alt 🖼️ /path")
// in rendered output for Kitty graphics sequences. A line holding only the
// placeholder is replaced; a line with other text keeps it and gets the image
// on the next line. Used by render mode (-r); the TUI does not draw images.
func ReplaceImagePlaceholders(rendered, mdContent, mdPath string, maxCols int) string {
	images := parser.ExtractImages(mdContent)
	if len(images) == 0 {
		return rendered
	}

	absMdPath := mdPath
	if mdPath != "" && !filepath.IsAbs(mdPath) {
		if abs, err := filepath.Abs(mdPath); err == nil {
			absMdPath = abs
		}
	}

	// Key by the path glamour displays, so "./a.png" and "a.png" share an entry.
	type entry struct {
		seq  string
		alts []string
	}
	entries := map[string]*entry{}
	var order []string
	for _, img := range images {
		gp := displayedImagePath(img.Src)
		if e, ok := entries[gp]; ok {
			e.alts = append(e.alts, img.Alt)
			continue
		}
		resolved := ResolveImagePath(img.Src, absMdPath)
		if resolved == "" {
			continue
		}
		seq, err := RenderKittyImage(resolved, maxCols)
		if err != nil {
			continue
		}
		entries[gp] = &entry{seq: seq, alts: []string{img.Alt}}
		order = append(order, gp)
	}
	if len(entries) == 0 {
		return rendered
	}

	var result []string
	for _, line := range strings.Split(rendered, "\n") {
		stripped := strings.TrimSpace(ansiStripRe.ReplaceAllString(line, ""))
		// A line holding only one image's caption is replaced by the image.
		standalone := ""
		for gp, e := range entries {
			if stripped == gp || stripped == "/"+gp {
				standalone = e.seq
			}
			for _, alt := range e.alts {
				if stripped == strings.TrimSpace(alt+" 🖼️ /"+gp) {
					standalone = e.seq
				}
			}
		}
		if standalone != "" {
			result = append(result, standalone)
			continue
		}
		result = append(result, line)
		if !strings.Contains(stripped, "🖼️") {
			continue
		}
		// Inline images: keep the text, put each image on the following lines
		// in the order they appear.
		type hit struct {
			at  int
			seq string
		}
		var hits []hit
		for _, gp := range order {
			if at := strings.Index(stripped, "🖼️ /"+gp); at >= 0 {
				hits = append(hits, hit{at, entries[gp].seq})
			}
		}
		sort.Slice(hits, func(i, j int) bool { return hits[i].at < hits[j].at })
		for _, h := range hits {
			result = append(result, h.seq)
		}
	}
	return strings.Join(result, "\n")
}

// displayedImagePath is the path glamour shows for an image src: leading "./"
// and "../" segments stripped, URL escapes kept as written.
func displayedImagePath(src string) string {
	gp := strings.TrimPrefix(src, "./")
	for strings.HasPrefix(gp, "../") {
		gp = strings.TrimPrefix(gp, "../")
	}
	return gp
}
