package tui

import (
	"strings"
	"testing"

	"github.com/grota/gomd/internal/config"
	"github.com/grota/gomd/internal/parser"
)

const featureTestDoc = `# Introduction

Welcome to the project.

## Getting Started

Install with ` + "`go install`" + `.

### Prerequisites

You need Go 1.22+.

## API Reference

See [the docs](https://example.com/docs) and [#getting-started](#getting-started).

### Authentication

Use tokens.

## Contributing

PRs welcome.
`

func newTestApp(md string) *App {
	doc := parser.ParseMarkdown(md)
	cfg := config.Default()
	app := NewApp(doc, "test.md", "test.md", cfg)
	app.width = 120
	app.height = 40
	return app
}

func TestFuzzySearch(t *testing.T) {
	app := newTestApp(featureTestDoc)

	// "gst" should fuzzy-match "Getting Started"
	if !fuzzyMatch("getting started", "gst") {
		t.Error("expected fuzzyMatch('getting started', 'gst') to be true")
	}

	// "auth" should match "Authentication"
	if !fuzzyMatch("authentication", "auth") {
		t.Error("expected fuzzyMatch('authentication', 'auth') to be true")
	}

	// Non-match
	if fuzzyMatch("introduction", "xyz") {
		t.Error("expected fuzzyMatch('introduction', 'xyz') to be false")
	}

	// Test search integration — search for "prereq" should find Prerequisites
	app.mode = ModeSearch
	app.searchQuery = "prereq"
	// Simulate enter
	app.searchMatches = nil
	q := strings.ToLower(app.searchQuery)
	for i, h := range app.doc.Headings {
		if fuzzyMatch(strings.ToLower(h.Text), q) {
			app.searchMatches = append(app.searchMatches, i)
		}
	}
	if len(app.searchMatches) != 1 {
		t.Fatalf("expected 1 match for 'prereq', got %d", len(app.searchMatches))
	}
	if app.doc.Headings[app.searchMatches[0]].Text != "Prerequisites" {
		t.Errorf("expected 'Prerequisites', got %q", app.doc.Headings[app.searchMatches[0]].Text)
	}
}

func TestBreadcrumb(t *testing.T) {
	app := newTestApp(featureTestDoc)

	// Select "Prerequisites" (nested under Getting Started > Introduction...)
	// Find the index of Prerequisites
	for i, h := range app.doc.Headings {
		if h.Text == "Prerequisites" {
			app.selectedIdx = i
			break
		}
	}

	bc := app.breadcrumb()
	// Should show Getting Started > Prerequisites (since Getting Started is the parent h2)
	if !strings.Contains(bc, "Getting Started") {
		t.Errorf("breadcrumb should contain parent 'Getting Started', got %q", bc)
	}
	if !strings.Contains(bc, "Prerequisites") {
		t.Errorf("breadcrumb should contain 'Prerequisites', got %q", bc)
	}
}

func TestNavigationHistory(t *testing.T) {
	app := newTestApp(featureTestDoc)

	// Start at root (-1)
	if app.selectedIdx != -1 {
		t.Fatalf("expected starting at root, got %d", app.selectedIdx)
	}

	// Navigate to heading 0
	app.navigateTo(0)
	if app.selectedIdx != 0 {
		t.Fatalf("expected selectedIdx=0, got %d", app.selectedIdx)
	}

	// Navigate to heading 2
	app.navigateTo(2)
	if app.selectedIdx != 2 {
		t.Fatalf("expected selectedIdx=2, got %d", app.selectedIdx)
	}

	// Go back
	app.navGoBack()
	if app.selectedIdx != 0 {
		t.Errorf("after back: expected 0, got %d", app.selectedIdx)
	}

	// Go back again
	app.navGoBack()
	if app.selectedIdx != -1 {
		t.Errorf("after second back: expected -1, got %d", app.selectedIdx)
	}

	// Go forward
	app.navGoForward()
	if app.selectedIdx != 0 {
		t.Errorf("after forward: expected 0, got %d", app.selectedIdx)
	}

	// Go forward again
	app.navGoForward()
	if app.selectedIdx != 2 {
		t.Errorf("after second forward: expected 2, got %d", app.selectedIdx)
	}

	// Navigate somewhere new clears forward stack
	app.navigateTo(4)
	app.navGoForward()
	// Should not change (forward stack cleared)
	if app.selectedIdx != 4 {
		t.Errorf("forward after new nav should not change, got %d", app.selectedIdx)
	}
}

func TestInternalLinkAnchorResolution(t *testing.T) {
	app := newTestApp(featureTestDoc)

	// "getting-started" should resolve to the "Getting Started" heading
	idx := app.findHeadingByAnchor("getting-started")
	if idx < 0 {
		t.Fatal("expected to find heading for #getting-started")
	}
	if app.doc.Headings[idx].Text != "Getting Started" {
		t.Errorf("expected 'Getting Started', got %q", app.doc.Headings[idx].Text)
	}

	// "api-reference" should resolve
	idx = app.findHeadingByAnchor("api-reference")
	if idx < 0 {
		t.Fatal("expected to find heading for #api-reference")
	}
	if app.doc.Headings[idx].Text != "API Reference" {
		t.Errorf("expected 'API Reference', got %q", app.doc.Headings[idx].Text)
	}

	// Non-existent anchor
	idx = app.findHeadingByAnchor("nonexistent")
	if idx >= 0 {
		t.Errorf("expected -1 for nonexistent anchor, got %d", idx)
	}
}

func TestHeadingToAnchor(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Getting Started", "getting-started"},
		{"API Reference", "api-reference"},
		{"Hello World!", "hello-world"},
		{"C++ Guide", "c-guide"},
		{"Under_score", "under_score"},
	}
	for _, tt := range tests {
		got := headingToAnchor(tt.input)
		if got != tt.want {
			t.Errorf("headingToAnchor(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResponsiveSidebarWidth(t *testing.T) {
	// Short headings = narrower sidebar
	shortDoc := "# A\n## B\n## C\n"
	app := newTestApp(shortDoc)
	sw := app.sidebarWidth()

	// Long headings = wider sidebar
	longDoc := "# This is a very long heading title\n## Another quite long heading here\n"
	app2 := newTestApp(longDoc)
	sw2 := app2.sidebarWidth()

	if sw2 <= sw {
		t.Errorf("longer headings should produce wider sidebar: short=%d, long=%d", sw, sw2)
	}

	// Should never exceed 40% of width
	maxAllowed := app2.width * 2 / 5
	if sw2 > maxAllowed {
		t.Errorf("sidebar %d exceeds 40%% max %d", sw2, maxAllowed)
	}
}

func TestJumpLabelGeneration(t *testing.T) {
	// Should generate unique labels
	labels := generateLabels(5)
	if len(labels) != 5 {
		t.Fatalf("expected 5 labels, got %d", len(labels))
	}
	seen := make(map[string]bool)
	for _, l := range labels {
		if seen[l] {
			t.Errorf("duplicate label: %q", l)
		}
		seen[l] = true
	}

	// More than 24 items should use two-char labels
	labels = generateLabels(30)
	if len(labels) != 30 {
		t.Fatalf("expected 30 labels, got %d", len(labels))
	}
	for _, l := range labels[24:] {
		if len(l) != 2 {
			t.Errorf("expected 2-char label after first 24, got %q (len=%d)", l, len(l))
		}
	}
}

func TestJumpModeEnterAndCancel(t *testing.T) {
	md := "# Title\n\n```go\nfmt.Println()\n```\n\nUse `foo` here.\n"
	app := newTestApp(md)

	// Rebuild to populate codeNodes
	app.rebuildSection()

	if len(app.codeNodes) == 0 {
		t.Fatal("expected code nodes in section")
	}

	// Enter jump mode
	app.enterJumpMode()
	if app.mode != ModeJump {
		t.Fatalf("expected ModeJump, got %d", app.mode)
	}
	if app.jumpLabels == nil {
		t.Fatal("jumpLabels should be populated")
	}
	if len(app.jumpLabels) != len(app.codeNodes) {
		t.Errorf("expected %d labels, got %d", len(app.codeNodes), len(app.jumpLabels))
	}

	// Cancel with mode reset
	app.mode = ModeNormal
	app.jumpLabels = nil
	if app.mode != ModeNormal {
		t.Error("should be back to normal mode")
	}
}
