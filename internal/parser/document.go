// Package parser provides markdown document parsing functionality.
// It extracts headings with byte offsets and builds hierarchical tree structures
// using goldmark for AST parsing.
package parser

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Heading represents a single heading in a markdown document.
type Heading struct {
	// Level is the heading level (1 for #, 2 for ##, etc.)
	Level int
	// Text is the heading text content (stripped of inline markdown formatting)
	Text string
	// Offset is the byte offset where the heading starts in the source document
	Offset int
	// Line is the 1-based line number of the heading
	Line int
}

// HeadingNode is a node in the heading tree.
type HeadingNode struct {
	Heading  Heading
	Children []*HeadingNode
}

// Document represents a parsed markdown document.
type Document struct {
	Content  string
	Headings []Heading
}

// NewDocument creates a new document from content and headings.
func NewDocument(content string, headings []Heading) *Document {
	return &Document{Content: content, Headings: headings}
}

// BuildTree builds a hierarchical tree from the flat heading list.
func (d *Document) BuildTree() []*HeadingNode {
	type stackEntry struct {
		level int
		node  *HeadingNode
	}

	var roots []*HeadingNode
	var stack []stackEntry

	for _, h := range d.Headings {
		node := &HeadingNode{Heading: h}

		// Pop stack until we find a parent (heading with level < current)
		for len(stack) > 0 && stack[len(stack)-1].level >= h.Level {
			stack = stack[:len(stack)-1]
		}

		if len(stack) > 0 {
			parent := stack[len(stack)-1].node
			parent.Children = append(parent.Children, node)
		} else {
			roots = append(roots, node)
		}

		stack = append(stack, stackEntry{level: h.Level, node: node})
	}

	return roots
}

// HeadingsAtLevel returns all headings at a specific level.
func (d *Document) HeadingsAtLevel(level int) []Heading {
	var result []Heading
	for _, h := range d.Headings {
		if h.Level == level {
			result = append(result, h)
		}
	}
	return result
}

// FindHeading finds a heading by text (case-insensitive).
func (d *Document) FindHeading(text string) *Heading {
	search := strings.ToLower(text)
	for i, h := range d.Headings {
		if strings.ToLower(h.Text) == search {
			return &d.Headings[i]
		}
	}
	return nil
}

// FilterHeadings returns all headings matching a filter (case-insensitive substring).
func (d *Document) FilterHeadings(filter string) []Heading {
	search := strings.ToLower(filter)
	var result []Heading
	for _, h := range d.Headings {
		if strings.Contains(strings.ToLower(h.Text), search) {
			result = append(result, h)
		}
	}
	return result
}

// ExtractSection extracts the content of a section by heading text.
// Uses stored byte offsets for fast, accurate extraction.
func (d *Document) ExtractSection(headingText string) (string, bool) {
	headingIdx := -1
	for i, h := range d.Headings {
		if strings.ToLower(h.Text) == strings.ToLower(headingText) {
			headingIdx = i
			break
		}
	}
	if headingIdx < 0 {
		return "", false
	}

	heading := d.Headings[headingIdx]
	start := heading.Offset

	// Skip the heading line itself
	afterHeading := d.Content[start:]
	contentStart := start
	if nl := strings.Index(afterHeading, "\n"); nl >= 0 {
		contentStart = start + nl + 1
	}

	// Find end: next heading at same or higher level
	end := len(d.Content)
	for i := headingIdx + 1; i < len(d.Headings); i++ {
		if d.Headings[i].Level <= heading.Level {
			end = d.Headings[i].Offset
			break
		}
	}

	section := strings.TrimSpace(d.Content[contentStart:end])
	return section, true
}

// HeadingAtLine returns the heading at or before the given line number.
func (d *Document) HeadingAtLine(line int) *Heading {
	var result *Heading
	for i, h := range d.Headings {
		if h.Line <= line {
			result = &d.Headings[i]
		} else {
			break
		}
	}
	return result
}

// RenderBoxTree renders a HeadingNode as a tree with box-drawing characters.
func (n *HeadingNode) RenderBoxTree(prefix string, isLast bool, compact bool) string {
	var sb strings.Builder

	var connector, continuation string
	if compact {
		if isLast {
			connector = "└──"
			continuation = "   "
		} else {
			connector = "├──"
			continuation = "│  "
		}
	} else {
		if isLast {
			connector = "└─ "
			continuation = "    "
		} else {
			connector = "├─ "
			continuation = "│   "
		}
	}

	marker := strings.Repeat("#", n.Heading.Level)
	sb.WriteString(prefix)
	sb.WriteString(connector)
	sb.WriteString(marker)
	sb.WriteString(" ")
	sb.WriteString(n.Heading.Text)
	sb.WriteString("\n")

	childPrefix := prefix + continuation
	for i, child := range n.Children {
		isLastChild := i == len(n.Children)-1
		sb.WriteString(child.RenderBoxTree(childPrefix, isLastChild, compact))
	}

	return sb.String()
}

// extractTextFromNode recursively extracts plain text from an AST node's children.
func extractTextFromNode(n ast.Node, source []byte) string {
	var sb strings.Builder
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		switch c := child.(type) {
		case *ast.Text:
			sb.Write(c.Segment.Value(source))
		case *ast.CodeSpan:
			// Extract text inside code span
			for gc := c.FirstChild(); gc != nil; gc = gc.NextSibling() {
				if t, ok := gc.(*ast.Text); ok {
					sb.Write(t.Segment.Value(source))
				}
			}
		case *ast.Link:
			// Extract link text
			sb.WriteString(extractTextFromNode(c, source))
		default:
			// Recurse for emphasis, strong, etc.
			sb.WriteString(extractTextFromNode(child, source))
		}
	}
	return sb.String()
}

// parseMarkdownHeadings extracts headings from markdown content using goldmark.
func parseMarkdownHeadings(content string) []Heading {
	source := []byte(content)
	md := goldmark.New()
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	// Build a line offset table to convert byte offsets to line numbers
	lineOffsets := buildLineOffsets(source)

	var headings []Heading
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if heading, ok := n.(*ast.Heading); ok {
			// Get byte offset of content, then find line start
			contentOffset := 0
			if heading.Lines().Len() > 0 {
				contentOffset = heading.Lines().At(0).Start
			}

			// Find the start of the line (the actual `#` characters)
			lineNum := offsetToLine(lineOffsets, contentOffset)
			lineStart := lineOffsets[lineNum-1] // lineNum is 1-based, lineOffsets is 0-based

			// Extract plain text from heading children
			headingText := extractTextFromNode(heading, source)

			headings = append(headings, Heading{
				Level:  heading.Level,
				Text:   headingText,
				Offset: lineStart,
				Line:   lineNum,
			})
		}
		return ast.WalkContinue, nil
	})

	return headings
}

// buildLineOffsets returns byte offsets where each line starts (0-indexed lines).
func buildLineOffsets(source []byte) []int {
	offsets := []int{0}
	for i, b := range source {
		if b == '\n' {
			offsets = append(offsets, i+1)
		}
	}
	return offsets
}

// offsetToLine converts a byte offset to a 1-based line number.
func offsetToLine(lineOffsets []int, offset int) int {
	// Binary search for the line containing this offset
	lo, hi := 0, len(lineOffsets)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		if lineOffsets[mid] <= offset {
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return lo // 1-based because lineOffsets[0] = 0 means line 1
}

// stripFrontmatter removes YAML frontmatter (delimited by "---") from the
// beginning of a markdown document. Returns the content without frontmatter.
func stripFrontmatter(content string) string {
	if !strings.HasPrefix(content, "---") {
		return content
	}
	// Find the closing "---" after the opening one.
	rest := content[3:]
	// Skip the remainder of the opening line (e.g. "---\n")
	nl := strings.Index(rest, "\n")
	if nl < 0 {
		return content // no newline after opening ---, not valid frontmatter
	}
	rest = rest[nl+1:]
	closing := strings.Index(rest, "---")
	if closing < 0 {
		return content // no closing ---
	}
	// Ensure the closing --- is at the start of a line
	if closing > 0 && rest[closing-1] != '\n' {
		return content
	}
	// Skip past the closing --- line
	after := rest[closing+3:]
	if idx := strings.Index(after, "\n"); idx >= 0 {
		after = after[idx+1:]
	} else {
		after = ""
	}
	return after
}

// ParseMarkdown parses markdown content and returns a Document.
func ParseMarkdown(content string) *Document {
	stripped := stripFrontmatter(content)
	headings := parseMarkdownHeadings(stripped)
	return NewDocument(stripped, headings)
}
