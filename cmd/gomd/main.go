package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"charm.land/glamour/v2"
	"github.com/spf13/cobra"
	"golang.org/x/term"

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
	flagRender    bool
	flagDisableBg bool
	flagTheme     string
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
  gomd -s Installation doc.md # Extract section
  gomd -r README.md            # Render to stdout (pipeable)`,
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
	rootCmd.Flags().BoolVarP(&flagRender, "render", "r", false, "Render markdown to stdout (non-interactive)")
	rootCmd.Flags().BoolVar(&flagDisableBg, "disable-background", false, "Suppress background color in render mode (useful for piping)")
	rootCmd.Flags().StringVar(&flagTheme, "theme", "", "Override theme (name from config or Ghostty themes directory)")
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

	if flagRender {
		return renderToStdout(doc, filePath)
	}

	// TUI mode
	cfg := config.Load()
	if flagTheme != "" {
		cfg.UI.Theme = flagTheme
	}
	if flagImages {
		cfg.Images.Enabled = true
	}
	if flagNoImages {
		cfg.Images.Enabled = false
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

func renderToStdout(doc *parser.Document, mdPath string) error {
	// Use terminal width if stdout is a terminal, otherwise no wrapping.
	width := 80
	if fd := int(os.Stdout.Fd()); term.IsTerminal(fd) {
		if w, _, err := term.GetSize(fd); err == nil && w > 0 {
			width = w
		}
	}

	// Load config and resolve theme to build a matching glamour style.
	cfg := config.Load()
	if flagTheme != "" {
		cfg.UI.Theme = flagTheme
	}
	theme := tui.ResolveTheme(cfg)

	style := tui.StyleConfigFromTheme(theme)
	if !flagDisableBg {
		bg := string(theme.Background)
		style.Document.BackgroundColor = &bg
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(style),
		glamour.WithWordWrap(width),
		glamour.WithChromaFormatter("terminal16m"),
	)
	if err != nil {
		return fmt.Errorf("glamour renderer: %w", err)
	}

	content := tui.PreProcessMarkdown(doc.Content)
	out, err := r.Render(content)
	if err != nil {
		return fmt.Errorf("render: %w", err)
	}
	out = tui.PostProcessLinks(out)

	if !flagDisableBg {
		// Glamour/Chroma emit \x1b[0m resets that strip the background color.
		// Re-apply the document background after every reset so the whole
		// output renders on the intended bg, similar to the TUI fix.
		bg := string(theme.Background)
		bgOpen := tui.BGOpen(bg)
		out = strings.ReplaceAll(out, "\x1b[0m", "\x1b[0m"+bgOpen)
		out = strings.ReplaceAll(out, "\x1b[m", "\x1b[m"+bgOpen)
	}

	// Image rendering via Kitty graphics protocol.
	// Outside tmux, auto-detect Kitty-capable terminals.
	// Inside tmux, require --images (passthrough works but images are "sticky").
	if imagesEnabled(cfg) && (flagImages || tui.IsKittySupported()) {
		out = replaceImagePlaceholders(out, content, mdPath)
	}

	fmt.Print(out)
	return nil
}

// imagesEnabled determines if image rendering is active based on flags and config.
func imagesEnabled(cfg config.Config) bool {
	if flagNoImages {
		return false
	}
	if flagImages {
		return true
	}
	return cfg.Images.Enabled
}

// ansiStripRe matches ANSI escape sequences (CSI, OSC, etc.) for stripping.
var ansiStripRe = regexp.MustCompile(`\x1b(?:\[[0-9;]*[a-zA-Z]|\].*?\x07|_.*?\x1b\\)`)

// replaceImagePlaceholders finds image placeholders in glamour output and replaces
// them with Kitty graphics escape sequences where possible.
func replaceImagePlaceholders(rendered, mdContent, mdPath string) string {
	images := parser.ExtractImages(mdContent)
	if len(images) == 0 {
		return rendered
	}

	// Get terminal width for sizing
	maxCols := 60
	if fd := int(os.Stdout.Fd()); term.IsTerminal(fd) {
		if w, _, err := term.GetSize(fd); err == nil && w > 0 {
			maxCols = w - 4 // leave some margin
		}
	}

	// Resolve mdPath to absolute for consistent path resolution
	absMdPath := mdPath
	if mdPath != "" && !filepath.IsAbs(mdPath) {
		if abs, err := filepath.Abs(mdPath); err == nil {
			absMdPath = abs
		}
	}

	// Build a map from image src → kitty sequence
	kittyMap := map[string]string{}
	for _, img := range images {
		resolved := tui.ResolveImagePath(img.Src, absMdPath)
		if resolved == "" {
			continue
		}
		if _, done := kittyMap[img.Src]; done {
			continue
		}
		kittySeq, err := tui.RenderKittyImage(resolved, maxCols)
		if err != nil {
			continue
		}
		kittyMap[img.Src] = kittySeq
	}
	if len(kittyMap) == 0 {
		return rendered
	}

	// Process line by line.  Glamour renders images as:
	//   <ANSI>Image: <OSC8 hyperlink>alt text<OSC8 close> →<ANSI>...
	// We strip ANSI to find lines starting with "Image: ", then match
	// the image src from the OSC8 URL in the raw line.
	lines := strings.Split(rendered, "\n")
	var result []string
	for _, line := range lines {
		stripped := ansiStripRe.ReplaceAllString(line, "")
		stripped = strings.TrimSpace(stripped)

		// Match lines that are image placeholders.
		// Glamour renders images with an OSC8 hyperlink whose URL is the original
		// image src. We detect image lines by:
		//   1. The raw line contains the src (inside the OSC8 URL).
		//   2. The visible text matches what glamour produces for images:
		//      - With alt text: "alt 🖼️ /path" or "Image: alt →"
		//      - Without alt text: just the path (with "./" and "../" stripped).
		replaced := false
		for src, kittySeq := range kittyMap {
			if !strings.Contains(line, src) {
				continue
			}
			// Compute what glamour shows as visible text for this src.
			// Glamour strips leading "./" and all leading "../" segments.
			glamourPath := src
			glamourPath = strings.TrimPrefix(glamourPath, "./")
			for strings.HasPrefix(glamourPath, "../") {
				glamourPath = strings.TrimPrefix(glamourPath, "../")
			}
			cleanStripped := strings.TrimPrefix(stripped, "/")
			isImageLine := strings.HasPrefix(stripped, "Image: ") ||
				stripped == src ||
				stripped == "/"+glamourPath ||
				cleanStripped == glamourPath ||
				strings.Contains(stripped, "🖼️")
			if isImageLine {
				result = append(result, kittySeq)
				replaced = true
				break
			}
		}
		if !replaced {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}
