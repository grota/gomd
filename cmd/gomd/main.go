package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/grota/gomd/internal/config"
	"github.com/grota/gomd/internal/input"
	"github.com/grota/gomd/internal/parser"
	"github.com/grota/gomd/internal/query"
	"github.com/grota/gomd/internal/tui"
)

var (
	flagList        bool
	flagTree        bool
	flagFilter      string
	flagLevel       int
	flagOutput      string
	flagSection     string
	flagCount       bool
	flagQuery       string
	flagQueryHelp   bool
	flagQueryOutput string
	flagTheme       string
	flagColorMode   string
	flagNoImages    bool
	flagImages      bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "gomd [OPTIONS] [FILE...]",
		Short: "A markdown navigator with tree-based structural navigation",
		Long: `gomd - A modern markdown viewer combining tree-based navigation with interactive TUI.

Launch without flags for interactive mode with dual-pane interface, vim-style navigation,
syntax highlighting, and real-time search. Use flags for CLI mode to extract, filter,
and analyze markdown structure.

Examples:
  gomd README.md              # Interactive TUI mode
  gomd -l README.md           # List all headings
  gomd --tree README.md       # Show heading tree
  gomd -s Installation doc.md # Extract section
  gomd -q '.h2' doc.md        # Query headings`,
		RunE:                  runRoot,
		Args:                  cobra.ArbitraryArgs,
		DisableFlagParsing:    false,
		TraverseChildren:      true,
	}

	// Flags
	rootCmd.Flags().BoolVarP(&flagList, "list", "l", false, "List all headings (non-interactive)")
	rootCmd.Flags().BoolVar(&flagTree, "tree", false, "Show heading tree with box-drawing characters")
	rootCmd.Flags().StringVar(&flagFilter, "filter", "", "Filter headings by text pattern (case-insensitive)")
	rootCmd.Flags().IntVarP(&flagLevel, "level", "L", 0, "Show only headings at specific level (1-6)")
	rootCmd.Flags().StringVarP(&flagOutput, "output", "o", "plain", "Output format: plain, json, tree")
	rootCmd.Flags().StringVarP(&flagSection, "section", "s", "", "Extract specific section by heading name")
	rootCmd.Flags().BoolVar(&flagCount, "count", false, "Count headings by level")
	rootCmd.Flags().StringVarP(&flagQuery, "query", "q", "", "Execute a tql query expression")
	rootCmd.Flags().BoolVar(&flagQueryHelp, "query-help", false, "Show query language documentation")
	rootCmd.Flags().StringVar(&flagQueryOutput, "query-output", "plain", "Query output format: plain, json, jsonp, jsonl, md, tree")
	rootCmd.Flags().StringVar(&flagTheme, "theme", "", "Override TUI theme")
	rootCmd.Flags().StringVar(&flagColorMode, "color-mode", "auto", "Terminal color mode: auto, rgb, 256")
	rootCmd.Flags().BoolVar(&flagNoImages, "no-images", false, "Disable image rendering")
	rootCmd.Flags().BoolVar(&flagImages, "images", false, "Enable image rendering (override config)")

	// at-line subcommand
	atLineCmd := &cobra.Command{
		Use:   "at-line <LINE>",
		Short: "Find heading at or before a specific line number",
		Args:  cobra.ExactArgs(1),
		RunE:  runAtLine,
	}
	rootCmd.AddCommand(atLineCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runRoot(cmd *cobra.Command, args []string) error {
	// --query-help doesn't require input
	if flagQueryHelp {
		printQueryHelp()
		return nil
	}

	// Load input
	inp, filePath, err := loadInput(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	doc := parser.ParseMarkdown(inp.Content)

	// Query mode
	if flagQuery != "" {
		return handleQueryMode(doc, flagQuery, flagQueryOutput)
	}

	// CLI mode if any non-TUI flags are set
	if flagList || flagTree || flagCount || flagSection != "" {
		handleCLIMode(doc)
		return nil
	}

	// TUI mode
	cfg := config.Load()
	if flagTheme != "" {
		cfg.UI.Theme = flagTheme
	}

	filename := inp.FilePath
	if filename == "" {
		filename = "stdin"
	}

	return tui.Run(doc, filename, filePath, cfg)
}

func runAtLine(cmd *cobra.Command, args []string) error {
	var line int
	if _, err := fmt.Sscanf(args[0], "%d", &line); err != nil {
		return fmt.Errorf("invalid line number: %q", args[0])
	}

	// Get file from parent flags
	parentArgs := cmd.Parent().Flags().Args()
	inp, _, err := loadInput(parentArgs)
	if err != nil {
		return err
	}

	doc := parser.ParseMarkdown(inp.Content)
	h := doc.HeadingAtLine(line)
	if h == nil {
		fmt.Println("No heading found at or before line", line)
		return nil
	}

	fmt.Printf("%s %s (line %d)\n", strings.Repeat("#", h.Level), h.Text, h.Line)
	return nil
}

func loadInput(args []string) (*input.Input, string, error) {
	switch len(args) {
	case 0:
		if input.IsStdinPiped() {
			inp, err := input.ReadStdin()
			return inp, "", err
		}
		// No args, no stdin: look for markdown files in cwd
		entries, _ := os.ReadDir(".")
		var mdFiles []string
		for _, e := range entries {
			if !e.IsDir() {
				name := e.Name()
				if strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".markdown") {
					mdFiles = append(mdFiles, name)
				}
			}
		}
		if len(mdFiles) == 0 {
			fmt.Fprintln(os.Stderr, "No markdown files found.")
			fmt.Fprintln(os.Stderr, "Usage: gomd [OPTIONS] <FILE>")
			os.Exit(0)
		}
		// Use first found file
		inp, err := input.ReadFile(mdFiles[0])
		return inp, mdFiles[0], err
	case 1:
		path := args[0]
		if path == "-" {
			inp, err := input.ReadStdin()
			return inp, "", err
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, "", fmt.Errorf("cannot open %q: %w", path, err)
		}
		if info.IsDir() {
			// Find markdown files in directory
			entries, _ := os.ReadDir(path)
			for _, e := range entries {
				name := e.Name()
				if strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".markdown") {
					fullPath := path + "/" + name
					inp, err := input.ReadFile(fullPath)
					return inp, fullPath, err
				}
			}
			return nil, "", fmt.Errorf("no markdown files in %q", path)
		}
		inp, err := input.ReadFile(path)
		return inp, path, err
	default:
		// Multiple files: open first
		inp, err := input.ReadFile(args[0])
		return inp, args[0], err
	}
}

func handleCLIMode(doc *parser.Document) {
	// Apply filters
	var headings []parser.Heading
	switch {
	case flagLevel > 0:
		headings = doc.HeadingsAtLevel(flagLevel)
	case flagFilter != "":
		headings = doc.FilterHeadings(flagFilter)
	default:
		headings = doc.Headings
	}

	switch {
	case flagCount:
		printHeadingCounts(doc)
	case flagTree:
		printTree(doc)
	case flagSection != "":
		extractSection(doc, flagSection)
	case flagList:
		printHeadings(headings)
	}
}

func printHeadings(headings []parser.Heading) {
	switch strings.ToLower(flagOutput) {
	case "json":
		b, _ := json.MarshalIndent(headings, "", "  ")
		fmt.Println(string(b))
	default:
		for _, h := range headings {
			fmt.Printf("%s %s\n", strings.Repeat("#", h.Level), h.Text)
		}
	}
}

func printTree(doc *parser.Document) {
	tree := doc.BuildTree()
	cfg := config.Load()
	compact := cfg.IsCompactTree()

	for i, node := range tree {
		isLast := i == len(tree)-1
		fmt.Print(node.RenderBoxTree("", isLast, compact))
	}
}

func printHeadingCounts(doc *parser.Document) {
	counts := make(map[int]int)
	for _, h := range doc.Headings {
		counts[h.Level]++
	}

	fmt.Println("Heading counts:")
	levels := make([]int, 0, len(counts))
	for l := range counts {
		levels = append(levels, l)
	}
	sort.Ints(levels)
	for _, l := range levels {
		fmt.Printf("  %s: %d\n", strings.Repeat("#", l), counts[l])
	}
	fmt.Printf("\nTotal: %d\n", len(doc.Headings))
}

func extractSection(doc *parser.Document, name string) {
	section, ok := doc.ExtractSection(name)
	if !ok {
		fmt.Fprintf(os.Stderr, "Section %q not found\n", name)
		os.Exit(1)
	}
	fmt.Println(section)
}

func handleQueryMode(doc *parser.Document, queryStr, outputFormat string) error {
	format, err := query.ParseOutputFormat(outputFormat)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	results, err := query.Execute(doc, queryStr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Query error:", err)
		os.Exit(1)
	}

	if len(results) == 0 {
		return nil
	}

	fmt.Println(query.FormatOutput(results, format))
	return nil
}

func printQueryHelp() {
	help := `
gomd Query Language (tql)

A jq-like query language for navigating and extracting markdown structure.

ELEMENT SELECTORS
    .h, .heading    All headings (any level)
    .h1 - .h6       Headings by level
    .code           All code blocks
    .code[rust]     Code blocks by language
    .link, .a       All links
    .link[external] External links only
    .img            All images
    .table          All tables
    .list           All lists

FILTERS & INDEXING
    .h2[Features]       Heading containing "Features" (fuzzy)
    .h2["Installation"] Heading with exact text
    .h2[0]              First h2
    .h2[-1]             Last h2
    .h2[1:3]            h2s at index 1 and 2
    .h2[:3]             First 3 h2s

HIERARCHY
    .h1 > .h2           Direct child h2s under h1s
    .h1 >> .code        Code blocks anywhere under h1s

PIPES
    .h2 | text          Get heading text (strips ##)
    [.h2] | count       Count all h2s
    .code | lang        Get code block languages
    .link | url         Get link URLs

COLLECTION FUNCTIONS
    count, length       Count elements
    first, last         First/last element
    limit(n), take(n)   First n elements
    skip(n), drop(n)    Skip first n elements
    nth(n)              Get element at index
    reverse             Reverse order
    sort                Sort alphabetically
    sort_by(key)        Sort by property
    unique              Remove duplicates
    flatten             Flatten nested arrays
    group_by(key)       Group elements by key
    min, max            Min/max value
    add                 Sum numbers or concat strings

STRING FUNCTIONS
    text                Get text representation
    upper, lower        Case conversion
    trim                Strip whitespace
    split(sep)          Split by separator
    join(sep)           Join with separator
    replace(a, b)       Replace substring
    slugify             URL-friendly slug
    lines, words, chars Count lines/words/chars

FILTER FUNCTIONS
    select(cond)        Keep if condition true (alias: where, filter)
    contains(s)         Contains substring
    startswith(s)       Starts with prefix
    endswith(s)         Ends with suffix
    matches(regex)      Matches regex pattern
    any, all            Check if any/all truthy
    not                 Negate boolean

CONTENT FUNCTIONS
    content             Section content (for headings)
    md, raw             Raw markdown
    url, href, src      Get URL/link/image source
    lang                Code block language

AGGREGATION FUNCTIONS
    stats               Document statistics
    levels              Heading count by level
    langs               Code block count by language
    types               Link types count

EXAMPLES
    # List all h2 headings
    gomd -q '.h2' doc.md

    # Get heading text only
    gomd -q '.h2 | text' doc.md

    # Count headings
    gomd -q '[.h2] | count' doc.md

    # Filter headings
    gomd -q '.h | select(contains("API"))' doc.md

    # All Rust code blocks
    gomd -q '.code[rust]' doc.md

    # External link URLs
    gomd -q '.link[external] | url' doc.md

    # h2s under "Features" section
    gomd -q '.h1[Features] > .h2' doc.md

    # JSON output
    gomd -q '.h2' --query-output json doc.md

OUTPUT FORMATS (--query-output)
    plain       Human-readable text (default)
    json        Compact JSON
    jsonp       Pretty-printed JSON
    jsonl       Line-delimited JSON
    md          Raw markdown
    tree        Tree structure
`
	fmt.Println(strings.TrimSpace(help))
}
