// Package tui provides the interactive terminal user interface for gomd.
package tui

import (
	"regexp"
	"sort"
	"strings"

	"charm.land/glamour/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// ─────────────────────────────────────────────
// Section helpers
// ─────────────────────────────────────────────

// ensureGlamourRenderer returns a cached glamour renderer for the given inner width
// and current theme, rebuilding it only when those change.
func (a *App) ensureGlamourRenderer(innerW int) interface{ Render(string) (string, error) } {
	themeName := a.theme.Name
	if a.glamourRenderer != nil && a.glamourWidth == innerW && a.glamourStyleName == themeName {
		return a.glamourRenderer
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(StyleConfigFromTheme(a.theme)),
		glamour.WithWordWrap(innerW),
		glamour.WithChromaFormatter("terminal16m"),
	)
	if err != nil {
		return nil
	}
	a.glamourRenderer = r
	a.glamourWidth = innerW
	a.glamourStyleName = themeName
	return r
}

// expandTabs replaces tab characters in s with spaces, advancing to the next
// 8-column tab stop. ANSI escape sequences are skipped (zero-width) so column
// counting stays accurate even in styled output.
func expandTabs(s string) string {
	if !strings.ContainsRune(s, '\t') {
		return s
	}
	var buf strings.Builder
	col := 0
	inEsc := false
	for _, r := range s {
		switch {
		case inEsc:
			buf.WriteRune(r)
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEsc = false
			}
		case r == '\x1b':
			inEsc = true
			buf.WriteRune(r)
		case r == '\t':
			spaces := tabStopWidth - (col % tabStopWidth)
			for i := 0; i < spaces; i++ {
				buf.WriteByte(' ')
			}
			col += spaces
		default:
			buf.WriteRune(r)
			col++
		}
	}
	return buf.String()
}

// imgTagRe matches <img ... /> or <img ...> HTML tags.
var imgTagRe = regexp.MustCompile(`(?i)<img\s[^>]*>`)

// aTagRe matches <a href="...">text</a> HTML tags.
var aTagRe = regexp.MustCompile(`(?i)<a\s[^>]*href\s*=\s*["']([^"']*)["'][^>]*>(.*?)</a>`)

// PreProcessMarkdown converts HTML image and anchor tags into their markdown
// equivalents so that glamour renders them properly. This must be called
// before passing content to glamour.
func PreProcessMarkdown(md string) string {
	out, _, _ := preProcess(md)
	return out
}

// preProcess is PreProcessMarkdown that also returns its two passes of
// rewrites, to map offsets in the result back to md: imgs then anchors.
func preProcess(md string) (string, rewrites, rewrites) {
	// Replace <img ... src="url" alt="text" ... /> with ![text](url)
	inCode := codeChecker(md)
	md, imgs := replaceAll(imgTagRe, md, func(m []int) (string, bool) {
		if inCode(m[0]) {
			return "", false
		}
		tag := md[m[0]:m[1]]
		src := extractHTMLAttr(tag, "src")
		if src == "" {
			return "", false
		}
		return "![" + extractHTMLAttr(tag, "alt") + "](" + src + ")", true
	})

	// Replace <a href="url">text</a> with [text](url)
	inCode = codeChecker(md)
	md, anchors := replaceAll(aTagRe, md, func(m []int) (string, bool) {
		if inCode(m[0]) {
			return "", false
		}
		return "[" + md[m[4]:m[5]] + "](" + md[m[2]:m[3]] + ")", true
	})
	return md, imgs, anchors
}

// codeChecker returns a function reporting whether byte offset off of md
// lies in a code block or code span, where HTML is shown as written. md is
// parsed on the first call, so text without HTML tags costs nothing.
func codeChecker(md string) func(off int) bool {
	var ranges [][2]int
	parsed := false
	return func(off int) bool {
		if !parsed {
			parsed = true
			src := []byte(md)
			doc := goldmark.New().Parser().Parse(text.NewReader(src))
			ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
				if !entering {
					return ast.WalkContinue, nil
				}
				switch n.(type) {
				case *ast.FencedCodeBlock, *ast.CodeBlock:
					for i := 0; i < n.Lines().Len(); i++ {
						s := n.Lines().At(i)
						ranges = append(ranges, [2]int{s.Start, s.Stop})
					}
					return ast.WalkSkipChildren, nil
				case *ast.CodeSpan:
					for c := n.FirstChild(); c != nil; c = c.NextSibling() {
						if t, ok := c.(*ast.Text); ok {
							ranges = append(ranges, [2]int{t.Segment.Start, t.Segment.Stop})
						}
					}
					return ast.WalkSkipChildren, nil
				}
				return ast.WalkContinue, nil
			})
		}
		for _, r := range ranges {
			if off >= r[0] && off < r[1] {
				return true
			}
		}
		return false
	}
}

// A rewrite replaced bytes [in0, in1) of a text with [out0, out1) of the result.
type rewrite struct{ in0, in1, out0, out1 int }

type rewrites []rewrite

// back maps offset off of the result to the text before the rewrites. An
// offset inside a rewrite maps to its start, or to its end when end is set.
func (rs rewrites) back(off int, end bool) int {
	delta := 0
	for _, r := range rs {
		switch {
		case off < r.out0 || off == r.out0 && end:
			return off + delta
		case off < r.out1 || off == r.out1 && end:
			if end {
				return r.in1
			}
			return r.in0
		}
		delta = r.in1 - r.out1
	}
	return off + delta
}

// replaceAll replaces the matches of re in s for which repl returns true.
func replaceAll(re *regexp.Regexp, s string, repl func(m []int) (string, bool)) (string, rewrites) {
	var sb strings.Builder
	var rs rewrites
	last := 0
	for _, m := range re.FindAllStringSubmatchIndex(s, -1) {
		r, ok := repl(m)
		if !ok {
			continue
		}
		sb.WriteString(s[last:m[0]])
		out0 := sb.Len()
		sb.WriteString(r)
		rs = append(rs, rewrite{m[0], m[1], out0, sb.Len()})
		last = m[1]
	}
	if rs == nil {
		return s, nil
	}
	sb.WriteString(s[last:])
	return sb.String(), rs
}

// PostProcessLinks inserts a 🔗 glyph between link text and URL in
// glamour-rendered output. Glamour renders [text](url) as:
//
//	OSC8(url) STYLED_TEXT RESET OSC8_RESET FG_STYLE SPACE RESET LINK_STYLE OSC8(url) URL ...
//
// We find the OSC8 reset after the text and insert " 🔗" before the space.
var linkGlyphRe = regexp.MustCompile(`(\x1b\]8;;\x07)((?:\x1b\[[0-9;]*m)*)([ \t])((?:\x1b\[[0-9;]*m)*\x1b\]8;id=)`)

// PostProcessLinks adds a link glyph between the text and URL portions of
// rendered links. It should be called on the full rendered output string.
func PostProcessLinks(s string) string {
	return linkGlyphRe.ReplaceAllString(s, "${1} 🔗${2}${3}${4}")
}

// renderGlamour renders a markdown string through glamour and returns the output
// split into lines, with a trailing empty line stripped.
func (a *App) renderGlamour(markdown string, innerW int) []string {
	r := a.ensureGlamourRenderer(innerW)
	if r == nil {
		return strings.Split(markdown, "\n")
	}
	markdown = PreProcessMarkdown(markdown)
	out, err := r.Render(markdown)
	if err != nil {
		return strings.Split(markdown, "\n")
	}
	out = PostProcessLinks(out)

	// glamour always ends with a newline; split and drop the trailing empty entry
	lines := strings.Split(out, "\n")
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	// Expand any tab characters left in the glamour output so that
	// lipgloss.Width() and ansi.Truncate() measure the correct column widths.
	for i, l := range lines {
		lines[i] = expandTabs(l)
	}
	if a.cfg.Images.Enabled {
		var uploads []string
		lines, uploads = placeImages(lines, markdown, a.filepath, innerW-4, true, a.images)
		a.pendingUploads = append(a.pendingUploads, uploads...)
	}
	return lines
}

// rebuildSection recomputes sectionLines and codeNodes for the selected heading.
func (a *App) rebuildSection() {
	a.contentOffset = 0
	a.nodeSelIdx = 0
	a.codeNodes = nil
	a.nodeRenderInfo = nil
	a.renderedLines = nil

	// selectedIdx == -1 means the root (Document) node: show entire file.
	//
	// Node extraction (extractCodeNodes → extractLinkNodes) runs on the raw
	// markdown lines, *before* PreProcessMarkdown converts HTML tags to
	// markdown equivalents. This is intentional: the extracted colStart/colEnd
	// byte offsets must match the raw source so that highlightSpanInLine can
	// highlight the correct column range in the rendered output. Running
	// extraction after pre-processing would shift those offsets because
	// PreProcessMarkdown changes line content and length (e.g. a long <img>
	// tag becomes a shorter ![alt](url)). The trade-off is that extraction
	// must handle both markdown and HTML syntax for links/images.
	if a.selectedIdx < 0 || len(a.doc.Headings) == 0 {
		content := strings.TrimRight(a.doc.Content, "\n")
		a.sectionLines = strings.Split(content, "\n")
		a.codeNodes = extractCodeNodes(a.sectionLines)
		return
	}

	if a.selectedIdx >= len(a.doc.Headings) {
		a.selectedIdx = len(a.doc.Headings) - 1
	}

	h := a.doc.Headings[a.selectedIdx]

	// Byte range for section
	start := h.Offset
	end := len(a.doc.Content)
	for i := a.selectedIdx + 1; i < len(a.doc.Headings); i++ {
		if a.doc.Headings[i].Level <= h.Level {
			end = a.doc.Headings[i].Offset
			break
		}
	}

	section := a.doc.Content[start:end]
	section = strings.TrimRight(section, "\n")
	a.sectionLines = strings.Split(section, "\n")
	a.codeNodes = extractCodeNodes(a.sectionLines)
}

// extractCodeNodes finds all fenced code blocks and inline backtick spans in lines.
func extractCodeNodes(lines []string) []codeNode {
	var nodes []codeNode
	inBlock := false
	var fence, lang string
	var bodyLines []string
	var startLine int

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inBlock {
			if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
				inBlock = true
				fence = trimmed[:3]
				lang = strings.TrimSpace(trimmed[3:])
				bodyLines = nil
				startLine = i
			} else {
				// Scan for inline backtick spans and links (including on heading lines)
				nodes = append(nodes, extractInlineNodes(line, i)...)
				nodes = append(nodes, extractLinkNodes(line, i)...)
			}
		} else {
			if strings.HasPrefix(trimmed, fence) {
				nodes = append(nodes, codeNode{
					kind:      nodeCodeBlock,
					lang:      lang,
					content:   strings.Join(bodyLines, "\n"),
					startLine: startLine,
					endLine:   i,
				})
				inBlock = false
				fence = ""
				lang = ""
				bodyLines = nil
			} else {
				bodyLines = append(bodyLines, line)
			}
		}
	}
	// Sort nodes by source position (line, then column) so that
	// mapNodesToRenderedLines processes them in document order. Without this,
	// inline code spans (extracted first) and link nodes (extracted second) on
	// the same line may be out of column order, causing the forward-scanning
	// mapper to skip earlier occurrences.
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].startLine != nodes[j].startLine {
			return nodes[i].startLine < nodes[j].startLine
		}
		return nodes[i].colStart < nodes[j].colStart
	})
	return nodes
}

// extractInlineNodes finds all `code` spans in a single line and returns them as codeNodes.
func extractInlineNodes(line string, lineIdx int) []codeNode {
	var nodes []codeNode
	i := 0
	for i < len(line) {
		if line[i] != '`' {
			i++
			continue
		}
		// Count opening backticks
		j := i
		for j < len(line) && line[j] == '`' {
			j++
		}
		tick := line[i:j] // one or more backticks
		// Find matching closing sequence
		k := j
		for k < len(line) {
			closeStart := strings.Index(line[k:], tick)
			if closeStart == -1 {
				break
			}
			closeStart += k
			// Make sure it's not longer than tick (e.g. `` inside ``` context)
			// Check that the char before or after isn't another backtick
			closeEnd := closeStart + len(tick)
			// If surrounded by more backticks, skip
			if closeStart > 0 && line[closeStart-1] == '`' {
				k = closeStart + 1
				continue
			}
			if closeEnd < len(line) && line[closeEnd] == '`' {
				k = closeEnd
				continue
			}
			// Found a valid span
			inner := line[j:closeStart]
			// Trim single leading/trailing space per CommonMark
			if len(inner) > 2 && inner[0] == ' ' && inner[len(inner)-1] == ' ' {
				inner = inner[1 : len(inner)-1]
			}
			nodes = append(nodes, codeNode{
				kind:      nodeInlineCode,
				lang:      "",
				content:   inner,
				startLine: lineIdx,
				endLine:   lineIdx,
				inline:    true,
				colStart:  i,
				colEnd:    closeEnd,
			})
			i = closeEnd
			goto nextChar
		}
		i = j
	nextChar:
	}
	return nodes
}

// linkDestination drops an optional title from a link's parenthesised part:
// `x.png "A title"` -> `x.png`.
func linkDestination(dest string) string {
	if i := strings.IndexAny(dest, " \t"); i >= 0 {
		return dest[:i]
	}
	return dest
}

// extractLinkNodes finds markdown links [text](url) and autolinks <url> in a line.
func extractLinkNodes(line string, lineIdx int) []codeNode {
	var nodes []codeNode
	i := 0
	for i < len(line) {
		// Check for ![alt](url) image link or [text](url) regular link
		if line[i] == '!' && i+1 < len(line) && line[i+1] == '[' {
			// Image link ![alt](url)
			j := i + 2
			depth := 1
			for j < len(line) && depth > 0 {
				if line[j] == '[' {
					depth++
				} else if line[j] == ']' {
					depth--
				}
				j++
			}
			if depth == 0 && j < len(line) && line[j] == '(' {
				k := j + 1
				for k < len(line) && line[k] != ')' {
					k++
				}
				if k < len(line) {
					url := linkDestination(line[j+1 : k])
					alt := line[i+2 : j-1]
					display := imageCaption(alt, url)
					nodes = append(nodes, codeNode{
						kind:      nodeLink,
						lang:      "link",
						content:   url,
						display:   display,
						startLine: lineIdx,
						endLine:   lineIdx,
						inline:    true,
						colStart:  i,
						colEnd:    k + 1,
					})
					i = k + 1
					continue
				}
			}
			i = j
		} else if line[i] == '[' {
			// Regular link [text](url)
			j := i + 1
			depth := 1
			for j < len(line) && depth > 0 {
				if line[j] == '[' {
					depth++
				} else if line[j] == ']' {
					depth--
				}
				j++
			}
			if depth == 0 && j < len(line) && line[j] == '(' {
				k := j + 1
				for k < len(line) && line[k] != ')' {
					k++
				}
				if k < len(line) {
					url := linkDestination(line[j+1 : k])
					text := line[i+1 : j-1]
					// Glamour renders [text](url) as "text" (with OSC8 hyperlink)
					nodes = append(nodes, codeNode{
						kind:      nodeLink,
						lang:      "link",
						content:   url,
						display:   text,
						startLine: lineIdx,
						endLine:   lineIdx,
						inline:    true,
						colStart:  i,
						colEnd:    k + 1,
					})
					i = k + 1
					continue
				}
			}
			i = j
		} else if line[i] == '<' && i+1 < len(line) {
			// Check for <img ... src="url" ... /> HTML tag
			if i+4 < len(line) && strings.ToLower(line[i+1:i+4]) == "img" && (line[i+4] == ' ' || line[i+4] == '\t') {
				// Find the closing >
				closeIdx := strings.Index(line[i:], ">")
				if closeIdx >= 0 {
					tag := line[i : i+closeIdx+1]
					// Extract src attribute
					src := extractHTMLAttr(tag, "src")
					alt := extractHTMLAttr(tag, "alt")
					if src != "" {
						display := imageCaption(alt, src)
						nodes = append(nodes, codeNode{
							kind:      nodeLink,
							lang:      "link",
							content:   src,
							display:   display,
							startLine: lineIdx,
							endLine:   lineIdx,
							inline:    true,
							colStart:  i,
							colEnd:    i + closeIdx + 1,
						})
					}
					i = i + closeIdx + 1
					continue
				}
			}
			// Check for <a href="url">text</a> HTML tag
			if i+2 < len(line) && strings.ToLower(line[i+1:i+2]) == "a" && i+2 < len(line) && (line[i+2] == ' ' || line[i+2] == '\t') {
				m := aTagRe.FindStringIndex(line[i:])
				if m != nil {
					sm := aTagRe.FindStringSubmatch(line[i:])
					if sm != nil {
						href := sm[1]
						text := sm[2]
						nodes = append(nodes, codeNode{
							kind:      nodeLink,
							lang:      "link",
							content:   href,
							display:   text,
							startLine: lineIdx,
							endLine:   lineIdx,
							inline:    true,
							colStart:  i,
							colEnd:    i + m[1],
						})
						i = i + m[1]
						continue
					}
				}
			}
			// Check for autolink <url>
			j := i + 1
			for j < len(line) && line[j] != '>' && line[j] != ' ' {
				j++
			}
			if j < len(line) && line[j] == '>' {
				url := line[i+1 : j]
				// Basic validation: contains :// or @ (link/email)
				if strings.Contains(url, "://") || strings.Contains(url, "@") {
					nodes = append(nodes, codeNode{
						kind:      nodeLink,
						lang:      "link",
						content:   url,
						display:   url,
						startLine: lineIdx,
						endLine:   lineIdx,
						inline:    true,
						colStart:  i,
						colEnd:    j + 1,
					})
				}
				i = j + 1
			} else {
				i++
			}
		} else if i+8 < len(line) && (line[i:i+8] == "https://" || line[i:i+7] == "http://") {
			// Bare URL (not inside [] or <>)
			j := i
			for j < len(line) && line[j] != ' ' && line[j] != '\t' && line[j] != ')' && line[j] != '>' {
				j++
			}
			// Trim trailing punctuation that's unlikely part of the URL
			for j > i && (line[j-1] == '.' || line[j-1] == ',' || line[j-1] == ';' || line[j-1] == ':') {
				j--
			}
			url := line[i:j]
			nodes = append(nodes, codeNode{
				kind:      nodeLink,
				lang:      "link",
				content:   url,
				display:   url,
				startLine: lineIdx,
				endLine:   lineIdx,
				inline:    true,
				colStart:  i,
				colEnd:    j,
			})
			i = j
		} else {
			i++
		}
	}
	return nodes
}

// extractHTMLAttr extracts the value of an HTML attribute from a tag string.
// Returns empty string if the attribute is not found.
func extractHTMLAttr(tag, attr string) string {
	// Search for attr= (case-insensitive)
	lower := strings.ToLower(tag)
	key := strings.ToLower(attr) + "="
	idx := strings.Index(lower, key)
	if idx < 0 {
		return ""
	}
	rest := tag[idx+len(key):]
	if len(rest) == 0 {
		return ""
	}
	if rest[0] == '"' || rest[0] == '\'' {
		quote := rest[0]
		end := strings.IndexByte(rest[1:], quote)
		if end < 0 {
			return ""
		}
		return rest[1 : 1+end]
	}
	// Unquoted: read until space or >
	end := strings.IndexAny(rest, " \t>")
	if end < 0 {
		return rest
	}
	return rest[:end]
}

// filteredNodeIndices returns the indices into a.codeNodes for each node
// matching the current sub-mode filter.
func (a *App) filteredNodeIndices() []int {
	var indices []int
	for i, n := range a.codeNodes {
		if a.nodeSubMode == nodeAll || n.kind == a.nodeSubMode {
			indices = append(indices, i)
		}
	}
	return indices
}

// filteredNodes returns only the codeNodes matching the current sub-mode.
func (a *App) filteredNodes() []codeNode {
	indices := a.filteredNodeIndices()
	result := make([]codeNode, len(indices))
	for i, idx := range indices {
		result[i] = a.codeNodes[idx]
	}
	return result
}
