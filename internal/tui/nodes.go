// Package tui provides the interactive terminal user interface for gomd.
package tui

import (
	"regexp"
	"sort"
	"strings"

	"charm.land/glamour/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
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
		// Links in tables stay in their cell, as everywhere else, instead of
		// footnotes under the table that the source map cannot follow.
		glamour.WithInlineTableLinks(true),
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
	before := md
	md, anchors := replaceAll(aTagRe, md, func(m []int) (string, bool) {
		if inCode(m[0]) {
			return "", false
		}
		return "[" + md[m[4]:m[5]] + "](" + md[m[2]:m[3]] + ")", true
	})
	return md, imgs, splitAnchorText(before, anchors)
}

// splitAnchorText splits each rewrite of an <a> tag in two around the link
// text, which "[text](url)" copies unchanged, so the text maps back to its
// own bytes and only the markup around it maps to the tag.
func splitAnchorText(src string, rs rewrites) rewrites {
	var out rewrites
	for _, r := range rs {
		m := aTagRe.FindStringSubmatchIndex(src[r.in0:r.in1])
		t0, t1 := r.in0+m[4], r.in0+m[5]
		out = append(out,
			rewrite{r.in0, t0, r.out0, r.out0 + 1},             // the opening tag became "["
			rewrite{t1, r.in1, r.out0 + 1 + (t1 - t0), r.out1}) // "</a>" became "](url)"
	}
	return out
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
// Table rows are left alone: glamour has laid out their borders, and the
// glyph would push them out of line.
func PostProcessLinks(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if !isTableLine(strings.TrimSpace(ansi.Strip(l))) {
			lines[i] = linkGlyphRe.ReplaceAllString(l, "${1} 🔗${2}${3}${4}")
		}
	}
	return strings.Join(lines, "\n")
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
		var remote func(string) string
		if a.remote != nil {
			remote = a.remoteImage
		}
		lines, uploads = placeImages(lines, markdown, a.filepath, innerW-4, true, a.images, remote)
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

	section := a.doc.Content[a.doc.Headings[a.selectedIdx].Offset:a.doc.SectionEnd(a.selectedIdx)]
	section = strings.TrimRight(section, "\n")
	a.sectionLines = strings.Split(section, "\n")
	a.codeNodes = extractCodeNodes(a.sectionLines)
}

// extractMoreCodeBlocks returns the code blocks the line scanner in
// extractCodeNodes misses, found by goldmark: indented code blocks and
// fenced blocks inside block quotes. A block on lines of one in found is
// left out.
func extractMoreCodeBlocks(lines []string, found []codeNode) []codeNode {
	src := []byte(strings.Join(lines, "\n"))
	off := make([]int, len(lines)) // byte offset of each line
	for i := 1; i < len(lines); i++ {
		off[i] = off[i-1] + len(lines[i-1]) + 1
	}
	lineOf := func(pos int) int { return sort.SearchInts(off, pos+1) - 1 }
	taken := func(first, last int) bool {
		for _, n := range found {
			if n.kind == nodeCodeBlock && first <= n.endLine && last >= n.startLine {
				return true
			}
		}
		return false
	}
	inQuote := func(n ast.Node) bool {
		for p := n.Parent(); p != nil; p = p.Parent() {
			if _, ok := p.(*ast.Blockquote); ok {
				return true
			}
		}
		return false
	}
	var nodes []codeNode
	doc := goldmark.New().Parser().Parse(text.NewReader(src))
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		var lang string
		indented := false
		switch b := n.(type) {
		case *ast.CodeBlock:
			indented = true
		case *ast.FencedCodeBlock:
			if !inQuote(b) {
				return ast.WalkContinue, nil
			}
			lang = string(b.Language(src))
		default:
			return ast.WalkContinue, nil
		}
		segs := n.Lines()
		var body strings.Builder
		for i := 0; i < segs.Len(); i++ {
			seg := segs.At(i)
			body.Write(seg.Value(src))
		}
		content := strings.TrimRight(body.String(), "\n")
		if strings.TrimSpace(content) == "" {
			return ast.WalkSkipChildren, nil
		}
		first, last := lineOf(segs.At(0).Start), lineOf(segs.At(segs.Len()-1).Start)
		if !indented { // the fence lines around the body
			first, last = first-1, min(last+1, len(lines)-1)
		}
		if !taken(first, last) {
			nodes = append(nodes, codeNode{kind: nodeCodeBlock, lang: lang, content: content,
				startLine: first, endLine: last, indented: indented})
		}
		return ast.WalkSkipChildren, nil
	})
	return nodes
}

// extractCodeNodes finds all fenced code blocks and inline backtick spans in lines.
func extractCodeNodes(lines []string) []codeNode {
	var nodes []codeNode
	inBlock := false
	var fence, lang string
	var bodyLines []string
	var startLine, indent int

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inBlock {
			if f := fenceRun(trimmed); f != "" {
				inBlock = true
				fence = f
				lang = strings.TrimSpace(trimmed[len(f):])
				bodyLines = nil
				startLine = i
				indent = len(line) - len(strings.TrimLeft(line, " "))
			} else {
				// Scan for inline backtick spans and links (including on heading
				// lines). A link reference definition is not shown, so it has none.
				if linkDefRe.MatchString(line) {
					continue
				}
				inline := extractInlineNodes(line, i)
				if strings.HasPrefix(trimmed, "#") {
					for j := range inline {
						inline[j].heading = true
					}
				}
				nodes = append(nodes, inline...)
				nodes = append(nodes, extractLinkNodes(line, i)...)
			}
		} else {
			// A closing fence is the opening character repeated at least as
			// often, with nothing else on the line.
			if strings.HasPrefix(trimmed, fence) && strings.Trim(trimmed, fence[:1]) == "" {
				// A block with no text has nothing to show or copy.
				if content := strings.Join(bodyLines, "\n"); strings.TrimSpace(content) != "" {
					nodes = append(nodes, codeNode{
						kind:      nodeCodeBlock,
						lang:      lang,
						content:   content,
						startLine: startLine,
						endLine:   i,
					})
				}
				inBlock = false
				fence = ""
				lang = ""
				bodyLines = nil
			} else {
				// As in CommonMark, the body loses up to the fence's
				// indentation (a block inside a list item).
				n := min(indent, len(line)-len(strings.TrimLeft(line, " ")))
				bodyLines = append(bodyLines, line[n:])
			}
		}
	}
	nodes = append(nodes, extractMoreCodeBlocks(lines, nodes)...)
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

// linkDefRe matches a link reference definition line ("[1]: https://x
// "title""), which glamour does not show.
var linkDefRe = regexp.MustCompile(`^ {0,3}\[[^\]]+\]:[ \t]*\S+([ \t]+("[^"]*"|'[^']*'|\([^)]*\)))?[ \t]*$`)

// fenceRun returns the opening code fence that starts line, three or more
// backticks or tildes, or "" when there is none.
func fenceRun(line string) string {
	if line == "" || line[0] != '`' && line[0] != '~' {
		return ""
	}
	n := len(line) - len(strings.TrimLeft(line, line[:1]))
	if n < 3 {
		return ""
	}
	return line[:n]
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

// extractNodes finds the code blocks, code spans and links of lines as
// glamour shows them: it parses them as glamour does, after PreProcessMarkdown,
// and maps every position back to lines. Of an HTML block, which goldmark does
// not parse, it takes the <a href> and <img src> tags.
func extractNodes(lines []string) []codeNode {
	raw := strings.Join(lines, "\n")
	pre, imgs, anchors := preProcess(raw)
	b := []byte(pre)
	back := func(off int, end bool) int { return imgs.back(anchors.back(off, end), end) }
	off := make([]int, len(lines)) // byte offset of each line in raw
	for i := 1; i < len(lines); i++ {
		off[i] = off[i-1] + len(lines[i-1]) + 1
	}
	lineOf := func(rawOff int) int { return sort.SearchInts(off, rawOff+1) - 1 }
	var nodes []codeNode
	// add adds inline node n over raw bytes [start, end); colEnd is past the
	// end of its first line when it spans lines.
	add := func(n codeNode, start, end int) {
		n.startLine, n.endLine = lineOf(start), lineOf(max(end-1, start))
		n.colStart, n.colEnd = start-off[n.startLine], end-off[n.startLine]
		n.inline = true
		nodes = append(nodes, n)
	}
	addPre := func(n codeNode, start, end int) { add(n, back(start, false), back(end, true)) }
	doc := goldmark.New(goldmark.WithExtensions(extension.GFM, extension.DefinitionList)).Parser().Parse(text.NewReader(b))
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch x := n.(type) {
		case *ast.FencedCodeBlock, *ast.CodeBlock:
			segs := n.Lines()
			var body strings.Builder
			for i := 0; i < segs.Len(); i++ {
				seg := segs.At(i)
				body.WriteString(strings.Repeat(" ", seg.Padding)) // a tab partly taken by the indentation
				body.Write(seg.Value(b))
			}
			// A block with no text has nothing to show or copy.
			content := strings.TrimRight(body.String(), "\n")
			if strings.TrimSpace(content) == "" {
				return ast.WalkSkipChildren, nil
			}
			c := codeNode{kind: nodeCodeBlock, content: content}
			first := lineOf(back(segs.At(0).Start, false))
			last := lineOf(back(segs.At(segs.Len()-1).Start, false))
			if f, ok := x.(*ast.FencedCodeBlock); ok {
				c.lang = string(f.Language(b))
				first, last = first-1, min(last+1, len(lines)-1) // the fence lines
			} else {
				c.indented = true
			}
			c.startLine, c.endLine = first, last
			nodes = append(nodes, c)
			return ast.WalkSkipChildren, nil
		case *ast.HTMLBlock:
			segs := n.Lines()
			if segs.Len() > 0 {
				htmlTagNodes(raw, back(segs.At(0).Start, false), back(segs.At(segs.Len()-1).Stop, true), add)
			}
			return ast.WalkSkipChildren, nil
		case *ast.CodeSpan:
			if x.Pos() < 0 {
				return ast.WalkSkipChildren, nil
			}
			var content strings.Builder
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if t, ok := c.(*ast.Text); ok {
					content.Write(t.Segment.Value(b))
				}
			}
			addPre(codeNode{kind: nodeInlineCode, content: strings.ReplaceAll(content.String(), "\n", " "), heading: inHeading(n)},
				x.Pos(), codeSpanEnd(pre, x))
			return ast.WalkSkipChildren, nil
		case *ast.Link:
			if p := x.Pos(); p >= 0 {
				textEnd, end := linkEnd(pre, lastStop(pre, n, p+1))
				addPre(codeNode{kind: nodeLink, lang: "link", content: string(x.Destination), display: pre[p+1 : textEnd]}, p, end)
			}
		case *ast.Image:
			if p := x.Pos(); p >= 0 {
				textEnd, end := linkEnd(pre, lastStop(pre, n, p+2))
				addPre(codeNode{kind: nodeLink, lang: "link", content: string(x.Destination),
					display: imageCaption(pre[p+2:textEnd], string(x.Destination))}, p, end)
			}
			return ast.WalkSkipChildren, nil
		case *ast.AutoLink:
			p, label := x.Pos(), string(x.Label(b))
			if p < 0 {
				break
			}
			if !strings.HasPrefix(pre[p:], "<") && !strings.HasPrefix(pre[p:], label) {
				p += max(strings.Index(pre[p:], label), 0) // linkify can point before the URL
			}
			end := p + len(label)
			if pre[p] == '<' {
				end += 2
			}
			addPre(codeNode{kind: nodeLink, lang: "link", content: string(x.URL(b)), display: label}, p, end)
		}
		return ast.WalkContinue, nil
	})
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].startLine != nodes[j].startLine {
			return nodes[i].startLine < nodes[j].startLine
		}
		return nodes[i].colStart < nodes[j].colStart
	})
	return nodes
}

// htmlTagNodes passes to add the link nodes of the <a href> and <img src>
// tags in raw[start:end].
func htmlTagNodes(raw string, start, end int, add func(n codeNode, start, end int)) {
	for _, m := range aTagRe.FindAllStringSubmatchIndex(raw[start:end], -1) {
		add(codeNode{kind: nodeLink, lang: "link", content: raw[start+m[2] : start+m[3]], display: raw[start+m[4] : start+m[5]]},
			start+m[0], start+m[1])
	}
	for _, m := range imgTagRe.FindAllStringIndex(raw[start:end], -1) {
		tag := raw[start+m[0] : start+m[1]]
		if src := extractHTMLAttr(tag, "src"); src != "" {
			add(codeNode{kind: nodeLink, lang: "link", content: src, display: imageCaption(extractHTMLAttr(tag, "alt"), src)},
				start+m[0], start+m[1])
		}
	}
}

// inHeading reports whether n is inside a heading.
func inHeading(n ast.Node) bool {
	for p := n.Parent(); p != nil; p = p.Parent() {
		if _, ok := p.(*ast.Heading); ok {
			return true
		}
	}
	return false
}

// goldmark records where a link, an image or a code span starts but not
// where it ends: the functions below find the end in the source s.

// codeSpanEnd returns where the closing backticks of code span n end: after
// its code, the next run of as many backticks as open it.
func codeSpanEnd(s string, n *ast.CodeSpan) int {
	p := n.Pos()
	ticks := len(s[p:]) - len(strings.TrimLeft(s[p:], "`"))
	end := p + ticks
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			end = t.Segment.Stop
		}
	}
	for end < len(s) {
		k := strings.Index(s[end:], s[p:p+ticks])
		if k < 0 {
			break
		}
		end += k
		run := len(s[end:]) - len(strings.TrimLeft(s[end:], "`"))
		if run == ticks {
			return end + ticks
		}
		end += run
	}
	return end
}

// lastStop returns where the last element inside link or image n ends, or
// from when it has none: the link's text ends at the next "]".
func lastStop(s string, n ast.Node, from int) int {
	stop := from
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch c := c.(type) {
		case *ast.Text:
			stop = max(stop, c.Segment.Stop)
		case *ast.RawHTML:
			if c.Segments.Len() > 0 {
				stop = max(stop, c.Segments.At(c.Segments.Len()-1).Stop)
			}
		case *ast.CodeSpan:
			if c.Pos() >= 0 {
				stop = max(stop, codeSpanEnd(s, c))
			}
		case *ast.Image:
			if c.Pos() >= 0 {
				_, end := linkEnd(s, lastStop(s, c, c.Pos()+2))
				stop = max(stop, end)
			}
		case *ast.AutoLink:
			if c.Pos() >= 0 {
				stop = max(stop, c.Pos()+len(c.Label([]byte(s))))
			}
		default:
			stop = max(stop, lastStop(s, c, stop))
		}
	}
	return stop
}

// linkEnd returns, for the link whose text ends at the first "]" from i in
// s, where its text ends and where its markup ends: after "(destination
// "title")", "[label]" or the "]" alone.
func linkEnd(s string, i int) (textEnd, end int) {
	for i < len(s) && s[i] != ']' {
		if s[i] == '\\' {
			i++
		}
		i++
	}
	textEnd, end = min(i, len(s)), min(i+1, len(s))
	if end < len(s) && s[end] == '[' {
		if k := strings.IndexByte(s[end:], ']'); k >= 0 {
			return textEnd, end + k + 1
		}
		return textEnd, end
	}
	if end >= len(s) || s[end] != '(' {
		return textEnd, end
	}
	j := end + 1
	skipSpace := func() {
		for j < len(s) && strings.IndexByte(" \t\r\n", s[j]) >= 0 {
			j++
		}
	}
	skipSpace()
	if j < len(s) && s[j] == '<' { // <destination>
		if k := strings.IndexByte(s[j:], '>'); k >= 0 {
			j += k + 1
		}
	} else { // a destination with balanced parentheses
		for depth := 0; j < len(s) && strings.IndexByte(" \t\r\n", s[j]) < 0; j++ {
			if s[j] == '\\' {
				j++
			} else if s[j] == '(' {
				depth++
			} else if s[j] == ')' {
				if depth == 0 {
					break
				}
				depth--
			}
		}
	}
	skipSpace()
	if j < len(s) && strings.IndexByte(`"'(`, s[j]) >= 0 { // a title
		closer := s[j]
		if closer == '(' {
			closer = ')'
		}
		for j++; j < len(s) && s[j] != closer; j++ {
			if s[j] == '\\' {
				j++
			}
		}
		j++
		skipSpace()
	}
	if j < len(s) && s[j] == ')' {
		return textEnd, j + 1
	}
	return textEnd, end
}
