<div align="center">
  <h1>gomd</h1>
</div>
A terminal markdown viewer with interactive element selection and vim-style navigation.

<br>
<br>

Think [glow](https://github.com/charmbracelet/glow) but with more interactivity with the markdown document.

## Screenshots

<div align="center">
normal mode:
</div>

<img width="1536" height="1022" alt="gomd screenshot 1" src="https://github.com/user-attachments/assets/0f0db041-0dc3-4d35-8ee1-d89701707aff" />

---

<div align="center">
jump mode:
</div>

<img width="1536" height="1020" alt="gomd screenshot 2" src="https://github.com/user-attachments/assets/ba477053-48d4-4380-b682-8f2bf39b72b2" />

---

<div align="center">
interactive mode:
</div>

<img width="1536" height="1020" alt="gomd screenshot 3" src="https://github.com/user-attachments/assets/d0c03603-662a-4092-ad15-e03c8c1f6aaa" />

## Features

- **Interactive TUI** — dual-pane layout: outline sidebar + rendered content
- **Vim-style navigation** — `j`/`k`, `gg`/`G`, `/` search
- **Fuzzy search** — sidebar heading search uses fuzzy matching (characters in order)
- **Breadcrumb bar** — title shows current heading path (e.g., `gomd — file.md > Section > Subsection`)
- **Interactive mode** (`i`) — cycle through code blocks, inline code spans, and links; copy with `y`, open links with `Enter` (works in any sub-mode)
- **Jump mode** (`f`) — EasyMotion-style labels to quickly jump to any selectable element
- **Navigation history** — `Ctrl+O` / `Ctrl+I` for back/forward heading navigation
- **Internal link following** — `Enter` on `#anchor` links jumps to the referenced heading
- **Mouse support** — click sidebar headings, scroll content with mouse wheel
- **Responsive sidebar** — width auto-adjusts based on heading text length
- **Multiple themes** — `T` opens the theme picker (OceanDark, Nord, Dracula, Gruvbox, TokyoNight, or any Ghostty theme)
- **CLI mode** — list, tree, count, extract sections non-interactively
- **File watcher** — auto-reloads on change
- **Editor integration** — `e` opens the source file in `$EDITOR`
- **Configurable link opener** — set `opener` in config to customize how URLs are opened
- **Image rendering** — local images in the TUI and render mode via Kitty Unicode placeholders (Kitty, Ghostty), including inside tmux
- **HTML tag support** — `<img>` and `<a>` HTML tags are rendered as images and links alongside native markdown syntax

## Installation

Download the [latest release](https://github.com/grota/gomd/releases/latest) from this project.

Or via go install:

```bash
go install github.com/grota/gomd/cmd/gomd@latest
```

Or build from source:

```bash
git clone https://github.com/grota/gomd
cd gomd
go build ./cmd/gomd
```

Requires Go 1.26+.

## Usage

### Interactive TUI

```
gomd file.md          # open a file
gomd some/dir/        # open first .md in directory (alphabetical)
cat file.md | gomd    # read from stdin
gomd                  # print help
```

### Render Mode

Render mode (`-r` / `--render`) outputs rendered markdown to stdout without launching the TUI. Useful for piping, previewing, or printing.

```
gomd file.md -r                        # render to stdout with background
gomd file.md -r --disable-background   # render without background (for piping)
gomd file.md -r --no-images            # render without images
```

Background color is emitted by default so the output looks correct in the terminal. Use `--disable-background` when piping to other tools or files, since ANSI background sequences can interfere with downstream processing.

Images are rendered as `alt 📷 path` and links as `text 🔗 url`. HTML `<img>` and `<a>` tags are converted to their markdown equivalents before rendering.

#### Key bindings

| Key                 | Action                                      |
| ------------------- | ------------------------------------------- |
| `j` / `↓`           | Next heading                                |
| `k` / `↑`           | Previous heading                            |
| `gg` / `G`          | Jump to root / last heading                 |
| `H` / `M` / `L`     | Jump to top / mid / bottom of visible area  |
| `zz` / `zt` / `zb`  | Center / top / bottom selection in viewport |
| `Ctrl+D` / `Ctrl+U` | Half page down / up                         |
| `/`                 | Search headings (fuzzy) or content          |
| `n` / `N`           | Next / previous search match                |
| `Tab`               | Switch focus (sidebar ↔ content)            |
| `w`                 | Toggle sidebar                              |
| `i`                 | Enter interactive mode                      |
| `f`                 | Jump mode (EasyMotion labels)               |
| `Ctrl+O`            | Navigate back                               |
| `Ctrl+I`            | Navigate forward                            |
| `e`                 | Open in `$EDITOR`                           |
| `r`                 | Reload file                                 |
| `T`                 | Theme picker (selection is saved to config) |
| `?`                 | Help                                        |
| `q` / `Ctrl+C`      | Quit                                        |
| Mouse click         | Select sidebar heading                      |
| Scroll wheel        | Scroll content                              |

`Ctrl+I` is indistinguishable from `Tab` on terminals without the Kitty keyboard protocol, so there it toggles focus instead of navigating forward. Rebind `nav_forward` if that matters to you.

#### Interactive mode (`i`)

Cycles through selectable nodes in the current section: fenced code blocks, inline code spans, and links. Press `m` to cycle sub-modes.

| Key                | Action                                                                  |
| ------------------ | ----------------------------------------------------------------------- |
| `j` / `k`          | Next / previous node                                                    |
| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)                            |
| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                                 |
| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                                  |
| `y`                | Copy node content (exits in CODE and INLINE sub-modes)                  |
| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |
| `Esc`              | Exit interactive mode                                                   |

#### Jump mode (`f`)

Shows short labels next to all selectable elements. Type a label to jump directly to that element (enters interactive ALL mode). Press `Esc` to cancel.

### CLI mode

```bash
gomd -l file.md                  # list headings
gomd --tree file.md              # heading tree
gomd -l -L 2 file.md             # only h2 headings
gomd -l --filter "install" file.md  # filter by text (combines with -L)
gomd -l -o json file.md          # headings as JSON
gomd --count file.md             # count by level
gomd -s "Installation" file.md   # extract section
```

## Configuration

Config file: `~/.config/gomd/config.toml` on Linux, `~/Library/Application Support/gomd/config.toml` on macOS. An invalid file is ignored with a warning.

All settings are optional — only specify what you want to override.

```toml
[ui]
theme = "OceanDark"       # OceanDark | Nord | Dracula | Gruvbox | TokyoNight (or a Ghostty theme name)
compact_tree = false      # compact --tree output
sidebar_hidden = true     # start with the sidebar hidden
opener = "xdg-open"       # command to open URLs, may include arguments (macOS default: "open")
ghostty_theme_directory = ""  # path to Ghostty themes (e.g., "/usr/share/ghostty/themes/")

[ui.theme_override]
# Override individual colors of the active theme (hex values).
# Any field left empty keeps the theme default.
# border = "#504945"
# selected = "#3c3836"
# heading1 = "#fabd2f"
# heading2 = "#b8bb26"
# heading3 = "#83a598"
# headingn = "#928374"
# background = "#282828"
# foreground = "#ebdbb2"
# statusbar = "#3c3836"
# highlight = "#fe8019"
# code = "#1d2021"
# search = "#fb4934"
# nodesel = "#8ec07c"

[images]
enabled = true

[keys]
# Shared
quit = "q,esc"
help = "?"
theme_picker = "T"
toggle_focus = "tab"
toggle_sidebar = "w"
reload = "r"
edit = "e"
nav_back = "ctrl+o"
nav_forward = "ctrl+i"

# Sidebar navigation
sidebar_down = "j,down"
sidebar_up = "k,up"
sidebar_top = "gg"
sidebar_bottom = "G"
sidebar_search = "/"
next_match = "n"
prev_match = "N"

# Content navigation
scroll_down = "j,down"
scroll_up = "k,up"
scroll_half_down = "ctrl+d,ctrl+f,pgdown,space"
scroll_half_up = "ctrl+u,ctrl+b,pgup"
content_top = "gg"
content_bottom = "G"
content_search = "/"
content_next_match = "n"
content_prev_match = "N"
jump = "f"

# Viewport (sidebar and interactive mode)
view_center = "zz"
view_top = "zt"
view_bottom = "zb"
jump_high = "H"
jump_mid = "M"
jump_low = "L"

# Interactive (node-select) mode
node_select = "i"
node_next = "j,down,tab"
node_prev = "k,up,shift+tab"
node_mode = "m"
node_copy = "y"
node_open = "enter"
node_exit = "esc,q,i"
```

Key bindings use comma-separated values for multiple keys mapped to the same action. Key names follow bubbletea conventions (e.g., `ctrl+d`, `shift+tab`, `pgdown`, `space`, `esc`). A two-character entry such as `gg` is a key sequence. `Esc` always cancels search, jump mode and the theme picker.

## Theme resolution

gomd resolves the active theme in three steps:

1. **Built-in lookup** — if `theme` matches a built-in name (OceanDark, Nord, Dracula, Gruvbox, TokyoNight), that theme is used.
2. **Ghostty fallback** — if the name doesn't match a built-in theme and `ghostty_theme_directory` is set, gomd looks for a file with that name in the directory and converts the Ghostty color palette to a gomd theme. The mapping is:
   - `background` / `foreground` → direct
   - `selection-background` → Selected (fallback: palette 0)
   - palette 0 → Code
   - palette 1 → Search (red)
   - palette 2 → Heading2 (green)
   - palette 3 → Heading1 (yellow)
   - palette 4 → Heading3 (blue)
   - palette 6 → NodeSel (cyan)
   - palette 8 → Border, StatusBar, HeadingN (bright black)
   - palette 11 → Highlight (bright yellow)
3. **Color overrides** — any non-empty field in `[ui.theme_override]` replaces the corresponding color, regardless of whether the base theme is built-in or loaded from Ghostty.

If the theme name doesn't match a built-in and no Ghostty directory is configured (or the file isn't found), gomd falls back to OceanDark.

## Input resolution

When no file argument is given, `gomd` reads from stdin if piped, otherwise prints help and exits.

### Images

gomd shows local images in the TUI and in render mode (`-r`) on terminals with [Kitty Unicode placeholder](https://sw.kovidgoyal.net/kitty/graphics-protocol/#unicode-placeholders) support: **Kitty** and **Ghostty**. In the TUI, the caption line (`alt 📷 path`) stays selectable and the image is drawn below it.

Images are on when the terminal is detected as Kitty or Ghostty, unless turned off with the **`--no-images`** flag or in the config file:

```toml
[images]
enabled = true   # default
```

gomd detects Kitty or Ghostty from `KITTY_WINDOW_ID`, `TERM_PROGRAM` and `TERM`.

**tmux**: gomd asks tmux which terminal the client uses and shows images when it is Kitty or Ghostty and `allow-passthrough` is `on` (needed to upload the image). Images scroll and survive window switches like text. tmux must pass 24-bit colour to the terminal (`terminal-features` with `RGB`, the default for most setups), because the image id is encoded in the text colour.

Only local image files are supported (relative or absolute paths). URLs are not fetched; they display as text.

## License

MIT
