package tui

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// A sourceMap ties the rendered content to its markdown source, so a mouse
// selection can be copied as raw markdown. glamour does not report where its
// output comes from, so the map aligns the rendered characters, in reading
// order, with the characters goldmark says are visible: text, code and link
// destinations. Rendered characters left unmatched are decorations (bullets,
// borders, 🔗). Selections snap outward to whole elements (emphasis, code
// spans, links, images, tables, paragraphs with HTML), so the highlight, the
// rendered copy and the markdown copy cover the same content.
type sourceMap struct {
	src    string
	units  []srcUnit  // visible source characters, in source order
	elems  []srcElem  // elements a selection must not cut
	fences []srcFence // fenced code blocks
	cells  []srcCell  // rendered characters matched to a unit, in reading order
	// lineFirst and lineLast hold the first and last non-space column of
	// each rendered line, or -1 for a blank line.
	lineFirst, lineLast []int
}

type srcUnit struct {
	r          rune
	start, end int // byte range in src
	elem       int // index in elems, or -1
}

type srcElem struct {
	start, end int // byte range in src, markup included
	c0, c1     int // first and last matched cell, c0 > c1 when none
}

type srcFence struct {
	u0, u1     int // first and last unit of the code
	start, end int // byte range in src, fence lines included
}

type srcCell struct {
	pos  textPos
	w    int
	unit int
}

// Characters that open or close inline markup next to an element's text.
const (
	openMarks  = "*_~`[!<"
	closeMarks = "*_~`>"
)

// sourceLineAt returns the 0-based line of src holding the first source
// character rendered at p or after it, or -1 when there is none.
func (m *sourceMap) sourceLineAt(p textPos) int {
	c := sort.Search(len(m.cells), func(i int) bool { return !before(m.cells[i].pos, p) })
	if c == len(m.cells) {
		return -1
	}
	return strings.Count(m.src[:m.units[m.cells[c].unit].start], "\n")
}

// buildSourceMap maps rendered, the render of the markdown src. It reads
// src as glamour does, after PreProcessMarkdown, and maps every offset back.
func buildSourceMap(src string, rendered []string) *sourceMap {
	pre, imgs, anchors := preProcess(src)
	m := &sourceMap{src: pre}
	m.collect()
	back := func(off *int, end bool) { *off = imgs.back(anchors.back(*off, end), end) }
	for i := range m.units {
		back(&m.units[i].start, false)
		back(&m.units[i].end, true)
	}
	for i := range m.elems {
		if m.elems[i].start >= 0 {
			back(&m.elems[i].start, false)
			back(&m.elems[i].end, true)
		}
	}
	for i := range m.fences {
		back(&m.fences[i].start, false)
		back(&m.fences[i].end, true)
	}
	m.src = src
	m.align(rendered)
	return m
}

// collect walks the goldmark AST, parsed as glamour parses it, for the
// visible characters, the elements and the fenced code blocks. HTML blocks
// count as visible: glamour shows their text, so their tags go unmatched.
func (m *sourceMap) collect() {
	b := []byte(m.src)
	doc := goldmark.New(goldmark.WithExtensions(extension.GFM, extension.DefinitionList)).Parser().Parse(text.NewReader(b))
	cur, curU0 := -1, 0 // open element and its first unit
	var curNode ast.Node
	markEnd := 0 // end of the last link's markup, for an element closing after it
	linkU0 := map[ast.Node]int{}
	add := func(start, end int) {
		for i, r := range m.src[start:end] {
			if !unicode.IsSpace(r) {
				m.units = append(m.units, srcUnit{r, start + i, start + i + utf8.RuneLen(r), cur})
			}
		}
	}
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			switch n.(type) {
			case *ast.Link, *ast.Image:
				m.addDestination(linkU0[n], &markEnd, add)
			}
			if n == curNode {
				m.closeElem(cur, curU0, n, markEnd)
				cur, curNode = -1, nil
			}
			return ast.WalkContinue, nil
		}
		if cur < 0 && opensElem(n) {
			cur, curU0, curNode = len(m.elems), len(m.units), n
			m.elems = append(m.elems, srcElem{})
		}
		switch n := n.(type) {
		case *ast.Text:
			add(n.Segment.Start, n.Segment.Stop)
		case *ast.AutoLink:
			if l := n.Label(b); len(l) > 0 {
				start := cap(b) - cap(l) // l is a slice of b
				add(start, start+len(l))
			}
		case *ast.Link, *ast.Image:
			linkU0[n] = len(m.units)
		case *ast.FencedCodeBlock, *ast.CodeBlock, *ast.HTMLBlock:
			lines := n.Lines()
			u0 := len(m.units)
			for i := 0; i < lines.Len(); i++ {
				add(lines.At(i).Start, lines.At(i).Stop)
			}
			if _, fenced := n.(*ast.FencedCodeBlock); fenced && lines.Len() > 0 && len(m.units) > u0 {
				m.fences = append(m.fences, m.fence(u0, lines.At(0).Start, lines.At(lines.Len()-1).Stop))
			}
		}
		return ast.WalkContinue, nil
	})
}

// opensElem reports whether n is an element a selection must not cut.
func opensElem(n ast.Node) bool {
	switch n.(type) {
	case *ast.Emphasis, *ast.CodeSpan, *ast.Link, *ast.Image, *ast.AutoLink, *east.Strikethrough, *east.Table, *ast.HTMLBlock:
		return true
	case *ast.Paragraph, *ast.Heading, *ast.TextBlock:
		return hasRawHTML(n)
	}
	return false
}

func hasRawHTML(n ast.Node) bool {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if _, ok := c.(*ast.RawHTML); ok || hasRawHTML(c) {
			return true
		}
	}
	return false
}

// addDestination adds the destination of the link or image whose text
// starts at unit u0 as visible characters, since glamour shows it, and
// records where the link's markup ends.
func (m *sourceMap) addDestination(u0 int, markEnd *int, add func(start, end int)) {
	s := m.src
	i := 0 // where to look for the closing ']'
	if len(m.units) > 0 {
		i = m.units[len(m.units)-1].end
	}
	if len(m.units) == u0 && u0 > 0 {
		i = m.units[u0-1].end // no text: look after the previous one
	}
	j := strings.IndexByte(s[i:], ']')
	if j < 0 {
		return
	}
	i += j + 1
	switch {
	case i < len(s) && s[i] == '(':
		d := i + 1
		for d < len(s) && s[d] == ' ' {
			d++
		}
		e := d
		if d < len(s) && s[d] == '<' {
			d++
			e = d + max(strings.IndexByte(s[d:], '>'), 0)
		} else {
			for depth := 0; e < len(s) && s[e] != ' ' && s[e] != '\n' && (s[e] != ')' || depth > 0); e++ {
				switch s[e] {
				case '(':
					depth++
				case ')':
					depth--
				}
			}
		}
		add(d, e)
		if k := strings.IndexByte(s[e:], ')'); k >= 0 {
			*markEnd = e + k + 1
		}
	case i < len(s) && s[i] == '[':
		if k := strings.IndexByte(s[i:], ']'); k >= 0 {
			*markEnd = i + k + 1
		}
	default:
		*markEnd = i
	}
}

// closeElem sets the source range of element e, which holds the units from
// u0 on and ends no earlier than markEnd when that falls inside it.
func (m *sourceMap) closeElem(e, u0 int, n ast.Node, markEnd int) {
	if u0 == len(m.units) {
		m.elems[e].start, m.elems[e].end = -1, -1 // nothing visible
		return
	}
	s := m.src
	start, end := m.units[u0].start, m.units[len(m.units)-1].end
	switch n := n.(type) {
	case *east.Table, *ast.HTMLBlock:
		m.elems[e].start, m.elems[e].end = lineStart(s, start), lineEnd(s, end)
		return
	case *ast.Paragraph, *ast.Heading, *ast.TextBlock:
		lines := n.Lines()
		m.elems[e].start = lineStart(s, min(start, lines.At(0).Start))
		m.elems[e].end = lineEnd(s, max(end-1, lines.At(lines.Len()-1).Start))
		return
	}
	for start > 0 && strings.IndexByte(openMarks, s[start-1]) >= 0 ||
		start > 1 && s[start-1] == ' ' && s[start-2] == '`' {
		start--
	}
	if markEnd > end && markEnd <= len(s) {
		end = markEnd
	}
	for end < len(s) && strings.IndexByte(closeMarks, s[end]) >= 0 ||
		end+1 < len(s) && s[end] == ' ' && s[end+1] == '`' {
		end++
	}
	m.elems[e].start, m.elems[e].end = start, end
}

// fence returns the fenced code block whose code starts with unit u0 and
// spans bytes [codeStart, codeEnd), with its fence lines.
func (m *sourceMap) fence(u0, codeStart, codeEnd int) srcFence {
	s := m.src
	f := srcFence{u0: u0, u1: len(m.units) - 1, start: codeStart, end: codeEnd}
	if codeStart > 0 {
		f.start = lineStart(s, codeStart-1)
	}
	if close := strings.TrimSpace(s[codeEnd:lineEnd(s, codeEnd)]); strings.HasPrefix(close, "```") || strings.HasPrefix(close, "~~~") {
		f.end = lineEnd(s, codeEnd)
	} else if codeEnd > 0 && s[codeEnd-1] == '\n' {
		f.end = codeEnd - 1 // unclosed: the block ends with the document
	}
	return f
}

// align matches the rendered characters to the units, in order.
func (m *sourceMap) align(rendered []string) {
	type rc struct {
		r   rune
		pos textPos
		w   int
	}
	var cells []rc
	m.lineFirst = make([]int, len(rendered))
	m.lineLast = make([]int, len(rendered))
	for i, l := range rendered {
		m.lineFirst[i], m.lineLast[i] = -1, -1
		s := ansi.Strip(l)
		if strings.ContainsRune(s, kitty.Placeholder) {
			continue
		}
		col := 0
		for _, r := range s {
			w := ansi.StringWidth(string(r))
			if !unicode.IsSpace(r) && w > 0 {
				cells = append(cells, rc{r, textPos{i, col}, w})
				if m.lineFirst[i] < 0 {
					m.lineFirst[i] = col
				}
				m.lineLast[i] = col + w - 1
			}
			col += w
		}
	}

	// In sync, a character matches the next unit, or one a few units ahead
	// when the two characters after it match too ("&amp;" and "\*" render
	// shorter). An unmatched letter or digit is text the source map lacks, so
	// the alignment is lost until a run of characters matches again; other
	// unmatched characters are decorations and keep it.
	const lookahead, confirm, anchor, window = 8, 2, 6, 4000
	u := m.units
	runMatches := func(q, ci, n int) bool {
		for j := range n {
			if ci+j >= len(cells) || q+j >= len(u) || u[q+j].r != cells[ci+j].r {
				return ci+j >= len(cells) && j > 0
			}
		}
		return true
	}
	p, inSync := 0, true
	for ci, c := range cells {
		q := -1
		switch {
		case p >= len(u):
		case inSync && u[p].r == c.r || !inSync && runMatches(p, ci, confirm+1):
			q = p
		case inSync:
			for k := 1; k <= lookahead && p+k < len(u); k++ {
				if runMatches(p+k, ci, confirm+1) {
					q = p + k
					break
				}
			}
		default:
			for k := 1; p+k < len(u) && k < window; k++ {
				// A far jump needs a longer run: text the source map lacks
				// may well appear again later.
				if runMatches(p+k, ci, anchor+min(k/64, anchor)) {
					q = p + k
					break
				}
			}
		}
		if q < 0 {
			if unicode.IsLetter(c.r) || unicode.IsDigit(c.r) {
				inSync = false
			}
			continue
		}
		m.cells = append(m.cells, srcCell{c.pos, c.w, q})
		p, inSync = q+1, true
	}

	for i := range m.elems {
		m.elems[i].c0, m.elems[i].c1 = len(m.cells), -1
	}
	for ci, c := range m.cells {
		if e := u[c.unit].elem; e >= 0 && m.elems[e].start >= 0 {
			m.elems[e].c0 = min(m.elems[e].c0, ci)
			m.elems[e].c1 = max(m.elems[e].c1, ci)
		}
	}
}

func before(a, b textPos) bool { return a.line < b.line || a.line == b.line && a.col < b.col }

// snap grows the selection from..to to whole elements. It returns the new
// ends and the first and last matched cell inside, or ok false when the
// selection covers no source text.
func (m *sourceMap) snap(from, to textPos) (f, t textPos, c0, c1 int, ok bool) {
	c0 = sort.Search(len(m.cells), func(i int) bool {
		p := m.cells[i].pos
		return p.line > from.line || p.line == from.line && p.col+m.cells[i].w-1 >= from.col
	})
	c1 = sort.Search(len(m.cells), func(i int) bool { return before(to, m.cells[i].pos) }) - 1
	if c0 > c1 {
		return from, to, 0, 0, false
	}
	if e := m.elemOf(c0); e >= 0 {
		c0 = min(c0, m.elems[e].c0)
	}
	if e := m.elemOf(c1); e >= 0 {
		c1 = max(c1, m.elems[e].c1)
	}
	if p := m.cells[c0].pos; before(p, from) {
		from = p
	}
	if p := m.cells[c1].pos; before(to, textPos{p.line, p.col + m.cells[c1].w - 1}) {
		to = textPos{p.line, p.col + m.cells[c1].w - 1}
	}
	return from, to, c0, c1, true
}

func (m *sourceMap) elemOf(c int) int {
	if e := m.units[m.cells[c].unit].elem; e >= 0 && m.elems[e].start >= 0 {
		return e
	}
	return -1
}

// markdown returns the source of a snapped selection from..to with matched
// cells c0..c1. A selection that starts before a line's first character
// takes the source line's prefix too ("- ", "> ", "## "), one that ends
// after its last character takes the rest of the source line, and one that
// covers all of a fenced code block takes its fences.
func (m *sourceMap) markdown(from, to textPos, c0, c1 int) string {
	s := m.src
	u0, u1 := m.cells[c0].unit, m.cells[c1].unit
	start, end := m.units[u0].start, m.units[u1].end
	if e := m.elemOf(c0); e >= 0 {
		start = min(start, m.elems[e].start)
	}
	if e := m.elemOf(c1); e >= 0 {
		end = max(end, m.elems[e].end)
	}
	if p := m.cells[c0].pos; (from.line < p.line || from.col <= m.lineFirst[p.line]) && !m.unitIn(lineStart(s, start), start) {
		start = lineStart(s, start)
	}
	if p := m.cells[c1].pos; (to.line > p.line || to.col >= m.lineLast[p.line]) && !m.unitIn(end, lineEnd(s, end)) {
		end = lineEnd(s, end)
	}
	for _, f := range m.fences {
		if u0 <= f.u0 && u1 >= f.u1 {
			start, end = min(start, f.start), max(end, f.end)
		}
	}
	return s[start:end]
}

// unitIn reports whether a visible character starts in bytes [a, b).
func (m *sourceMap) unitIn(a, b int) bool {
	i := sort.Search(len(m.units), func(i int) bool { return m.units[i].start >= a })
	return i < len(m.units) && m.units[i].start < b
}

func lineStart(s string, pos int) int { return strings.LastIndexByte(s[:pos], '\n') + 1 }

func lineEnd(s string, pos int) int {
	if i := strings.IndexByte(s[pos:], '\n'); i >= 0 {
		return pos + i
	}
	return len(s)
}
