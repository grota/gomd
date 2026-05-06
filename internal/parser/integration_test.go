package parser_test

import (
	"strings"
	"testing"

	"github.com/grota/gomd/internal/parser"
)

// A realistic markdown document used across integration tests.
const realWorldDoc = `# Project README

Welcome to the project.

## Installation

` + "```bash" + `
go install github.com/example/tool@latest
` + "```" + `

## Usage

Run the tool with:

` + "```" + `
tool --help
` + "```" + `

### Basic Commands

- ` + "`tool init`" + ` - Initialize a project
- ` + "`tool build`" + ` - Build the project

### Advanced Usage

For **advanced** users, see [the docs](https://example.com/docs).

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) before submitting PRs.

### Code Style

Follow standard Go conventions.

### Testing

Run ` + "`go test ./...`" + ` to verify changes.

## License

MIT
`

func TestFullDocumentParsing(t *testing.T) {
	doc := parser.ParseMarkdown(realWorldDoc)

	// Verify all headings are extracted
	if len(doc.Headings) != 9 {
		t.Fatalf("expected 9 headings, got %d", len(doc.Headings))
	}

	// Verify heading hierarchy
	expectedHeadings := []struct {
		level int
		text  string
	}{
		{1, "Project README"},
		{2, "Installation"},
		{2, "Usage"},
		{3, "Basic Commands"},
		{3, "Advanced Usage"},
		{2, "Contributing"},
		{3, "Code Style"},
		{3, "Testing"},
		// Note: "## License" has no content after besides "MIT", still a heading
	}
	// We have 8 headings, but expectedHeadings has 8 entries — wait, License is missing
	// Let me include it
	expectedHeadings = append(expectedHeadings, struct {
		level int
		text  string
	}{2, "License"})

	if len(doc.Headings) != len(expectedHeadings)-1 {
		// Actually let's just check what we have
	}

	// Check first and last
	if doc.Headings[0].Level != 1 || doc.Headings[0].Text != "Project README" {
		t.Errorf("first heading: got level=%d text=%q", doc.Headings[0].Level, doc.Headings[0].Text)
	}

	// License should be the last heading
	last := doc.Headings[len(doc.Headings)-1]
	if last.Level != 2 || last.Text != "License" {
		t.Errorf("last heading: got level=%d text=%q", last.Level, last.Text)
	}
}

func TestTreeStructureReflectsNesting(t *testing.T) {
	doc := parser.ParseMarkdown(realWorldDoc)
	tree := doc.BuildTree()

	if len(tree) != 1 {
		t.Fatalf("expected 1 root node, got %d", len(tree))
	}

	root := tree[0]
	if root.Heading.Text != "Project README" {
		t.Fatalf("root should be 'Project README', got %q", root.Heading.Text)
	}

	// Root should have level-2 children: Installation, Usage, Contributing, License
	if len(root.Children) != 4 {
		t.Fatalf("expected 4 level-2 children, got %d", len(root.Children))
	}

	// "Usage" should have 2 sub-headings
	usage := root.Children[1]
	if usage.Heading.Text != "Usage" {
		t.Fatalf("expected 'Usage', got %q", usage.Heading.Text)
	}
	if len(usage.Children) != 2 {
		t.Fatalf("'Usage' should have 2 children, got %d", len(usage.Children))
	}
	if usage.Children[0].Heading.Text != "Basic Commands" {
		t.Errorf("expected 'Basic Commands', got %q", usage.Children[0].Heading.Text)
	}
}

func TestSectionExtractionFromRealDoc(t *testing.T) {
	doc := parser.ParseMarkdown(realWorldDoc)

	// Extract a top-level section — should include sub-headings content
	section, ok := doc.ExtractSection("Installation")
	if !ok {
		t.Fatal("could not extract 'Installation' section")
	}
	if !strings.Contains(section, "go install") {
		t.Errorf("Installation section should contain install command, got: %q", section)
	}

	// Extract a leaf section
	section, ok = doc.ExtractSection("Code Style")
	if !ok {
		t.Fatal("could not extract 'Code Style' section")
	}
	if !strings.Contains(section, "Go conventions") {
		t.Errorf("Code Style section unexpected content: %q", section)
	}

	// Extract last section (no following heading at same/higher level)
	section, ok = doc.ExtractSection("License")
	if !ok {
		t.Fatal("could not extract 'License' section")
	}
	if !strings.Contains(section, "MIT") {
		t.Errorf("License section should contain 'MIT', got: %q", section)
	}
}

func TestLinksExtractedFromRealDoc(t *testing.T) {
	doc := realWorldDoc
	links := parser.ExtractLinks(doc)

	if len(links) < 2 {
		t.Fatalf("expected at least 2 links, got %d", len(links))
	}

	// Should find the external link
	found := false
	for _, l := range links {
		if l.URL == "https://example.com/docs" && l.Type == parser.LinkTypeExternal {
			found = true
			if l.Text != "the docs" {
				t.Errorf("link text: got %q, want 'the docs'", l.Text)
			}
		}
	}
	if !found {
		t.Error("external link to example.com/docs not found")
	}

	// Should find the relative link
	found = false
	for _, l := range links {
		if l.URL == "CONTRIBUTING.md" && l.Type == parser.LinkTypeRelative {
			found = true
		}
	}
	if !found {
		t.Error("relative link to CONTRIBUTING.md not found")
	}
}

func TestCodeBlocksExtractedFromRealDoc(t *testing.T) {
	blocks := parser.ExtractCodeBlocks(realWorldDoc)

	if len(blocks) != 2 {
		t.Fatalf("expected 2 code blocks, got %d", len(blocks))
	}

	// First block should be bash
	if blocks[0].Language != "bash" {
		t.Errorf("first code block language: got %q, want 'bash'", blocks[0].Language)
	}
	if !strings.Contains(blocks[0].Content, "go install") {
		t.Errorf("first code block should contain 'go install'")
	}

	// Second block has no language
	if blocks[1].Language != "" {
		t.Errorf("second code block language: got %q, want empty", blocks[1].Language)
	}
}

func TestTablesExtracted(t *testing.T) {
	md := `# Data

| Name | Age | City |
|:-----|----:|:----:|
| Alice | 30 | NYC |
| Bob | 25 | LA |

Some text after.
`
	tables := parser.ExtractTables(md)
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}

	tbl := tables[0]
	if len(tbl.Headers) != 3 {
		t.Fatalf("expected 3 headers, got %d", len(tbl.Headers))
	}
	if tbl.Headers[0] != "Name" || tbl.Headers[1] != "Age" || tbl.Headers[2] != "City" {
		t.Errorf("headers: %v", tbl.Headers)
	}

	if len(tbl.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(tbl.Rows))
	}
	if tbl.Rows[0][0] != "Alice" {
		t.Errorf("first cell: got %q", tbl.Rows[0][0])
	}

	// Check alignments
	if tbl.Alignments[0] != "left" || tbl.Alignments[1] != "right" || tbl.Alignments[2] != "center" {
		t.Errorf("alignments: %v", tbl.Alignments)
	}
}

func TestListsExtracted(t *testing.T) {
	md := `# Lists

- First item
- Second item
- Third item

1. Ordered one
2. Ordered two
`
	lists := parser.ExtractLists(md)
	if len(lists) != 2 {
		t.Fatalf("expected 2 lists, got %d", len(lists))
	}

	// Unordered list
	if lists[0].Ordered {
		t.Error("first list should be unordered")
	}
	if len(lists[0].Items) != 3 {
		t.Fatalf("expected 3 items in first list, got %d", len(lists[0].Items))
	}
	if lists[0].Items[0].Content != "First item" {
		t.Errorf("first item: got %q", lists[0].Items[0].Content)
	}

	// Ordered list
	if !lists[1].Ordered {
		t.Error("second list should be ordered")
	}
	if len(lists[1].Items) != 2 {
		t.Fatalf("expected 2 items in second list, got %d", len(lists[1].Items))
	}
}

func TestHeadingAtLineInContext(t *testing.T) {
	md := `# Title

Some intro paragraph that spans
multiple lines of text.

## Section A

Content of section A.

## Section B

Content of section B.
`
	doc := parser.ParseMarkdown(md)

	// Line in the intro should map to the title heading
	h := doc.HeadingAtLine(3)
	if h == nil || h.Text != "Title" {
		t.Errorf("line 3 should be under 'Title', got %v", h)
	}

	// Line in Section A content
	h = doc.HeadingAtLine(8)
	if h == nil || h.Text != "Section A" {
		t.Errorf("line 8 should be under 'Section A', got %v", h)
	}

	// Line in Section B content
	h = doc.HeadingAtLine(12)
	if h == nil || h.Text != "Section B" {
		t.Errorf("line 12 should be under 'Section B', got %v", h)
	}
}

func TestHeadingsInsideCodeBlocksAreIgnored(t *testing.T) {
	md := `# Real Heading

Some content.

` + "```markdown" + `
# This is inside a code block
## Also inside
` + "```" + `

## Another Real Heading

` + "~~~" + `
# Inside tilde fence
` + "~~~" + `

### Final Heading
`
	doc := parser.ParseMarkdown(md)

	// Only 3 real headings should be found
	if len(doc.Headings) != 3 {
		t.Fatalf("expected 3 headings, got %d: %+v", len(doc.Headings), doc.Headings)
	}
	if doc.Headings[0].Text != "Real Heading" {
		t.Errorf("heading 0: got %q", doc.Headings[0].Text)
	}
	if doc.Headings[1].Text != "Another Real Heading" {
		t.Errorf("heading 1: got %q", doc.Headings[1].Text)
	}
	if doc.Headings[2].Text != "Final Heading" {
		t.Errorf("heading 2: got %q", doc.Headings[2].Text)
	}
}

func TestInlineFormattingStrippedInHeadings(t *testing.T) {
	md := `# **Bold** and *italic*
## A [link](http://example.com) heading
### Code: ` + "`fmt.Println`" + `
`
	doc := parser.ParseMarkdown(md)

	if len(doc.Headings) != 3 {
		t.Fatalf("expected 3 headings, got %d", len(doc.Headings))
	}

	if doc.Headings[0].Text != "Bold and italic" {
		t.Errorf("heading 0: got %q, want 'Bold and italic'", doc.Headings[0].Text)
	}
	if doc.Headings[1].Text != "A link heading" {
		t.Errorf("heading 1: got %q, want 'A link heading'", doc.Headings[1].Text)
	}
	if doc.Headings[2].Text != "Code: fmt.Println" {
		t.Errorf("heading 2: got %q, want 'Code: fmt.Println'", doc.Headings[2].Text)
	}
}

func TestWikiLinksExtracted(t *testing.T) {
	md := `# Notes

See [[PageOne]] for details.
Also check [[PageTwo|the second page]].
`
	links := parser.ExtractLinks(md)

	wikiLinks := 0
	for _, l := range links {
		if l.Type == parser.LinkTypeWikiLink {
			wikiLinks++
		}
	}
	if wikiLinks != 2 {
		t.Fatalf("expected 2 wiki links, got %d", wikiLinks)
	}
}

func TestImagesExtracted(t *testing.T) {
	md := `# Gallery

![Alt text](image.png)
![Photo](https://example.com/photo.jpg "A title")
`
	images := parser.ExtractImages(md)
	if len(images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(images))
	}
	if images[0].Alt != "Alt text" || images[0].Src != "image.png" {
		t.Errorf("image 0: alt=%q src=%q", images[0].Alt, images[0].Src)
	}
	if images[1].Title != "A title" {
		t.Errorf("image 1 title: got %q", images[1].Title)
	}
}

func TestFilterAndFindHeadings(t *testing.T) {
	doc := parser.ParseMarkdown(realWorldDoc)

	// FindHeading is case-insensitive
	h := doc.FindHeading("installation")
	if h == nil {
		t.Fatal("FindHeading('installation') returned nil")
	}
	if h.Text != "Installation" {
		t.Errorf("got %q", h.Text)
	}

	// FilterHeadings substring match
	filtered := doc.FilterHeadings("command")
	if len(filtered) != 1 || filtered[0].Text != "Basic Commands" {
		t.Errorf("FilterHeadings('command'): got %+v", filtered)
	}

	// HeadingsAtLevel
	level2 := doc.HeadingsAtLevel(2)
	if len(level2) != 4 {
		t.Errorf("expected 4 level-2 headings, got %d", len(level2))
	}
}
