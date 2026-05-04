# gomd

A terminal markdown viewer with an interactive dual-pane TUI and vim-style navigation.

## Features

- **Interactive TUI** — dual-pane layout: outline sidebar + rendered content
- **Vim-style navigation** — `j`/`k`, `g`/`G`, `/` search
- **Node selection** (`i`) — cycle through code blocks, inline code spans, and headings; copy with `y`
- **Multiple themes** — `T` to cycle (OceanDark, Dracula, Nord, Solarized, …)
- **CLI mode** — list, tree, count, extract sections non-interactively
- **File watcher** — auto-reloads on change
- **Editor integration** — `e` opens the source file in `$EDITOR`

## Installation

```bash
go install github.com/grota/gomd/cmd/gomd@latest
```

Or build from source:

```bash
git clone https://github.com/grota/gomd
cd gomd
go build ./cmd/gomd
```

Requires Go 1.22+.

## Usage

### Interactive TUI

```
gomd file.md          # open a file
gomd                  # open first .md in current directory
gomd some/dir/        # open first .md in directory
cat file.md | gomd    # read from stdin
```

#### Key bindings

| Key | Action |
|-----|--------|
| `j` / `↓` | Next heading |
| `k` / `↑` | Previous heading |
| `g` / `G` | Jump to root / last heading |
| `/` | Search headings |
| `Tab` | Switch focus (sidebar ↔ content) |
| `w` | Toggle sidebar |
| `i` | Enter node-select mode |
| `e` | Open in `$EDITOR` |
| `T` | Cycle theme |
| `?` | Help |
| `q` / `Ctrl+C` | Quit |

#### Node-select mode (`i`)

Cycles through selectable nodes in the current section: headings, fenced code blocks, and inline code spans.

| Key | Action |
|-----|--------|
| `j` / `k` | Next / previous node |
| `y` | Copy node content to clipboard and exit |
| `Esc` | Exit node-select mode |

### CLI mode

```bash
gomd -l file.md                  # list headings
gomd --tree file.md              # heading tree
gomd -L 2 file.md                # only h2 headings
gomd --filter "install" file.md  # filter by text
gomd --count file.md             # count by level
gomd -s "Installation" file.md   # extract section
gomd at-line 42 file.md          # heading at line number
```

## Configuration

Config file: `~/.config/gomd/config.toml`

```toml
[ui]
theme = "OceanDark"   # OceanDark | Dracula | Nord | Solarized | TokyoNight | …

[terminal]
color_mode = "auto"   # auto | truecolor | 256 | ansi | none

[images]
enabled = false

[content]
word_wrap = 0         # 0 = auto (terminal width)
```

## Input resolution

When no file argument is given, `gomd` tries in order:

1. stdin (if not a TTY)
2. First `.md` file in the current directory
3. First `.md` file in a given directory argument

## License

MIT
