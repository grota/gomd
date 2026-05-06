package parser

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// CodeBlock represents a fenced code block in a markdown document.
type CodeBlock struct {
	Language  string
	Content   string
	StartLine int
	EndLine   int
}

// Link represents a hyperlink in a markdown document.
type Link struct {
	Text   string
	URL    string
	Type   LinkType
	Offset int
}

// LinkType describes the kind of link.
type LinkType int

const (
	LinkTypeExternal LinkType = iota
	LinkTypeAnchor
	LinkTypeRelative
	LinkTypeWikiLink
)

func (t LinkType) String() string {
	switch t {
	case LinkTypeExternal:
		return "external"
	case LinkTypeAnchor:
		return "anchor"
	case LinkTypeRelative:
		return "relative"
	case LinkTypeWikiLink:
		return "wikilink"
	default:
		return "unknown"
	}
}

// Image represents an image reference in a markdown document.
type Image struct {
	Alt   string
	Src   string
	Title string
}

// Table represents a markdown table.
type Table struct {
	Headers    []string
	Rows       [][]string
	Alignments []string
}

// ListItem represents a single item in a list.
type ListItem struct {
	Content string
	Checked *bool // nil for non-checkbox items
}

// List represents a markdown list.
type List struct {
	Ordered bool
	Items   []ListItem
}

// newGoldmarkParser creates a goldmark instance with table extension.
func newGoldmarkParser() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(extension.Table),
	)
}

// ExtractLinks extracts all links from markdown content.
func ExtractLinks(content string) []Link {
	source := []byte(content)
	md := newGoldmarkParser()
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	var links []Link
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if link, ok := n.(*ast.Link); ok {
			url := string(link.Destination)
			linkText := extractPlainText(link, source)
			offset := 0
			// Find the offset of the link in source by looking at first child
			if link.FirstChild() != nil {
				if t, ok := link.FirstChild().(*ast.Text); ok {
					// The `[` is one byte before the text segment start
					offset = t.Segment.Start - 1
				}
			}
			links = append(links, Link{
				Text:   linkText,
				URL:    url,
				Type:   classifyURL(url),
				Offset: offset,
			})
		}
		return ast.WalkContinue, nil
	})

	// WikiLinks are not supported natively by goldmark, use regex fallback
	links = append(links, extractWikiLinks(content)...)

	return links
}

// extractWikiLinks extracts [[target]] or [[target|alias]] links.
func extractWikiLinks(content string) []Link {
	var links []Link
	i := 0
	for i < len(content)-3 {
		if content[i] == '[' && content[i+1] == '[' {
			// Find closing ]]
			end := strings.Index(content[i+2:], "]]")
			if end < 0 {
				i++
				continue
			}
			inner := content[i+2 : i+2+end]
			parts := strings.SplitN(inner, "|", 2)
			target := parts[0]
			linkText := target
			if len(parts) == 2 {
				linkText = parts[1]
			}
			links = append(links, Link{
				Text:   linkText,
				URL:    target,
				Type:   LinkTypeWikiLink,
				Offset: i,
			})
			i = i + 2 + end + 2
		} else {
			i++
		}
	}
	return links
}

func classifyURL(url string) LinkType {
	if strings.HasPrefix(url, "#") {
		return LinkTypeAnchor
	}
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "ftp://") || strings.HasPrefix(url, "mailto:") {
		return LinkTypeExternal
	}
	return LinkTypeRelative
}

// extractPlainText extracts plain text from an inline node.
func extractPlainText(n ast.Node, source []byte) string {
	var sb strings.Builder
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if t, ok := child.(*ast.Text); ok {
			sb.Write(t.Segment.Value(source))
		} else {
			sb.WriteString(extractPlainText(child, source))
		}
	}
	return sb.String()
}

// ExtractImages extracts all images from markdown content.
func ExtractImages(content string) []Image {
	source := []byte(content)
	md := newGoldmarkParser()
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	var images []Image
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if img, ok := n.(*ast.Image); ok {
			images = append(images, Image{
				Alt:   extractPlainText(img, source),
				Src:   string(img.Destination),
				Title: string(img.Title),
			})
		}
		return ast.WalkContinue, nil
	})
	return images
}

// ExtractCodeBlocks extracts fenced code blocks from markdown.
func ExtractCodeBlocks(content string) []CodeBlock {
	source := []byte(content)
	md := newGoldmarkParser()
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	lineOffsets := buildLineOffsets(source)

	var blocks []CodeBlock
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if fb, ok := n.(*ast.FencedCodeBlock); ok {
			lang := ""
			if fb.Info != nil {
				lang = strings.TrimSpace(string(fb.Info.Segment.Value(source)))
			}

			// Collect content lines
			var contentLines []string
			for i := 0; i < fb.Lines().Len(); i++ {
				seg := fb.Lines().At(i)
				line := string(seg.Value(source))
				// Remove trailing newline
				line = strings.TrimRight(line, "\n")
				contentLines = append(contentLines, line)
			}

			// Determine start/end lines
			// The fence opening is one line before the first content line
			startLine := 1
			if fb.Lines().Len() > 0 {
				firstSeg := fb.Lines().At(0)
				startLine = offsetToLine(lineOffsets, firstSeg.Start) - 1
			}
			// The fence closing is one line after the last content line
			endLine := startLine + len(contentLines) + 1
			if fb.Lines().Len() > 0 {
				lastSeg := fb.Lines().At(fb.Lines().Len() - 1)
				endLine = offsetToLine(lineOffsets, lastSeg.Stop)
			}

			blocks = append(blocks, CodeBlock{
				Language:  lang,
				Content:   strings.Join(contentLines, "\n"),
				StartLine: startLine,
				EndLine:   endLine,
			})
		}
		return ast.WalkContinue, nil
	})
	return blocks
}

// ExtractTables extracts markdown tables.
func ExtractTables(content string) []Table {
	source := []byte(content)
	md := newGoldmarkParser()
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	var tables []Table
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if tbl, ok := n.(*east.Table); ok {
			var headers []string
			var alignments []string
			var rows [][]string

			for child := tbl.FirstChild(); child != nil; child = child.NextSibling() {
				switch row := child.(type) {
				case *east.TableHeader:
					// Header row
					for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
						if tc, ok := cell.(*east.TableCell); ok {
							headers = append(headers, extractPlainText(tc, source))
							alignments = append(alignments, alignmentToString(tc.Alignment))
						}
					}
				case *east.TableRow:
					// Data row
					var rowCells []string
					for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
						if tc, ok := cell.(*east.TableCell); ok {
							rowCells = append(rowCells, extractPlainText(tc, source))
						}
					}
					rows = append(rows, rowCells)
				}
			}

			tables = append(tables, Table{
				Headers:    headers,
				Rows:       rows,
				Alignments: alignments,
			})
		}
		return ast.WalkContinue, nil
	})
	return tables
}

func alignmentToString(a east.Alignment) string {
	switch a {
	case east.AlignLeft:
		return "left"
	case east.AlignRight:
		return "right"
	case east.AlignCenter:
		return "center"
	default:
		return "left"
	}
}

// ExtractLists extracts markdown lists.
func ExtractLists(content string) []List {
	source := []byte(content)
	md := newGoldmarkParser()
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	var lists []List
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if list, ok := n.(*ast.List); ok {
			l := List{Ordered: list.IsOrdered()}

			for child := list.FirstChild(); child != nil; child = child.NextSibling() {
				if li, ok := child.(*ast.ListItem); ok {
					item := ListItem{}

					// Extract text content from list item
					var textContent string
					for liChild := li.FirstChild(); liChild != nil; liChild = liChild.NextSibling() {
						if para, ok := liChild.(*ast.TextBlock); ok {
							textContent = extractListItemContent(para, source)
						} else if para, ok := liChild.(*ast.Paragraph); ok {
							textContent = extractListItemContent(para, source)
						}
					}

					// Check for task list checkbox
					if tb, ok := child.(*ast.ListItem); ok && tb.HasChildren() {
						rawLine := getFirstLineRaw(tb, source)
						if strings.HasPrefix(rawLine, "[x] ") || strings.HasPrefix(rawLine, "[X] ") {
							checked := true
							item.Checked = &checked
							textContent = strings.TrimSpace(rawLine[4:])
						} else if strings.HasPrefix(rawLine, "[ ] ") {
							checked := false
							item.Checked = &checked
							textContent = strings.TrimSpace(rawLine[4:])
						}
					}

					item.Content = textContent
					l.Items = append(l.Items, item)
				}
			}

			lists = append(lists, l)
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	return lists
}

// extractListItemContent extracts text from a paragraph or text block node.
func extractListItemContent(n ast.Node, source []byte) string {
	var sb strings.Builder
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if t, ok := child.(*ast.Text); ok {
			sb.Write(t.Segment.Value(source))
			if t.SoftLineBreak() {
				sb.WriteByte(' ')
			}
		} else {
			sb.WriteString(extractPlainText(child, source))
		}
	}
	return sb.String()
}

// getFirstLineRaw gets the raw text of the first line in a list item's text content.
func getFirstLineRaw(li *ast.ListItem, source []byte) string {
	for child := li.FirstChild(); child != nil; child = child.NextSibling() {
		switch p := child.(type) {
		case *ast.TextBlock:
			if p.Lines().Len() > 0 {
				seg := p.Lines().At(0)
				return string(seg.Value(source))
			}
		case *ast.Paragraph:
			if p.Lines().Len() > 0 {
				seg := p.Lines().At(0)
				return string(seg.Value(source))
			}
		}
	}
	return ""
}
