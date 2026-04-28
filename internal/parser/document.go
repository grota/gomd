// Package parser provides markdown document parsing functionality.
// It extracts headings with byte offsets and builds hierarchical tree structures.
package parser

import (
	"bufio"
	"strings"
	"unicode"
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

// parseMarkdownHeadings extracts headings from markdown content.
// It is code-block-aware (ignores headings inside fenced code blocks).
func parseMarkdownHeadings(content string) []Heading {
	var headings []Heading
	scanner := bufio.NewScanner(strings.NewReader(content))
	offset := 0
	lineNum := 0
	inCodeBlock := false
	var codeBlockFence string

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// Track fenced code blocks
		trimmed := strings.TrimLeft(line, " \t")
		if !inCodeBlock {
			if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
				inCodeBlock = true
				codeBlockFence = trimmed[:3]
			}
		} else {
			if strings.HasPrefix(trimmed, codeBlockFence) {
				inCodeBlock = false
				codeBlockFence = ""
			}
		}

		if !inCodeBlock && strings.HasPrefix(line, "#") {
			level, text := parseHeadingLine(line)
			if level > 0 {
				headings = append(headings, Heading{
					Level:  level,
					Text:   text,
					Offset: offset,
					Line:   lineNum,
				})
			}
		}

		offset += len(line) + 1 // +1 for newline
	}

	return headings
}

// parseHeadingLine parses a markdown heading line.
// Returns (level, text) or (0, "") if not a heading.
func parseHeadingLine(line string) (int, string) {
	level := 0
	for level < len(line) && line[level] == '#' {
		level++
	}
	if level == 0 || level > 6 {
		return 0, ""
	}
	// Must be followed by a space
	if level >= len(line) || line[level] != ' ' {
		return 0, ""
	}
	text := strings.TrimSpace(line[level+1:])
	text = stripInlineMarkdown(text)
	return level, text
}

// stripInlineMarkdown removes inline markdown formatting from text.
func stripInlineMarkdown(text string) string {
	var sb strings.Builder
	runes := []rune(text)
	i := 0
	for i < len(runes) {
		c := runes[i]
		switch c {
		case '*', '_':
			// Skip bold/italic markers
			j := i
			for j < len(runes) && runes[j] == c {
				j++
			}
			i = j
		case '`':
			// Skip code span
			j := i + 1
			for j < len(runes) && runes[j] != '`' {
				j++
			}
			if j < len(runes) {
				// Extract content inside backticks
				sb.WriteString(string(runes[i+1 : j]))
				i = j + 1
			} else {
				i++
			}
		case '[':
			// Link: [text](url) -> text
			j := i + 1
			for j < len(runes) && runes[j] != ']' {
				j++
			}
			if j < len(runes) && j+1 < len(runes) && runes[j+1] == '(' {
				// It's a link - extract the text
				sb.WriteString(string(runes[i+1 : j]))
				// Skip past the URL
				j += 2
				for j < len(runes) && runes[j] != ')' {
					j++
				}
				i = j + 1
			} else {
				sb.WriteRune(c)
				i++
			}
		case '\\':
			// Escape sequence
			if i+1 < len(runes) {
				sb.WriteRune(runes[i+1])
				i += 2
			} else {
				i++
			}
		default:
			if !unicode.IsControl(c) {
				sb.WriteRune(c)
			}
			i++
		}
	}
	return sb.String()
}

// ParseMarkdown parses markdown content and returns a Document.
func ParseMarkdown(content string) *Document {
	headings := parseMarkdownHeadings(content)
	return NewDocument(content, headings)
}
