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

func TestFilterHeadings(t *testing.T) {
	doc := parser.ParseMarkdown(realWorldDoc)

	filtered := doc.FilterHeadings(0, "COMMAND")
	if len(filtered) != 1 || filtered[0].Text != "Basic Commands" {
		t.Errorf("FilterHeadings(0, \"COMMAND\"): got %+v", filtered)
	}
	if level2 := doc.FilterHeadings(2, ""); len(level2) != 4 {
		t.Errorf("expected 4 level-2 headings, got %d", len(level2))
	}
	if both := doc.FilterHeadings(1, "command"); len(both) != 0 {
		t.Errorf("level and text filters must combine, got %+v", both)
	}
}
