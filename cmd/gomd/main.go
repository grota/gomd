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
	"github.com/grota/gomd/internal/tui"
)

var (
	flagList      bool
	flagTree      bool
	flagFilter    string
	flagLevel     int
	flagOutput    string
	flagSection   string
	flagCount     bool
	flagTheme     string
	flagColorMode string
	flagNoImages  bool
	flagImages    bool
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
  gomd -s Installation doc.md # Extract section`,
		RunE:               runRoot,
		Args:               cobra.ArbitraryArgs,
		DisableFlagParsing: false,
		TraverseChildren:   true,
	}

	// Flags
	rootCmd.Flags().BoolVarP(&flagList, "list", "l", false, "List all headings (non-interactive)")
	rootCmd.Flags().BoolVar(&flagTree, "tree", false, "Show heading tree with box-drawing characters")
	rootCmd.Flags().StringVar(&flagFilter, "filter", "", "Filter headings by text pattern (case-insensitive)")
	rootCmd.Flags().IntVarP(&flagLevel, "level", "L", 0, "Show only headings at specific level (1-6)")
	rootCmd.Flags().StringVarP(&flagOutput, "output", "o", "plain", "Output format: plain, json, tree")
	rootCmd.Flags().StringVarP(&flagSection, "section", "s", "", "Extract specific section by heading name")
	rootCmd.Flags().BoolVar(&flagCount, "count", false, "Count headings by level")
	rootCmd.Flags().StringVar(&flagTheme, "theme", "", "Override TUI theme")
	rootCmd.Flags().StringVar(&flagColorMode, "color-mode", "auto", "Terminal color mode: auto, rgb, 256")
	rootCmd.Flags().BoolVar(&flagNoImages, "no-images", false, "Disable image rendering")
	rootCmd.Flags().BoolVar(&flagImages, "images", false, "Enable image rendering (override config)")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runRoot(cmd *cobra.Command, args []string) error {
	// Load input
	inp, filePath, err := loadInput(args)
	if err != nil {
		if len(args) == 0 && !input.IsStdinPiped() {
			return cmd.Help()
		}
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	doc := parser.ParseMarkdown(inp.Content)

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

func loadInput(args []string) (*input.Input, string, error) {
	switch len(args) {
	case 0:
		if input.IsStdinPiped() {
			inp, err := input.ReadStdin()
			return inp, "", err
		}
		// No args, no stdin: print help and exit
		return nil, "", fmt.Errorf("no file specified")
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
