package query_test

import (
	"testing"

	"github.com/grota/gomd/internal/parser"
	"github.com/grota/gomd/internal/query"
)

func eval(t *testing.T, md, q string) []query.Value {
	t.Helper()
	doc := parser.ParseMarkdown(md)
	results, err := query.Execute(doc, q)
	if err != nil {
		t.Fatalf("query %q failed: %v", q, err)
	}
	return results
}

func TestIdentity(t *testing.T) {
	results := eval(t, "# Hello", ".")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Kind != query.ValDocument {
		t.Errorf("expected Document, got %s", results[0].Kind)
	}
}

func TestHeadingSelection(t *testing.T) {
	results := eval(t, "# H1\n## H2\n### H3", ".h2")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Kind != query.ValHeading {
		t.Fatalf("expected Heading, got %s", results[0].Kind)
	}
	if results[0].Heading.Text != "H2" {
		t.Errorf("expected 'H2', got %q", results[0].Heading.Text)
	}
	if results[0].Heading.Level != 2 {
		t.Errorf("expected level 2, got %d", results[0].Heading.Level)
	}
}

func TestAllHeadings(t *testing.T) {
	results := eval(t, "# H1\n## H2\n### H3", ".h")
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
}

func TestHeadingIndex(t *testing.T) {
	results := eval(t, "# H1\n## H2a\n## H2b", ".h2[0]")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Heading.Text != "H2a" {
		t.Errorf("expected 'H2a', got %q", results[0].Heading.Text)
	}
}

func TestHeadingFilter(t *testing.T) {
	results := eval(t, "# Hello\n## World\n## Goodbye", ".h2[World]")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Heading.Text != "World" {
		t.Errorf("expected 'World', got %q", results[0].Heading.Text)
	}
}

func TestCodeBlocks(t *testing.T) {
	md := "## Section\n\n```bash\necho hello\n```\n\n```go\nfmt.Println()\n```"
	results := eval(t, md, ".code")
	if len(results) != 2 {
		t.Fatalf("expected 2 code blocks, got %d", len(results))
	}
	if results[0].Code.Language != "bash" {
		t.Errorf("expected language 'bash', got %q", results[0].Code.Language)
	}
}

func TestPipeProperty(t *testing.T) {
	results := eval(t, "# Hello\n## World", ".h2 | text")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ToText() != "World" {
		t.Errorf("expected 'World', got %q", results[0].ToText())
	}
}

func TestCountFunction(t *testing.T) {
	results := eval(t, "# H1\n## H2a\n## H2b\n## H2c", "[.h2] | count")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Number != 3 {
		t.Errorf("expected 3, got %g", results[0].Number)
	}
}

func TestBinaryOp(t *testing.T) {
	results := eval(t, "# Hello", "1 + 2")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Number != 3 {
		t.Errorf("expected 3, got %g", results[0].Number)
	}
}

func TestContainsFilter(t *testing.T) {
	results := eval(t, "# Hello\n## API Reference\n## Installation", ".h | select(contains(\"API\"))")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Heading.Text != "API Reference" {
		t.Errorf("expected 'API Reference', got %q", results[0].Heading.Text)
	}
}
