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

// replaceImagePlaceholders finds glamour image placeholders in rendered output
// and replaces them with Kitty graphics escape sequences.
// Used in render mode (-r) only; TUI mode does not support inline images.
func (a *App) replaceImagePlaceholders(rendered, mdContent string, maxCols int) string {
	images := parser.ExtractImages(mdContent)
	if len(images) == 0 {
		return rendered
	}

	absMdPath := a.filepath
	if !filepath.IsAbs(absMdPath) {
		if abs, err := filepath.Abs(absMdPath); err == nil {
			absMdPath = abs
		}
	}

	// Build map from image src → kitty sequence
	kittyMap := map[string]string{}
	for _, img := range images {
		resolved := ResolveImagePath(img.Src, absMdPath)
		if resolved == "" {
			continue
		}
		if _, done := kittyMap[img.Src]; done {
			continue
		}
		kittySeq, err := RenderKittyImage(resolved, maxCols-4)
		if err != nil {
			continue
		}
		kittyMap[img.Src] = kittySeq
	}
	if len(kittyMap) == 0 {
		return rendered
	}

	lines := strings.Split(rendered, "\n")
	var result []string
	for _, line := range lines {
		stripped := ansiStripRe.ReplaceAllString(line, "")
		stripped = strings.TrimSpace(stripped)

		replaced := false
		for src, kittySeq := range kittyMap {
			if !strings.Contains(line, src) {
				continue
			}
			// Compute what glamour shows as visible text for this src.
			glamourPath := src
			glamourPath = strings.TrimPrefix(glamourPath, "./")
			for strings.HasPrefix(glamourPath, "../") {
				glamourPath = strings.TrimPrefix(glamourPath, "../")
			}
			cleanStripped := strings.TrimPrefix(stripped, "/")
			isImageLine := strings.HasPrefix(stripped, "Image: ") ||
				stripped == src ||
				stripped == "/"+glamourPath ||
				cleanStripped == glamourPath ||
				strings.Contains(stripped, "🖼️")
			if isImageLine {
				result = append(result, kittySeq)
				replaced = true
				break
			}
		}
		if !replaced {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}
