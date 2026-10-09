package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "gomd [OPTIONS] [FILE|DIR|-]",
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
  gomd -r README.md           # Render to stdout (pipeable)`,
		RunE:         runRoot,
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
	}

	// Flags
	rootCmd.Flags().BoolVarP(&flagList, "list", "l", false, "List all headings (non-interactive)")
	rootCmd.Flags().BoolVar(&flagTree, "tree", false, "Show heading tree with box-drawing characters")
	rootCmd.Flags().StringVar(&flagFilter, "filter", "", "Filter headings by text pattern (case-insensitive)")
	rootCmd.Flags().IntVarP(&flagLevel, "level", "L", 0, "Show only headings at specific level (1-6)")
	rootCmd.Flags().StringVarP(&flagOutput, "output", "o", "plain", "Output format for --list: plain, json")
	rootCmd.Flags().StringVarP(&flagSection, "section", "s", "", "Extract specific section by heading name")
	rootCmd.Flags().BoolVar(&flagCount, "count", false, "Count headings by level")
	rootCmd.Flags().BoolVarP(&flagRender, "render", "r", false, "Render markdown to stdout (non-interactive)")
	rootCmd.Flags().BoolVar(&flagDisableBg, "disable-background", false, "Suppress background color in render mode (useful for piping)")
	rootCmd.Flags().StringVar(&flagTheme, "theme", "", "Override theme (name from config or Ghostty themes directory)")
	rootCmd.Flags().BoolVar(&flagNoImages, "no-images", false, "Disable image rendering")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runRoot(cmd *cobra.Command, args []string) error {
	if len(args) == 0 && !input.IsStdinPiped() {
		return cmd.Help()
	}
	inp, filePath, err := loadInput(args)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Warning: ignoring config:", err)
	}
	if flagTheme != "" {
		cfg.UI.Theme = flagTheme
	}
	cfg.Images.Enabled = cfg.Images.Enabled && !flagNoImages && tui.IsKittySupported()

	doc := parser.ParseMarkdown(inp.Content)

	// CLI mode if any non-TUI flags are set
	if flagList || flagTree || flagCount || flagSection != "" {
		return handleCLIMode(doc, cfg)
	}

	if flagRender {
		dark, err := prepareThemes(cfg, term.IsTerminal(int(os.Stdout.Fd())))
		if err != nil {
			return err
		}
		return renderToStdout(doc, filePath, cfg, dark)
	}
	dark, err := prepareThemes(cfg, true)
	if err != nil {
		return err
	}

	filename := inp.FilePath
	if filename == "" {
		filename = "stdin"
	}
	return tui.Run(doc, filename, filePath, cfg, dark)
}

// prepareThemes rejects unknown theme names, downloads the pinned Ghostty
// themes when a configured theme needs them, and returns the system
// appearance (dark when the terminal is not asked or does not answer).
func prepareThemes(cfg config.Config, askTerminal bool) (bool, error) {
	source := "ui.theme"
	if flagTheme != "" {
		source = "--theme"
	}
	setting, err := tui.ParseThemeSetting(cfg.UI.Theme)
	if err != nil {
		return false, fmt.Errorf("%s: %w", source, err)
	}
	names := []string{setting.Light, setting.Dark}
	needDownload := !tui.IsBuiltinTheme(names[0]) || !tui.IsBuiltinTheme(names[1])
	if needDownload && !tui.RemoteThemesCached() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := tui.EnsureRemoteThemes(ctx)
		cancel()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Warning: using the Ocean themes:", err)
			names = nil // cannot tell unknown names from missing downloads
		}
	}
	for _, name := range names {
		if _, err := tui.LoadTheme(name, cfg.UI.ThemeOverride); err != nil {
			return false, fmt.Errorf("%w (%s)", err, source)
		}
	}
	dark := true
	if askTerminal {
		if d, ok := tui.QuerySystemDark(200 * time.Millisecond); ok {
			dark = d
		}
	}
	return dark, nil
}

func loadInput(args []string) (*input.Input, string, error) {
	if len(args) == 0 || args[0] == "-" {
		inp, err := input.ReadStdin()
		return inp, "", err
	}
	path := args[0]
	info, err := os.Stat(path)
	if err != nil {
		return nil, "", fmt.Errorf("cannot open %q: %w", path, err)
	}
	if info.IsDir() {
		// Find markdown files in directory
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, "", err
		}
		for _, e := range entries {
			name := e.Name()
			if strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".markdown") {
				fullPath := filepath.Join(path, name)
				inp, err := input.ReadFile(fullPath)
				return inp, fullPath, err
			}
		}
		return nil, "", fmt.Errorf("no markdown files in %q", path)
	}
	inp, err := input.ReadFile(path)
	return inp, path, err
}

func handleCLIMode(doc *parser.Document, cfg config.Config) error {
	switch {
	case flagCount:
		printHeadingCounts(doc)
	case flagTree:
		tree := doc.BuildTree()
		for i, node := range tree {
			fmt.Print(node.RenderBoxTree("", i == len(tree)-1, cfg.UI.CompactTree))
		}
	case flagSection != "":
		section, ok := doc.ExtractSection(flagSection)
		if !ok {
			return fmt.Errorf("section %q not found", flagSection)
		}
		fmt.Println(section)
	case flagList:
		return printHeadings(doc.FilterHeadings(flagLevel, flagFilter))
	}
	return nil
}

func printHeadings(filtered []parser.Heading) error {
	switch strings.ToLower(flagOutput) {
	case "json":
		b, err := json.MarshalIndent(filtered, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
	case "plain":
		for _, h := range filtered {
			fmt.Printf("%s %s\n", strings.Repeat("#", h.Level), h.Text)
		}
	default:
		return fmt.Errorf("unknown output format %q (want plain or json)", flagOutput)
	}
	return nil
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

func renderToStdout(doc *parser.Document, mdPath string, cfg config.Config, dark bool) error {
	// Use terminal width if stdout is a terminal, otherwise no wrapping.
	width := 80
	if fd := int(os.Stdout.Fd()); term.IsTerminal(fd) {
		if w, _, err := term.GetSize(fd); err == nil && w > 0 {
			width = w
		}
	}

	theme, _ := tui.ThemeFor(cfg, dark) // prepareThemes already warned

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

	if cfg.Images.Enabled {
		out = tui.RenderImages(out, content, mdPath, max(width-4, 1))
	}

	fmt.Print(out)
	return nil
}
