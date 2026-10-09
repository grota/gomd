# AGENTS.md

## Project Overview

`gomd` is a CLI + interactive TUI markdown viewer. It opens `.md` files in a dual-pane bubbletea interface with vim-style navigation. Also supports non-interactive modes: rendering to stdout and listing, filtering, counting or extracting headings and sections.

**Tech stack:** Go, Cobra (CLI), Bubbletea (TUI), Glamour v2 (rendering), Chroma (highlighting), Kitty graphics protocol with Unicode placeholders (images), TOML config.

## Setup

Standard Go toolchain. A `justfile` wraps the common commands (`just check` runs vet, format check and tests).

```bash
go build ./cmd/gomd    # produces ./gomd (gitignored)
go test ./...
```

The Go version in `go.mod` is `1.26.2` — use the matching toolchain to avoid toolchain mismatch errors.

## Key Commands

| Task            | Command                                    |
| --------------- | ------------------------------------------ |
| Build           | `go build ./cmd/gomd`                      |
| Run all tests   | `go test ./...`                            |
| Run single test | `go test ./internal/parser/ -run TestName` |
| Vet             | `go vet ./...`                             |
| Format          | `gofmt -w .`                               |

`internal/config`, `internal/parser` and `internal/tui` have tests.

## Architecture Notes

- Single entrypoint: `cmd/gomd/main.go` — all CLI flags, mode routing.
- **Three execution modes** selected by flags:
  1. TUI (default, no flags): `gomd file.md`
  2. Render (`-r`/`--render`): non-interactive stdout rendering with `--disable-background` and `--no-images` options
  3. CLI (`--list`, `--tree`, `--count`, `--section`): non-interactive structured output
- Subcommands: none. At most one positional argument (file, directory, or `-`).
- Input resolution: explicit arg → stdin → help and exit.
- Config file: `os.UserConfigDir()/gomd/config.toml` (`~/.config` on Linux, `~/Library/Application Support` on macOS). TOML sections: `[ui]`, `[ui.theme_override]`, `[images]`, `[keys]`. Loaded once in `main.go`; an invalid file is ignored with a warning.
- The theme picker persists only `ui.theme` via `config.SaveTheme`, editing that single line so user comments and keys survive. Never marshal the whole `Config` back to disk.
- Config structure is defined in `internal/config/config.go` (struct fields + defaults in `Default()` / `DefaultKeys()`).
- When adding, changing, or removing a config field or keybinding: update the struct, its default, the help overlay (`overlayHelp` in `internal/tui/view.go`) and the README config example. Match keys with `config.KeyMatches`; two-key sequences (`gg`, `zz`) go through `config.IsSequencePrefix`. Do not hardcode key strings except `esc` as a universal cancel.

## Package Boundaries

```
cmd/gomd/main.go         # Cobra entrypoint, all flag handling, renderToStdout()
internal/config/         # TOML config loader
internal/input/          # Stdin/file reading
internal/parser/         # ParseMarkdown(), Heading/Document types, section extraction
internal/tui/tui.go      # App state, lifecycle (reload, file watcher, opener), Run() entry
internal/tui/theme.go    # Built-in themes, ResolveTheme, glamour StyleConfig
internal/tui/nodes.go    # Section building, glamour pre/post-processing, node extraction
internal/tui/keys.go     # Key handling per mode/focus, navigation, search
internal/tui/view.go     # Layout, panes, node-to-line mapping, highlights, overlays
internal/tui/ghostty.go  # Ghostty theme file loader
internal/tui/kitty.go    # Images: Kitty Unicode placeholders, terminal/tmux detection, caption matching
```

### Rendering Pipeline

Markdown content goes through a three-stage pipeline before display:

1. **Pre-process** (`PreProcessMarkdown()`): converts HTML tags to markdown equivalents before glamour.
   - `<img src="url" alt="text">` → `![text](url)`
   - `<a href="url">text</a>` → `[text](url)`
2. **Glamour render**: renders markdown to ANSI-styled text. Custom `ImageText.Format` renders images as `alt 📷 path` instead of glamour's default `Image: alt → path`.
3. **Post-process** (`PostProcessLinks()`): inserts 🔗 glyph between link text and URL in the ANSI output.

Both `renderGlamour()` (TUI) and `renderToStdout()` (CLI render mode) apply this pipeline.

4. **Images** (`placeImages()` in `kitty.go`): finds caption lines (`alt 📷 /path`) and inserts Kitty Unicode placeholder rows. Each image is uploaded once (`a=T,U=1,q=2`, random 24-bit id in the foreground colour) and then drawn as plain text cells, so bubbletea diffs it and tmux scrolls it. The TUI sends uploads from `Update` via `tea.Raw` (View cannot return commands) and frees images on exit. Do not write raw image escape sequences into the view: bubbletea v2's cell renderer drops them, and tmux cannot track them.

Node extraction (`extractCodeNodes`/`extractLinkNodes`) works on **raw** markdown lines (before pre-processing), so it must handle both markdown and HTML syntax (`<img>`, `<a>` tags). This ordering is intentional: the extracted `colStart`/`colEnd` byte offsets must match the raw source lines so that `highlightSpanInLine` can highlight the correct column range in rendered output. Running extraction after pre-processing would break these offsets because `PreProcessMarkdown` changes line content and length (e.g. a long `<img>` tag becomes a shorter `![alt](url)`). The `display` field on `codeNode` bridges the gap by storing the visible text glamour renders, enabling mapping from nodes to rendered lines without relying on column offsets for line-level identification.

## Package Management

Go modules only (`go.mod` / `go.sum`); the `justfile` has a `tidy` recipe.

### Dependency Safety

Before adding or upgrading any dependency:

1. **Never assume you know the latest version.** Training data is outdated. Always verify first.
2. Check the live registry:
   ```bash
   curl -s "https://proxy.golang.org/<module>/@latest" | jq .
   ```
3. Avoid releases published within the last 5 days.
4. Run `go mod tidy` after changing `go.mod`.

## Testing

```bash
go test ./...                                        # all tests
go test ./internal/parser/ -run TestName             # single parser test
```

No test CI. `.github/workflows/release.yml` runs goreleaser (`.goreleaser.yaml`) on `v*` tags. No integration test prerequisites.

## Git Workflow

### Commits

Follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/):

```
<type>(<scope>): <description>
```

**Types:** `feat`, `fix`, `refactor`, `docs`, `test`, `chore`.

### Branching

- Branch naming: `feat/`, `fix/`, `chore/` prefix + kebab-case description.
- Never push directly to `main`.

### Rebasing

- Rebase onto `main` before pushing. No merge commits.
- Use `--force-with-lease`, never `--force`.

## Command Safety

### Safe (run autonomously)

- `go build ./cmd/gomd`
- `go test ./...` and any focused `go test` invocation
- `go vet ./...`
- `gofmt -w .`
- `go mod tidy`
- `git status`, `git log`, `git diff`

### Dangerous (ask user first)

- `go get` / dependency changes
- `git push`

### Destructive (never run)

- `rm -rf`
- `git push --force`

## Important Rules

- The compiled binary `./gomd` is gitignored — do not commit it.
- Match the Go toolchain version from `go.mod` (`1.26.2`); mismatches cause build errors.
- There is no linter config — `go vet` + `gofmt` are the only enforced checks.
- No codegen steps — do not add `go generate` without documenting it here.
- Always verify dependency versions against the live Go module proxy before adding.
