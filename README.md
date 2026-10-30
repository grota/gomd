# gomd

**A markdown viewer for the terminal that lets you _use_ the document, not just read it.**

[![CI](https://github.com/grota/gomd/actions/workflows/ci.yml/badge.svg)](https://github.com/grota/gomd/actions/workflows/ci.yml)
[![Latest release](https://img.shields.io/github/v/release/grota/gomd)](https://github.com/grota/gomd/releases/latest)
[![Go version](https://img.shields.io/github/go-mod/go-version/grota/gomd)](go.mod)
[![License: MIT](https://img.shields.io/github/license/grota/gomd)](LICENSE)

Browse a document by its outline, pick any code block, code span or link with a few keys, and copy it or open it. Select text with the mouse and copy what you see or the markdown behind it. Images, 700+ themes and vim keys included.

<img width="1918" height="1009" alt="gomd: outline sidebar and rendered document" src="https://github.com/user-attachments/assets/3b75ed06-affa-44b7-873d-f5e38fdec3ee" />

## Highlights

- **Outline and content side by side.** A heading sidebar (fuzzy search, breadcrumb title) next to the rendered section, with vim-style navigation.
- **Every element is selectable.** Interactive mode (`i`) steps through code blocks, inline code, links and images: `y` copies, `Enter` opens.
- **Jump labels.** `f` puts EasyMotion-style labels on every element; type one to land on it.
- **Copy what you see, or its source.** Drag to copy the rendered text; press `y` to copy the exact markdown behind it instead, snapped to whole links, code spans and tables. On Kitty and Ghostty 1.4+ the drag copy also carries rich HTML, images included, for pasting into mail or documents.
- **Follow links.** `#anchors` jump to their heading, links to local `.md` files open in gomd, and `Ctrl+O` / `Ctrl+I` walk back and forth through the history.
- **Images in the terminal.** Local and remote images, drawn on Kitty and Ghostty, even inside tmux.
- **700+ themes.** Ghostty themes with a live-preview picker, a theme for light and one for dark following the system appearance, and code coloured with each theme's own palette.
- **Edit at the right line.** `e` opens `$EDITOR` at the line you are looking at.
- **Scriptable.** Render to stdout, or list, count and extract headings and sections, as text or JSON.

<details>
<summary><b>More screenshots</b>: jump mode and interactive mode</summary>

<p align="center"><b>Jump mode</b></p>
<img width="1918" height="1009" alt="gomd jump mode: labels on every selectable element" src="https://github.com/user-attachments/assets/d46f4d82-a60f-42cd-afdd-914c0892ed7d" />

<p align="center"><b>Interactive mode</b></p>
<img width="1918" height="1009" alt="gomd interactive mode: a selected block" src="https://github.com/user-attachments/assets/e103eaa3-f4ca-4d8a-a0e9-285c78f022bc" />


</details>

## Installation

Download a binary from the [latest release](https://github.com/grota/gomd/releases/latest), or install with Go (1.26 or newer):

```bash
go install github.com/grota/gomd/cmd/gomd@latest
```

Or build from source:

```bash
git clone https://github.com/grota/gomd
cd gomd
go build ./cmd/gomd
```

## Quick start

```bash
gomd README.md              # open a file in the viewer
gomd docs/                  # open the first .md file of a folder (alphabetical)
cat notes.md | gomd         # read from stdin
gomd -r README.md           # render to stdout, no viewer
gomd -l README.md           # list the headings
gomd -s Installation doc.md # print one section
```

With no file and nothing on stdin, gomd prints its help. Press `?` in the viewer for the key bindings.

## The viewer

### Keys

| Key                 | Action                                              |
| ------------------- | --------------------------------------------------- |
| `j` / `↓`           | Next heading (sidebar) or scroll down (content)     |
| `k` / `↑`           | Previous heading or scroll up                       |
| `gg` / `G`          | Whole document / last heading, or top / bottom      |
| `Ctrl+D` / `Ctrl+U` | Half page down / up                                 |
| `H` / `M` / `L`     | Top / middle / bottom of the visible area           |
| `zz` / `zt` / `zb`  | Center / top / bottom the selection in the view     |
| `/`                 | Search headings (fuzzy) or content                  |
| `n` / `N`           | Next / previous search match                        |
| `Tab`               | Switch focus between sidebar and content            |
| `w`                 | Show or hide the sidebar                            |
| `i`                 | Interactive mode                                    |
| `f`                 | Jump mode                                           |
| `Ctrl+O` / `Ctrl+I` | Back / forward in the history                       |
| `e`                 | Open in `$EDITOR` at the shown line or search match |
| `r`                 | Reload the file (it also reloads on change)         |
| `T`                 | Theme picker                                        |
| `?`                 | Help                                                |
| `q` / `Ctrl+C`      | Quit                                                |

The sidebar sizes itself to the headings. The mouse works too: click a heading in the sidebar, scroll the content with the wheel, drag over it to select. The keys can be changed in the [configuration](#configuration).

### Interactive mode

`i` selects the elements of the current section one at a time: code blocks, inline code, links and images. `m` cycles which kinds are visited.

| Key                | Action                                                             |
| ------------------ | ------------------------------------------------------------------ |
| `j` / `k`          | Next / previous element                                            |
| `m`                | Cycle the kinds: ALL → CODE → INLINE → LINKS                       |
| `H` / `M` / `L`    | Top / middle / bottom visible element                              |
| `zz` / `zt` / `zb` | Center / top / bottom the element in the view                      |
| `y`                | Copy the element (leaves the mode in CODE and INLINE)              |
| `Enter`            | Open a link: in the browser, at its heading, or a local `.md` file |
| `Esc`              | Leave interactive mode                                             |

### Jump mode

`f` labels every selectable element on screen. Type a label to select that element in interactive mode; `Esc` cancels.

### Selecting and copying

Dragging over the content copies the selected text, as shown, to the system clipboard. While the selection is shown, `y` copies its markdown source instead. The selection snaps to whole bold, italic and strikethrough spans, inline code, links, images, tables and HTML blocks, so both copies cover what is highlighted. The markdown copy also takes a line's prefix (`- `, `> `, `## `) when the selection starts before the line's text, and a code block's fences when it covers the whole block. A selected image caption is copied as `![alt](path)`. Hold `Shift` while dragging to use the terminal's own selection.

Copies go through OSC 52, so they work over SSH. Inside tmux, gomd hands them to `tmux load-buffer -w`, which forwards them to the terminal when `set-clipboard` is `on` or `external` and the terminal has the `clipboard` feature:

```tmux
set -s set-clipboard external
set -as terminal-features ',xterm*:clipboard'
```

On terminals with the Kitty clipboard protocol (OSC 5522: kitty, Ghostty 1.4+), a drag copy also carries the selection as HTML, with local images up to 5 MB inlined. Applications that paste rich text (office suites, mail clients) get the formatting and the images; terminals and editors still get plain text. gomd recognises these terminals from `TERM_PROGRAM`, `TERM_PROGRAM_VERSION`, `KITTY_WINDOW_ID`, or tmux's `client_termtype`. Inside tmux this needs `allow-passthrough on`, as images do.

### Links, history and your editor

`Enter` on a link opens it with the configured `opener` (`xdg-open` on Linux, `open` on macOS). A `#anchor` link jumps to its heading (GitHub slugs, `-1` suffixes for repeated headings included), and a link to a local `.md` file, such as `docs/setup.md#install`, opens that file in gomd. `Ctrl+O` and `Ctrl+I` go back and forward across headings and files.

`e` opens the file in `$EDITOR` (`vi` when unset) at the current content search match when it is on screen, otherwise at the first line shown. The line is passed as `+N` (vi, vim, nvim, nano, emacs, kak, micro, joe, mg, ne), `--goto file:N` (VS Code, VSCodium, Cursor, Windsurf) or `file:N` (Sublime Text, Zed, Helix); other editors open the file at its top.

## Render mode

`-r` (`--render`) prints the rendered document to stdout, without the viewer: for previews, pagers and pipes.

```bash
gomd -r file.md                       # with the theme's background
gomd -r --disable-background file.md  # without it, for pipes and files
gomd -r --no-images file.md           # captions instead of images
gomd -r --theme Dracula file.md       # any theme by name
```

HTML `<a>` and `<img>` tags count as links and images. Links show as `text 🔗 url` and images as `alt 📷 path`, with the image drawn below on terminals that support it. The background is on by default so the output looks right in the terminal; turn it off when another program reads the output.

## CLI mode

```bash
gomd -l file.md                     # list headings
gomd -l -L 2 file.md                # only level 2 headings
gomd -l --filter install file.md    # filter by text (combines with -L)
gomd -l -o json file.md             # headings as JSON
gomd --tree file.md                 # heading tree
gomd --count file.md                # headings per level
gomd -s Installation file.md        # print a section
```

## Images

gomd draws local and remote images, in the viewer and in render mode, on terminals with [Kitty Unicode placeholders](https://sw.kovidgoyal.net/kitty/graphics-protocol/#unicode-placeholders): **Kitty** and **Ghostty**, recognised from `KITTY_WINDOW_ID`, `TERM_PROGRAM` and `TERM`. In the viewer the caption line (`alt 📷 path`) stays selectable and the image is drawn below it. Images are off with `--no-images` or `enabled = false` in the [configuration](#configuration).

**tmux**: gomd asks tmux which terminal the client uses, and draws images when it is Kitty or Ghostty and `allow-passthrough` is `on`. Images then scroll and survive window switches like text. tmux must pass 24-bit colour to the terminal (`terminal-features` with `RGB`, the default for most setups), because the image id travels in the text colour.

**Remote images** (`http://`, `https://`) are downloaded into the user cache directory (`~/.cache/gomd/images/` on Linux), one file per URL. In the viewer they appear when their download finishes; render mode waits up to 15 seconds per image.

- A copy is used for 7 days, then downloaded again; until then, or when the download fails (offline), the old copy is shown.
- Copies older than 30 days are removed, and the oldest go first when the cache passes 100 MB.
- gomd shows PNG, JPEG and GIF. Other formats, such as the SVG of most README badges, keep their caption; they are refused from the response's `Content-Type` or when the bytes do not decode, are not reported as failures, and are requested again on each start.
- A download over `remote_max_mb` is refused, and a failed download is not retried until gomd restarts.

Fetching an image tells its server that the document was opened. Set `remote = false` to keep gomd offline; images then show as their caption.

## Themes

gomd ships two themes, **OceanDark** and **OceanLight**, and can use the 700+ themes of [iTerm2-Color-Schemes](https://github.com/mbadolato/iTerm2-Color-Schemes) (MIT) in Ghostty format. With one of those, code blocks take their colours from the theme's 16-colour palette (magenta keywords, green strings, blue functions), lightened or darkened where needed to stay readable on the code background.

The `theme` setting uses Ghostty's format: `"light:A,dark:B"` picks a theme for each system appearance, and a single name is used for both. The default is `"light:OceanLight,dark:OceanDark"`. gomd follows the system's light or dark setting as reported by the terminal (mode 2031: Ghostty, and tmux 3.6+). It asks once at startup and switches live when the setting changes; without an answer it uses the dark theme. Render mode asks only when its output is a terminal.

### Theme picker

`T` opens the picker with a **Dark** and a **Light** tab. Dark lists the darkest backgrounds first, Light the lightest first, with the built-in theme at the top. Typing filters the list (fuzzy, like the heading search), and the screen previews the highlighted theme live. `●` marks the tab of the current system appearance; if the system switches while the picker is open, the picker says which theme will be used when it closes.

| Key                            | Action                                    |
| ------------------------------ | ----------------------------------------- |
| `↑` / `↓`, `Ctrl+P` / `Ctrl+N` | Move                                      |
| `PgUp` / `PgDn`                | Move a page                               |
| `Backspace` / `Ctrl+U`         | Delete a character / clear the filter     |
| `Tab`                          | Switch between Dark and Light             |
| `Enter`                        | Save the theme for that tab to the config |
| `Esc`                          | Cancel and restore the theme              |

These keys are fixed, since every other key goes into the filter. Saving changes only the `theme` line of the config file, so its comments and other settings stay.

### Downloads

The Ghostty themes come from a pinned release, verified by SHA-256 and cached in the user cache directory (`~/.cache/gomd/themes/` on Linux). gomd downloads them the first time the picker opens, or at startup when `theme` names a theme that is not cached yet; it waits at most 5 seconds, and on failure uses the Ocean themes with a warning. A name that is neither built-in nor in the pinned set is an error.

## Configuration

The config file is `~/.config/gomd/config.toml` on Linux and `~/Library/Application Support/gomd/config.toml` on macOS. Every setting is optional; an invalid file is ignored with a warning. These are the defaults:

```toml
[ui]
theme = "light:OceanLight,dark:OceanDark"   # or a single name; see Themes
compact_tree = false      # compact --tree output
sidebar_hidden = true     # start with the sidebar hidden
content_border = "always" # or "with_sidebar": no content border while the sidebar is hidden
opener = "xdg-open"       # command to open URLs, may include arguments (macOS default: "open")

[ui.theme_override]
# Override single colours of every theme (hex values); empty keeps the theme's.
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
# code = "#1d2021"       # tint of selectable elements in interactive mode
# search = "#fb4934"
# nodesel = "#8ec07c"

[images]
enabled = true
remote = true        # download http(s) images
remote_max_mb = 10   # largest remote image downloaded, in MB

[keys]
# Shared
quit = "q,esc"
help = "?"
theme_picker = "T"
toggle_focus = "tab"
toggle_sidebar = "w"
reload = "r"
edit = "e"
copy_markdown = "y"       # copy the mouse selection's markdown source
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

# Interactive mode
node_select = "i"
node_next = "j,down,tab"
node_prev = "k,up,shift+tab"
node_mode = "m"
node_copy = "y"
node_open = "enter"
node_exit = "esc,q,i"
```

A key setting takes comma-separated keys for the same action. Key names follow bubbletea (`ctrl+d`, `shift+tab`, `pgdown`, `space`, `esc`), and a two-character entry such as `gg` is a key sequence. `Esc` always cancels search, jump mode and the theme picker.

## Known issues

- **`Ctrl+I` is `Tab`** on terminals without the Kitty keyboard protocol, so there it switches focus instead of going forward in the history. Rebind `nav_forward` if you need it.
- **Ghostty with a single theme reports "light"** to the light/dark query, so gomd picks its light theme on a dark system. Use a `light:A,dark:B` pair in the Ghostty config until [ghostty-org/ghostty#13605](https://github.com/ghostty-org/ghostty/pull/13605) is released.
- **Link URLs with some non-ASCII characters** (such as `✓`, Cyrillic `М` or Greek `Μ`) show garbled, and their link opens a corrupted address, because of a bug in the terminal library ([charmbracelet/x#848](https://github.com/charmbracelet/x/issues/848)).
- **Untested: OSC 5522 inside tmux.** The terminal answers each rich copy with a status reply, which tmux may pass on as typed keys, and gomd acts on keys such as `q` (quit) or `e` (edit). If gomd reacts on its own right after a drag copy, please open an issue.

## License

[MIT](LICENSE)
