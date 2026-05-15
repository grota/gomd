# AGENTS.md

## Project Overview

`gomd` is a CLI + interactive TUI markdown viewer. It opens `.md` files in a dual-pane bubbletea interface with vim-style navigation. Also supports non-interactive modes for listing headings, querying markdown structure via a custom `tql` language, and finding the heading at a given line number.

**Tech stack:** Go, Cobra (CLI), Bubbletea (TUI), Glamour v2 (rendering), Chroma (highlighting), Kitty graphics protocol (images), TOML config.

## Setup

No task runner. Standard Go toolchain only.

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

Only `internal/parser` and `internal/tui` have tests.

## Architecture Notes

- Single entrypoint: `cmd/gomd/main.go` — all CLI flags, mode routing.
- **Three execution modes** selected by flags:
  1. TUI (default, no flags): `gomd file.md`
  2. Render (`-r`/`--render`): non-interactive stdout rendering with `--disable-background` and `--images` options
  3. CLI (`--list`, `--tree`, `--count`, `--section`): non-interactive structured output
- Subcommand: none (at-line was removed).
- Input resolution: explicit arg → stdin → help and exit.
- Config file: `~/.config/gomd/config.toml` (TOML, sections: `[ui]`, `[terminal]`, `[images]`, `[content]`, `[keys]`).
- Config structure is defined in `internal/config/config.go` (struct fields + defaults in `DefaultConfig()`).
- When adding, changing, or removing a config field or keybinding: update the struct, its default, and the help overlay in the TUI.

## Package Boundaries

```
cmd/gomd/main.go         # Cobra entrypoint, all flag handling, renderToStdout()
internal/config/         # TOML config loader
internal/input/          # Stdin/file reading
internal/parser/         # ParseMarkdown(), Heading/Document types, section extraction
internal/tui/            # Bubbletea TUI (Run() entry)
internal/tui/kitty.go    # Kitty graphics protocol: image rendering, tmux passthrough
```

## Package Management

No task runner wrapper — use Go toolchain directly.

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

No CI is configured. No integration test prerequisites.

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
