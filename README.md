# gomd

A terminal markdown viewer with an interactive dual-pane TUI and vim-style navigation.

## Features

- **Interactive TUI** — dual-pane layout: outline sidebar + rendered content
- **Vim-style navigation** — `j`/`k`, `gg`/`G`, `/` search
- **Fuzzy search** — sidebar heading search uses fuzzy matching (characters in order)
- **Breadcrumb bar** — title shows current heading path (e.g., `gomd — file.md > Section > Subsection`)
- **Interactive mode** (`i`) — cycle through code blocks, inline code spans, and links; copy with `y`, open links with `Enter`
- **Jump mode** (`f`) — EasyMotion-style labels to quickly jump to any selectable element
- **Navigation history** — `Ctrl+O` / `Ctrl+I` for back/forward heading navigation
- **Internal link following** — `Enter` on `#anchor` links jumps to the referenced heading
- **Mouse support** — click sidebar headings, scroll content with mouse wheel
- **Responsive sidebar** — width auto-adjusts based on heading text length
- **Multiple themes** — `T` to cycle (OceanDark, Dracula, Nord, Solarized, …)
- **CLI mode** — list, tree, count, extract sections non-interactively
- **File watcher** — auto-reloads on change
- **Editor integration** — `e` opens the source file in `$EDITOR`
- **Configurable link opener** — set `opener` in config to customize how URLs are opened

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
gomd some/dir/        # open first .md in directory
cat file.md | gomd    # read from stdin
gomd                  # print help
```

#### Key bindings

| Key | Action |
|-----|--------|
| `j` / `↓` | Next heading |
| `k` / `↑` | Previous heading |
| `gg` / `G` | Jump to root / last heading |
| `H` / `M` / `L` | Jump to top / mid / bottom of visible area |
| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |
| `/` | Search headings (fuzzy) |
| `Tab` | Switch focus (sidebar ↔ content) |
| `w` | Toggle sidebar |
| `i` | Enter interactive mode |
| `f` | Jump mode (EasyMotion labels) |
| `Ctrl+O` | Navigate back |
| `Ctrl+I` | Navigate forward |
| `e` | Open in `$EDITOR` |
| `T` | Cycle theme |
| `?` | Help |
| `q` / `Ctrl+C` | Quit |
| Mouse click | Select sidebar heading |
| Scroll wheel | Scroll content |

#### Interactive mode (`i`)

Cycles through selectable nodes in the current section: fenced code blocks, inline code spans, and links. Press `m` to cycle sub-modes.

| Key | Action |
|-----|--------|
| `j` / `k` | Next / previous node |
| `m` | Cycle sub-mode (ALL → CODE → INLINE → LINKS) |
| `H` / `M` / `L` | Jump to top / mid / bottom visible node |
| `zz` / `zt` / `zb` | Center / top / bottom node in viewport |
| `y` | Copy node content to clipboard and exit |
| `Enter` | Open link (external: browser; `#anchor`: jump to heading) |
| `Esc` | Exit interactive mode |

#### Jump mode (`f`)

Shows short labels next to all selectable elements. Type a label to jump directly to that element (enters interactive ALL mode). Press `Esc` to cancel.

### CLI mode

```bash
gomd -l file.md                  # list headings
gomd --tree file.md              # heading tree
gomd -L 2 file.md                # only h2 headings
gomd --filter "install" file.md  # filter by text
gomd --count file.md             # count by level
gomd -s "Installation" file.md   # extract section
```

## Configuration

Config file: `~/.config/gomd/config.toml`

All settings are optional — only specify what you want to override.

```toml
[ui]
theme = "OceanDark"       # OceanDark | Dracula | Nord | Gruvbox | TokyoNight
compact_tree = false      # compact sidebar tree
opener = "xdg-open"       # command to open URLs (macOS: "open")

[terminal]
color_mode = "auto"       # auto | truecolor | 256 | ansi | none

[images]
enabled = true

[content]
syntax_highlighting = true

[keys]
# Shared
quit = "q"
help = "?"
theme_picker = "T"
toggle_focus = "tab"
toggle_sidebar = "w"
reload = "r"
edit = "e"

# Sidebar navigation
sidebar_down = "j,down"
sidebar_up = "k,up"
sidebar_top = "g"
sidebar_bottom = "G"
sidebar_search = "/"
next_match = "n"
prev_match = "N"

# Content navigation
scroll_down = "j,down"
scroll_up = "k,up"
scroll_half_down = "ctrl+d,ctrl+f,pgdown, "
scroll_half_up = "ctrl+u,ctrl+b,pgup"
content_top = "g"
content_bottom = "G"
content_search = "/"
content_next_match = "n"
content_prev_match = "N"
jump = "f"
nav_back = "ctrl+o"
nav_forward = "ctrl+i"

# Node-select mode
node_select = "i"
node_next = "j,down,tab"
node_prev = "k,up,shift+tab"
node_copy = "y"
node_open = "enter"
node_exit = "esc,q,i"
```

Key bindings use comma-separated values for multiple keys mapped to the same action. Key names follow bubbletea conventions (e.g., `ctrl+d`, `shift+tab`, `pgdown`, `esc`).

## Input resolution

When no file argument is given, `gomd` reads from stdin if piped, otherwise prints help and exits.

## License

MIT
