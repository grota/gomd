package parser

import (
	"bufio"
	"regexp"
	"strings"
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

var (
	reLinkInline = regexp.MustCompile(`\[([^\]]*)\]\(([^)]*)\)`)
	reWikiLink   = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	reImage      = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]*?)(?:\s+"([^"]*)")?\)`)
	reTableRow   = regexp.MustCompile(`^\|(.+)\|$`)
	reTableSep   = regexp.MustCompile(`^\|[-| :]+\|$`)
	reCheckbox   = regexp.MustCompile(`^\[([xX ])\]\s+`)
)

// ExtractLinks extracts all links from markdown content.
func ExtractLinks(content string) []Link {
	var links []Link

	// Inline links [text](url)
	for _, m := range reLinkInline.FindAllStringSubmatchIndex(content, -1) {
		text := content[m[2]:m[3]]
		url := content[m[4]:m[5]]

		lt := classifyURL(url)
		links = append(links, Link{
			Text:   text,
			URL:    url,
			Type:   lt,
			Offset: m[0],
		})
	}

	// WikiLinks [[target]] or [[target|alias]]
	for _, m := range reWikiLink.FindAllStringSubmatchIndex(content, -1) {
		inner := content[m[2]:m[3]]
		parts := strings.SplitN(inner, "|", 2)
		target := parts[0]
		text := target
		if len(parts) == 2 {
			text = parts[1]
		}
		links = append(links, Link{
			Text:   text,
			URL:    target,
			Type:   LinkTypeWikiLink,
			Offset: m[0],
		})
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

// ExtractImages extracts all images from markdown content.
func ExtractImages(content string) []Image {
	var images []Image
	for _, m := range reImage.FindAllStringSubmatch(content, -1) {
		img := Image{
			Alt: m[1],
			Src: m[2],
		}
		if len(m) > 3 {
			img.Title = m[3]
		}
		images = append(images, img)
	}
	return images
}

// ExtractCodeBlocks extracts fenced code blocks from markdown.
func ExtractCodeBlocks(content string) []CodeBlock {
	var blocks []CodeBlock
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0
	inBlock := false
	var fence, lang string
	var blockLines []string
	var startLine int

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++
		trimmed := strings.TrimLeft(line, " \t")

		if !inBlock {
			if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
				inBlock = true
				fence = trimmed[:3]
				lang = strings.TrimSpace(trimmed[3:])
				startLine = lineNum
				blockLines = nil
			}
		} else {
			if strings.HasPrefix(trimmed, fence) {
				blocks = append(blocks, CodeBlock{
					Language:  lang,
					Content:   strings.Join(blockLines, "\n"),
					StartLine: startLine,
					EndLine:   lineNum,
				})
				inBlock = false
				fence = ""
				lang = ""
				blockLines = nil
			} else {
				blockLines = append(blockLines, line)
			}
		}
	}

	return blocks
}

// ExtractTables extracts markdown tables.
func ExtractTables(content string) []Table {
	var tables []Table
	lines := strings.Split(content, "\n")
	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if reTableRow.MatchString(line) && i+1 < len(lines) && reTableSep.MatchString(strings.TrimSpace(lines[i+1])) {
			// Parse header row
			headers := parseTableRow(line)
			// Parse separator for alignments
			alignments := parseTableAlignments(strings.TrimSpace(lines[i+1]))
			i += 2

			// Parse data rows
			var rows [][]string
			for i < len(lines) {
				rowLine := strings.TrimSpace(lines[i])
				if !reTableRow.MatchString(rowLine) {
					break
				}
				rows = append(rows, parseTableRow(rowLine))
				i++
			}

			tables = append(tables, Table{
				Headers:    headers,
				Rows:       rows,
				Alignments: alignments,
			})
		} else {
			i++
		}
	}
	return tables
}

func parseTableRow(line string) []string {
	// Remove leading/trailing pipes
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	cells := strings.Split(line, "|")
	for i, c := range cells {
		cells[i] = strings.TrimSpace(c)
	}
	return cells
}

func parseTableAlignments(line string) []string {
	cells := parseTableRow(line)
	alignments := make([]string, len(cells))
	for i, c := range cells {
		c = strings.TrimSpace(c)
		left := strings.HasPrefix(c, ":")
		right := strings.HasSuffix(c, ":")
		if left && right {
			alignments[i] = "center"
		} else if right {
			alignments[i] = "right"
		} else {
			alignments[i] = "left"
		}
	}
	return alignments
}

// ExtractLists extracts markdown lists.
func ExtractLists(content string) []List {
	var lists []List
	lines := strings.Split(content, "\n")
	i := 0
	for i < len(lines) {
		line := lines[i]
		stripped := strings.TrimSpace(line)
		if isListItem(stripped) {
			ordered := isOrderedListItem(stripped)
			var items []ListItem
			for i < len(lines) {
				l := strings.TrimSpace(lines[i])
				if !isListItem(l) {
					break
				}
				itemText := extractListItemText(l)
				item := ListItem{Content: itemText}
				if m := reCheckbox.FindStringIndex(itemText); m != nil {
					checked := strings.ToLower(itemText[1:2]) == "x"
					item.Checked = &checked
					item.Content = strings.TrimSpace(itemText[m[1]:])
				}
				items = append(items, item)
				i++
			}
			lists = append(lists, List{Ordered: ordered, Items: items})
		} else {
			i++
		}
	}
	return lists
}

func isListItem(line string) bool {
	return isUnorderedListItem(line) || isOrderedListItem(line)
}

func isUnorderedListItem(line string) bool {
	return len(line) > 1 && (line[0] == '-' || line[0] == '*' || line[0] == '+') && line[1] == ' '
}

func isOrderedListItem(line string) bool {
	i := 0
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	return i > 0 && i < len(line) && line[i] == '.' && i+1 < len(line) && line[i+1] == ' '
}

func extractListItemText(line string) string {
	if isUnorderedListItem(line) {
		return strings.TrimSpace(line[2:])
	}
	// ordered: find ". "
	i := 0
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	return strings.TrimSpace(line[i+2:])
}
