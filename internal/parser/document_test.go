package parser_test

import (
	"testing"

	"github.com/grota/gomd/internal/parser"
)

func TestParseHeadings(t *testing.T) {
	md := "# H1\n## H2\n### H3"
	doc := parser.ParseMarkdown(md)

	if len(doc.Headings) != 3 {
		t.Fatalf("expected 3 headings, got %d", len(doc.Headings))
	}

	tests := []struct {
		level int
		text  string
	}{
		{1, "H1"},
		{2, "H2"},
		{3, "H3"},
	}
	for i, tt := range tests {
		h := doc.Headings[i]
		if h.Level != tt.level {
			t.Errorf("heading[%d] level: got %d, want %d", i, h.Level, tt.level)
		}
		if h.Text != tt.text {
			t.Errorf("heading[%d] text: got %q, want %q", i, h.Text, tt.text)
		}
	}
}

func TestParseHeadingsWithBold(t *testing.T) {
	md := "# **Bold Title**\n## `code heading`"
	doc := parser.ParseMarkdown(md)

	if len(doc.Headings) != 2 {
		t.Fatalf("expected 2 headings, got %d", len(doc.Headings))
	}
	if doc.Headings[0].Text != "Bold Title" {
		t.Errorf("expected 'Bold Title', got %q", doc.Headings[0].Text)
	}
	if doc.Headings[1].Text != "code heading" {
		t.Errorf("expected 'code heading', got %q", doc.Headings[1].Text)
	}
}

func TestHeadingsStoreOffsets(t *testing.T) {
	md := "# Hello\n\n## World"
	doc := parser.ParseMarkdown(md)

	if len(doc.Headings) != 2 {
		t.Fatalf("expected 2 headings, got %d", len(doc.Headings))
	}
	if doc.Headings[0].Offset != 0 {
		t.Errorf("first heading offset: got %d, want 0", doc.Headings[0].Offset)
	}
	// "# Hello\n\n" = 9 bytes
	if doc.Headings[1].Offset != 9 {
		t.Errorf("second heading offset: got %d, want 9", doc.Headings[1].Offset)
	}
}

func TestExtractSection(t *testing.T) {
	md := "# Introduction\n\nThis is the intro.\n\n## Details\n\nThis is details.\n\n# Conclusion\n\nThe end."
	doc := parser.ParseMarkdown(md)

	section, ok := doc.ExtractSection("Details")
	if !ok {
		t.Fatal("section 'Details' not found")
	}
	if section != "This is details." {
		t.Errorf("unexpected section content: %q", section)
	}
}

func TestExtractSectionAtEndOfDocument(t *testing.T) {
	md := "# First\n\ncontent\n\n# Last\n\nfinal content"
	doc := parser.ParseMarkdown(md)

	section, ok := doc.ExtractSection("Last")
	if !ok {
		t.Fatal("section 'Last' not found")
	}
	if section != "final content" {
		t.Errorf("unexpected section content: %q", section)
	}
}

func TestBuildTree(t *testing.T) {
	md := "# Root\n## Child1\n### Grandchild\n## Child2"
	doc := parser.ParseMarkdown(md)

	tree := doc.BuildTree()
	if len(tree) != 1 {
		t.Fatalf("expected 1 root, got %d", len(tree))
	}
	root := tree[0]
	if root.Heading.Text != "Root" {
		t.Errorf("root text: got %q, want 'Root'", root.Heading.Text)
	}
	if len(root.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(root.Children))
	}
	if len(root.Children[0].Children) != 1 {
		t.Fatalf("expected 1 grandchild, got %d", len(root.Children[0].Children))
	}
}

func TestIgnoreHeadingsInCodeBlock(t *testing.T) {
	md := "# Real heading\n\n```\n# Not a heading\n```\n\n## Also real"
	doc := parser.ParseMarkdown(md)

	if len(doc.Headings) != 2 {
		t.Fatalf("expected 2 headings (ignoring one in code block), got %d", len(doc.Headings))
	}
}

func TestHeadingAtLine(t *testing.T) {
	md := "# First\n\nsome text\n\n## Second\n\nmore text"
	doc := parser.ParseMarkdown(md)

	h := doc.HeadingAtLine(6)
	if h == nil {
		t.Fatal("expected a heading at line 6")
	}
	if h.Text != "Second" {
		t.Errorf("expected 'Second', got %q", h.Text)
	}
}
