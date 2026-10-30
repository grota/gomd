######## section -1
== rendered

  gomd

  A terminal markdown viewer with interactive element selection and vim-style
  navigation.
  Think glow 🔗 https://github.com/charmbracelet/glow but with more interactivity
  with the markdown document.

  ## Screenshots
  normal mode:
  gomd screenshot 1 📷 https://github.com/user-attachments/assets/0f0db041-
  0dc3-4d35-8ee1-d89701707aff

  --------
  jump mode:
  gomd screenshot 2 📷 https://github.com/user-attachments/assets/ba477053-
  48d4-4380-b682-8f2bf39b72b2

  --------
  interactive mode:
  gomd screenshot 3 📷 https://github.com/user-attachments/assets/d0c03603-
  662a-4092-ad15-e03c8c1f6aaa

  ## Features

  • Interactive TUI — dual-pane layout: outline sidebar + rendered content
  • Vim-style navigation —  j / k ,  gg / G ,  /  search
  • Fuzzy search — sidebar heading search uses fuzzy matching (characters in
  order)
  • Breadcrumb bar — title shows current heading path (e.g.,  gomd — file.md >
  Section > Subsection )
  • Interactive mode ( i ) — cycle through code blocks, inline code spans, and
  links; copy with  y , open links with  Enter  (works in any sub-mode)
  • Jump mode ( f ) — EasyMotion-style labels to quickly jump to any
  selectable element
  • Navigation history —  Ctrl+O  /  Ctrl+I  for back/forward heading
  navigation
  • Internal link following —  Enter  on  #anchor  links jumps to the
  referenced heading
  • Mouse support — click sidebar headings, scroll content with mouse wheel
  • Responsive sidebar — width auto-adjusts based on heading text length
  • Multiple themes —  T  to cycle (OceanDark, Dracula, Nord, Solarized, …)
  • CLI mode — list, tree, count, extract sections non-interactively
  • File watcher — auto-reloads on change
  • Editor integration —  e  opens the source file in  $EDITOR
  • Configurable link opener — set  opener  in config to customize how URLs
  are opened
  • Image rendering — inline images via Kitty graphics protocol (Ghostty,
  Kitty, WezTerm)
  • HTML tag support —  <img>  and  <a>  HTML tags are rendered as images and
  links alongside native markdown syntax

  ## Installation

  Download the latest release 🔗 https://github.com/grota/gomd/releases/latest
  from this project.

  Or via go install:

    go install github.com/grota/gomd/cmd/gomd@latest

  Or build from source:

    git clone https://github.com/grota/gomd
    cd gomd
    go build ./cmd/gomd

  Requires Go 1.22+.

  ## Usage

  ### Interactive TUI

    gomd file.md          # open a file
    gomd some/dir/        # open first .md in directory
    cat file.md | gomd    # read from stdin
    gomd                  # print help

  ### Render Mode

  Render mode ( -r  /  --render ) outputs rendered markdown to stdout without
  launching the TUI. Useful for piping, previewing, or printing.

    gomd file.md -r                        # render to stdout with background
    gomd file.md -r --disable-background   # render without background (for
  piping)
    gomd file.md -r --images               # render with inline images

  Background color is emitted by default so the output looks correct in the
  terminal. Use  --disable-background  when piping to other tools or files,
  since ANSI background sequences can interfere with downstream processing.

  Images are rendered as  alt 🖼️ path  and links as  text 🔗 url . HTML  <img>
  and  <a>  tags are converted to their markdown equivalents before rendering.

  #### Key bindings

   Key                          │ Action
  ──────────────────────────────┼─────────────────────────────────────────────
    j  /  ↓                     │ Next heading
    k  /  ↑                     │ Previous heading
    gg  /  G                    │ Jump to root / last heading
    H  /  M  /  L               │ Jump to top / mid / bottom of visible area
    zz  /  zt  /  zb            │ Center / top / bottom selection in viewport
    /                           │ Search headings (fuzzy)
    Tab                         │ Switch focus (sidebar ↔ content)
    w                           │ Toggle sidebar
    i                           │ Enter interactive mode
    f                           │ Jump mode (EasyMotion labels)
    Ctrl+O                      │ Navigate back
    Ctrl+I                      │ Navigate forward
    e                           │ Open in  $EDITOR
    T                           │ Cycle theme
    ?                           │ Help
    q  /  Ctrl+C                │ Quit
   Mouse click                  │ Select sidebar heading
   Scroll wheel                 │ Scroll content

  #### Interactive mode ( i )

  Cycles through selectable nodes in the current section: fenced code blocks,
  inline code spans, and links. Press  m  to cycle sub-modes.

   Key                │ Action
  ────────────────────┼───────────────────────────────────────────────────────
    j  /  k           │ Next / previous node
    m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)
    H  /  M  /  L     │ Jump to top / mid / bottom visible node
    zz  /  zt  /  zb  │ Center / top / bottom node in viewport
    y                 │ Copy node content to clipboard and exit
    Enter             │ Open link (any sub-mode; external: browser;  #anchor
                      │ : jump to heading)
    Esc               │ Exit interactive mode

  #### Jump mode ( f )

  Shows short labels next to all selectable elements. Type a label to jump
  directly to that element (enters interactive ALL mode). Press  Esc  to
  cancel.

  ### CLI mode

    gomd -l file.md                  # list headings
    gomd --tree file.md              # heading tree
    gomd -L 2 file.md                # only h2 headings
    gomd --filter "install" file.md  # filter by text
    gomd --count file.md             # count by level
    gomd -s "Installation" file.md   # extract section

  ## Configuration

  Config file:  ~/.config/gomd/config.toml

  All settings are optional — only specify what you want to override.

    [ui]
    theme = "OceanDark"       # OceanDark | Dracula | Nord | Gruvbox |
  TokyoNight (or a Ghostty theme name)
    compact_tree = false      # compact sidebar tree
    opener = "xdg-open"       # command to open URLs (macOS: "open")
    ghostty_theme_directory = ""  # path to Ghostty themes (e.g.,
  "/usr/share/ghostty/themes/")

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

  Key bindings use comma-separated values for multiple keys mapped to the same
  action. Key names follow bubbletea conventions (e.g.,  ctrl+d ,  shift+tab ,
  pgdown ,  esc ).

  ## Theme resolution

  gomd resolves the active theme in three steps:

  1. Built-in lookup — if  theme  matches a built-in name (OceanDark, Nord,
  Dracula, Gruvbox, TokyoNight), that theme is used.
  2. Ghostty fallback — if the name doesn't match a built-in theme and
  ghostty_theme_directory  is set, gomd looks for a file with that name in the
  directory and converts the Ghostty color palette to a gomd theme. The
  mapping is:
    •  background  /  foreground  → direct
    • palette 0 → Selected, StatusBar, Code
    • palette 1 → Search (red)
    • palette 2 → Heading2 (green)
    • palette 3 → Heading1 (yellow)
    • palette 4 → Heading3 (blue)
    • palette 6 → NodeSel (cyan)
    • palette 8 → Border, HeadingN (bright black)
    • palette 11 → Highlight (bright yellow)
  3. Color overrides — any non-empty field in  [ui.theme_override]  replaces
  the corresponding color, regardless of whether the base theme is built-in or
  loaded from Ghostty.

  If the theme name doesn't match a built-in and no Ghostty directory is
  configured (or the file isn't found), gomd falls back to OceanDark.

  ## Input resolution

  When no file argument is given,  gomd  reads from stdin if piped, otherwise
  prints help and exits.

  ### Images

  gomd can render inline images using the Kitty graphics protocol
  https://sw.kovidgoyal.net/kitty/graphics-protocol/. This works in terminals
  that support it: Ghostty, Kitty, and WezTerm.

  Image rendering is controlled by three settings, evaluated in this order:

  1.  --no-images  flag: always disables images (highest priority).
  2.  --images  flag: always enables images, even if the terminal can't be
  auto-detected (e.g. inside tmux).
  3. Config file ( ~/.config/gomd/config.toml ): the  [images]  section sets
  the default.

    [images]
    enabled = true   # default

  When none of the flags are passed,  enabled  from the config is used. If
  enabled = true , gomd auto-detects whether the terminal supports Kitty
  graphics (by checking  TERM_PROGRAM  and related env vars). If detection
  succeeds, images are rendered automatically.

  tmux: inside tmux, terminal detection fails because  TERM_PROGRAM  is  tmux
  . Use  --images  to force image rendering. Images are sent via tmux's DCS
  passthrough (requires  allow-passthrough on  in your tmux config). Note:
  images rendered inside tmux are "sticky" — they remain painted on screen
  even when scrolling or switching tmux windows, because tmux manages text
  cells but not graphics placements. Run  reset  in the terminal to clear
  them. This is a known limitation of tmux's passthrough mechanism.

  Only local image files are supported (relative or absolute paths). URLs are
  not fetched — they display as text placeholders.

  ## License

  MIT
== nodes
link   src 8-8 col 6-51  shown 5-5 col 8-12  "glow -> https://github.com/charmbracelet/glow"
link   src 16-16 col 0-144  shown 10-11 col 2-75  "gomd screenshot 1 📷 https://github.com/user-attachments/assets/0f0db041-0dc3-4d35-8ee1-d89701707aff -> https://github.com/user-attachments/assets/0f0db041-0dc3-4d35-8ee1-d89701707aff"
link   src 24-24 col 0-144  shown 15-16 col 2-75  "gomd screenshot 2 📷 https://github.com/user-attachments/assets/ba477053-48d4-4380-b682-8f2bf39b72b2 -> https://github.com/user-attachments/assets/ba477053-48d4-4380-b682-8f2bf39b72b2"
link   src 32-32 col 0-144  shown 20-21 col 2-75  "gomd screenshot 3 📷 https://github.com/user-attachments/assets/d0c03603-662a-4092-ad15-e03c8c1f6aaa -> https://github.com/user-attachments/assets/d0c03603-662a-4092-ad15-e03c8c1f6aaa"
inline src 37-37 col 31-34  shown 26-26 col 28-29  "j"
inline src 37-37 col 35-38  shown 26-26 col 32-33  "k"
inline src 37-37 col 40-44  shown 26-26 col 37-39  "gg"
inline src 37-37 col 45-48  shown 26-26 col 42-43  "G"
inline src 37-37 col 50-53  shown 26-26 col 47-48  "/"
inline src 39-39 col 65-106  shown 29-30 col 62-78  "gomd — file.md > Section > Subsection"
inline src 40-40 col 24-27  shown 31-31 col 23-24  "i"
inline src 40-40 col 100-103  shown 32-32 col 20-21  "y"
inline src 40-40 col 121-128  shown 32-32 col 41-46  "Enter"
inline src 41-41 col 17-20  shown 33-33 col 16-17  "f"
inline src 42-42 col 29-37  shown 35-35 col 26-32  "Ctrl+O"
inline src 42-42 col 40-48  shown 35-35 col 37-43  "Ctrl+I"
inline src 43-43 col 34-41  shown 37-37 col 31-36  "Enter"
inline src 43-43 col 45-54  shown 37-37 col 42-49  "#anchor"
inline src 46-46 col 26-29  shown 41-41 col 23-24  "T"
inline src 49-49 col 29-32  shown 44-44 col 26-27  "e"
inline src 49-49 col 58-67  shown 44-44 col 55-62  "$EDITOR"
inline src 50-50 col 39-47  shown 45-45 col 36-42  "opener"
inline src 52-52 col 27-34  shown 49-49 col 24-29  "<img>"
inline src 52-52 col 39-44  shown 49-49 col 36-39  "<a>"
link   src 56-56 col 13-76  shown 54-54 col 15-29  "latest release -> https://github.com/grota/gomd/releases/latest"
block  src 60-62 col 0-0  shown 59-59 col -1--1  "go install github.com/grota/gomd/cmd/gomd@latest"
block  src 66-70 col 0-0  shown 63-65 col -1--1  "git clone https://github.com/grota/gomd\ncd gomd\ngo build ./cmd/gomd"
block  src 78-83 col 0-0  shown 73-76 col -1--1  "gomd file.md          # open a file\ngomd some/dir/        # open first .md in directory\ncat file.md | gomd    # read from stdin\ngomd                  # print help"
inline src 87-87 col 13-17  shown 80-80 col 16-18  "-r"
inline src 87-87 col 20-30  shown 80-80 col 23-31  "--render"
block  src 89-93 col 0-0  shown 83-86 col -1--1  "gomd file.md -r                        # render to stdout with background\ngomd file.md -r --disable-background   # render without background (for piping)\ngomd file.md -r --images               # render with inline images"
inline src 95-95 col 88-110  shown 89-89 col 17-37  "--disable-background"
inline src 97-97 col 23-41  shown 92-92 col 26-37  "alt 🖼️ path"
inline src 97-97 col 55-70  shown 92-92 col 53-64  "text 🔗 url"
inline src 97-97 col 77-84  shown 92-92 col 73-78  "<img>"
inline src 97-97 col 89-94  shown 93-93 col 7-10  "<a>"
inline src 103-103 col 2-5  shown 99-99 col 4-5  "j"
inline src 103-103 col 8-13  shown 99-99 col 10-11  "↓"
inline src 104-104 col 2-5  shown 100-100 col 4-5  "k"
inline src 104-104 col 8-13  shown 100-100 col 10-11  "↑"
inline src 105-105 col 2-6  shown 101-101 col 4-6  "gg"
inline src 105-105 col 9-12  shown 101-101 col 11-12  "G"
inline src 106-106 col 2-5  shown 102-102 col 4-5  "H"
inline src 106-106 col 8-11  shown 102-102 col 10-11  "M"
inline src 106-106 col 14-17  shown 102-102 col 16-17  "L"
inline src 107-107 col 2-6  shown 103-103 col 4-6  "zz"
inline src 107-107 col 9-13  shown 103-103 col 11-13  "zt"
inline src 107-107 col 16-20  shown 103-103 col 18-20  "zb"
inline src 108-108 col 2-5  shown 104-104 col 4-5  "/"
inline src 109-109 col 2-7  shown 105-105 col 4-7  "Tab"
inline src 110-110 col 2-5  shown 106-106 col 4-5  "w"
inline src 111-111 col 2-5  shown 107-107 col 4-5  "i"
inline src 112-112 col 2-5  shown 108-108 col 4-5  "f"
inline src 113-113 col 2-10  shown 109-109 col 4-10  "Ctrl+O"
inline src 114-114 col 2-10  shown 110-110 col 4-10  "Ctrl+I"
inline src 115-115 col 2-5  shown 111-111 col 4-5  "e"
inline src 115-115 col 31-40  shown 111-111 col 43-50  "$EDITOR"
inline src 116-116 col 2-5  shown 112-112 col 4-5  "T"
inline src 117-117 col 2-5  shown 113-113 col 4-5  "?"
inline src 118-118 col 2-5  shown 114-114 col 4-5  "q"
inline src 118-118 col 8-16  shown 114-114 col 10-16  "Ctrl+C"
inline src 122-122 col 23-26  shown 118-118 col 26-27  "i"
inline src 124-124 col 112-115  shown 121-121 col 39-40  "m"
inline src 128-128 col 2-5  shown 125-125 col 4-5  "j"
inline src 128-128 col 8-11  shown 125-125 col 10-11  "k"
inline src 129-129 col 2-5  shown 126-126 col 4-5  "m"
inline src 130-130 col 2-5  shown 127-127 col 4-5  "H"
inline src 130-130 col 8-11  shown 127-127 col 10-11  "M"
inline src 130-130 col 14-17  shown 127-127 col 16-17  "L"
inline src 131-131 col 2-6  shown 128-128 col 4-6  "zz"
inline src 131-131 col 9-13  shown 128-128 col 11-13  "zt"
inline src 131-131 col 16-20  shown 128-128 col 18-20  "zb"
inline src 132-132 col 2-5  shown 129-129 col 4-5  "y"
inline src 133-133 col 2-9  shown 130-130 col 4-9  "Enter"
inline src 133-133 col 67-76  shown 130-130 col 69-76  "#anchor"
inline src 134-134 col 2-7  shown 132-132 col 4-7  "Esc"
inline src 136-136 col 16-19  shown 134-134 col 19-20  "f"
inline src 138-138 col 135-140  shown 137-137 col 65-68  "Esc"
block  src 142-149 col 0-0  shown 142-147 col -1--1  "gomd -l file.md                  # list headings\ngomd --tree file.md              # heading tree\ngomd -L 2 file.md                # only h2 headings\ngomd --filter \"install\" file.md  # filter by text\ngomd --count file.md             # count by level\ngomd -s \"Installation\" file.md   # extract section"
inline src 153-153 col 13-41  shown 151-151 col 16-42  "~/.config/gomd/config.toml"
block  src 157-230 col 0-0  shown 155-228 col -1--1  "[ui]\ntheme = \"OceanDark\"       # OceanDark | Dracula | Nord | Gruvbox | TokyoNight (or a Ghostty theme name)\ncompact_tree = false      # compact sidebar tree\nopener = \"xdg-open\"       # command to open URLs (macOS: \"open\")\nghostty_theme_directory = \"\"  # path to Ghostty themes (e.g., \"/usr/share/ghostty/themes/\")\n\n[ui.theme_override]\n# Override individual colors of the active theme (hex values).\n# Any field left empty keeps the theme default.\n# border = \"#504945\"\n# selected = \"#3c3836\"\n# heading1 = \"#fabd2f\"\n# heading2 = \"#b8bb26\"\n# heading3 = \"#83a598\"\n# headingn = \"#928374\"\n# background = \"#282828\"\n# foreground = \"#ebdbb2\"\n# statusbar = \"#3c3836\"\n# highlight = \"#fe8019\"\n# code = \"#1d2021\"\n# search = \"#fb4934\"\n# nodesel = \"#8ec07c\"\n\n[terminal]\ncolor_mode = \"auto\"       # auto | truecolor | 256 | ansi | none\n\n[images]\nenabled = true\n\n[content]\nsyntax_highlighting = true\n\n[keys]\n# Shared\nquit = \"q\"\nhelp = \"?\"\ntheme_picker = \"T\"\ntoggle_focus = \"tab\"\ntoggle_sidebar = \"w\"\nreload = \"r\"\nedit = \"e\"\n\n# Sidebar navigation\nsidebar_down = \"j,down\"\nsidebar_up = \"k,up\"\nsidebar_top = \"g\"\nsidebar_bottom = \"G\"\nsidebar_search = \"/\"\nnext_match = \"n\"\nprev_match = \"N\"\n\n# Content navigation\nscroll_down = \"j,down\"\nscroll_up = \"k,up\"\nscroll_half_down = \"ctrl+d,ctrl+f,pgdown, \"\nscroll_half_up = \"ctrl+u,ctrl+b,pgup\"\ncontent_top = \"g\"\ncontent_bottom = \"G\"\ncontent_search = \"/\"\ncontent_next_match = \"n\"\ncontent_prev_match = \"N\"\njump = \"f\"\nnav_back = \"ctrl+o\"\nnav_forward = \"ctrl+i\"\n\n# Node-select mode\nnode_select = \"i\"\nnode_next = \"j,down,tab\"\nnode_prev = \"k,up,shift+tab\"\nnode_copy = \"y\"\nnode_open = \"enter\"\nnode_exit = \"esc,q,i\""
inline src 232-232 col 131-139  shown 231-231 col 57-63  "ctrl+d"
inline src 232-232 col 141-152  shown 231-231 col 67-76  "shift+tab"
inline src 232-232 col 154-162  shown 232-232 col 2-8  "pgdown"
inline src 232-232 col 164-169  shown 232-232 col 12-15  "esc"
inline src 238-238 col 30-37  shown 238-238 col 27-32  "theme"
inline src 239-239 col 75-100  shown 241-241 col 2-25  "ghostty_theme_directory"
inline src 240-240 col 5-17  shown 244-244 col 7-17  "background"
inline src 240-240 col 20-32  shown 244-244 col 22-32  "foreground"
inline src 249-249 col 50-71  shown 253-253 col 47-66  "[ui.theme_override]"
inline src 255-255 col 32-38  shown 262-262 col 35-39  "gomd"
link   src 259-259 col 40-117  shown 267-267 col 42-65  "Kitty graphics protocol -> https://sw.kovidgoyal.net/kitty/graphics-protocol/"
inline src 263-263 col 5-18  shown 273-273 col 6-17  "--no-images"
inline src 264-264 col 5-15  shown 274-274 col 6-14  "--images"
inline src 265-265 col 20-48  shown 276-276 col 19-45  "~/.config/gomd/config.toml"
inline src 265-265 col 55-65  shown 276-276 col 54-62  "[images]"
block  src 267-270 col 0-0  shown 279-280 col -1--1  "[images]\nenabled = true   # default"
inline src 272-272 col 35-44  shown 282-282 col 38-45  "enabled"
inline src 272-272 col 73-89  shown 283-283 col 2-16  "enabled = true"
inline src 272-272 col 167-181  shown 284-284 col 25-37  "TERM_PROGRAM"
inline src 274-274 col 56-70  shown 287-287 col 55-67  "TERM_PROGRAM"
inline src 274-274 col 74-80  shown 287-287 col 73-77  "tmux"
inline src 274-274 col 86-96  shown 288-288 col 9-17  "--images"
inline src 274-274 col 176-198  shown 289-289 col 25-45  "allow-passthrough on"
inline src 274-274 col 414-421  shown 292-292 col 42-47  "reset"
== copy per line
1: "gomd" | ""
3: "A terminal markdown viewer with interactive element selection and vim-style\nnavigation." | "<div align=\"center\">\n  <h1>gomd</h1>\n</div>\nA terminal markdown viewer with interactive element selection and vim-style navigation."
4: "A terminal markdown viewer with interactive element selection and vim-style\nnavigation." | "<div align=\"center\">\n  <h1>gomd</h1>\n</div>\nA terminal markdown viewer with interactive element selection and vim-style navigation."
5: "Think glow 🔗 https://github.com/charmbracelet/glow but with more interactivity" | "Think [glow](https://github.com/charmbracelet/glow) but with more interactivity"
6: "with the markdown document." | "with the markdown document."
8: "## Screenshots" | "## Screenshots"
9: "normal mode:" | "<div align=\"center\">\nnormal mode:\n</div>"
10: "gomd screenshot 1 📷 https://github.com/user-attachments/assets/0f0db041-\n0dc3-4d35-8ee1-d89701707aff" | "<img width=\"1536\" height=\"1022\" alt=\"gomd screenshot 1\" src=\"https://github.com/user-attachments/assets/0f0db041-0dc3-4d35-8ee1-d89701707aff\" />"
11: "gomd screenshot 1 📷 https://github.com/user-attachments/assets/0f0db041-\n0dc3-4d35-8ee1-d89701707aff" | "<img width=\"1536\" height=\"1022\" alt=\"gomd screenshot 1\" src=\"https://github.com/user-attachments/assets/0f0db041-0dc3-4d35-8ee1-d89701707aff\" />"
13: "--------" | ""
14: "jump mode:" | "<div align=\"center\">\njump mode:\n</div>"
15: "gomd screenshot 2 📷 https://github.com/user-attachments/assets/ba477053-\n48d4-4380-b682-8f2bf39b72b2" | "<img width=\"1536\" height=\"1020\" alt=\"gomd screenshot 2\" src=\"https://github.com/user-attachments/assets/ba477053-48d4-4380-b682-8f2bf39b72b2\" />"
16: "gomd screenshot 2 📷 https://github.com/user-attachments/assets/ba477053-\n48d4-4380-b682-8f2bf39b72b2" | "<img width=\"1536\" height=\"1020\" alt=\"gomd screenshot 2\" src=\"https://github.com/user-attachments/assets/ba477053-48d4-4380-b682-8f2bf39b72b2\" />"
18: "--------" | ""
19: "interactive mode:" | "<div align=\"center\">\ninteractive mode:\n</div>"
20: "gomd screenshot 3 📷 https://github.com/user-attachments/assets/d0c03603-\n662a-4092-ad15-e03c8c1f6aaa" | "<img width=\"1536\" height=\"1020\" alt=\"gomd screenshot 3\" src=\"https://github.com/user-attachments/assets/d0c03603-662a-4092-ad15-e03c8c1f6aaa\" />"
21: "gomd screenshot 3 📷 https://github.com/user-attachments/assets/d0c03603-\n662a-4092-ad15-e03c8c1f6aaa" | "<img width=\"1536\" height=\"1020\" alt=\"gomd screenshot 3\" src=\"https://github.com/user-attachments/assets/d0c03603-662a-4092-ad15-e03c8c1f6aaa\" />"
23: "## Features" | "## Features"
25: "• Interactive TUI — dual-pane layout: outline sidebar + rendered content" | "- **Interactive TUI** — dual-pane layout: outline sidebar + rendered content"
26: "• Vim-style navigation —  j / k ,  gg / G ,  /  search" | "- **Vim-style navigation** — `j`/`k`, `gg`/`G`, `/` search"
27: "• Fuzzy search — sidebar heading search uses fuzzy matching (characters in" | "- **Fuzzy search** — sidebar heading search uses fuzzy matching (characters in"
28: "order)" | "order)"
29: "• Breadcrumb bar — title shows current heading path (e.g.,  gomd — file.md >\nSection > Subsection" | "- **Breadcrumb bar** — title shows current heading path (e.g., `gomd — file.md > Section > Subsection`"
30: "gomd — file.md >\nSection > Subsection )" | "`gomd — file.md > Section > Subsection`)"
31: "• Interactive mode ( i ) — cycle through code blocks, inline code spans, and" | "- **Interactive mode** (`i`) — cycle through code blocks, inline code spans, and"
32: "links; copy with  y , open links with  Enter  (works in any sub-mode)" | "links; copy with `y`, open links with `Enter` (works in any sub-mode)"
33: "• Jump mode ( f ) — EasyMotion-style labels to quickly jump to any" | "- **Jump mode** (`f`) — EasyMotion-style labels to quickly jump to any"
34: "selectable element" | "selectable element"
35: "• Navigation history —  Ctrl+O  /  Ctrl+I  for back/forward heading" | "- **Navigation history** — `Ctrl+O` / `Ctrl+I` for back/forward heading"
36: "navigation" | "navigation"
37: "• Internal link following —  Enter  on  #anchor  links jumps to the" | "- **Internal link following** — `Enter` on `#anchor` links jumps to the"
38: "referenced heading" | "referenced heading"
39: "• Mouse support — click sidebar headings, scroll content with mouse wheel" | "- **Mouse support** — click sidebar headings, scroll content with mouse wheel"
40: "• Responsive sidebar — width auto-adjusts based on heading text length" | "- **Responsive sidebar** — width auto-adjusts based on heading text length"
41: "• Multiple themes —  T  to cycle (OceanDark, Dracula, Nord, Solarized, …)" | "- **Multiple themes** — `T` to cycle (OceanDark, Dracula, Nord, Solarized, …)"
42: "• CLI mode — list, tree, count, extract sections non-interactively" | "- **CLI mode** — list, tree, count, extract sections non-interactively"
43: "• File watcher — auto-reloads on change" | "- **File watcher** — auto-reloads on change"
44: "• Editor integration —  e  opens the source file in  $EDITOR" | "- **Editor integration** — `e` opens the source file in `$EDITOR`"
45: "• Configurable link opener — set  opener  in config to customize how URLs" | "- **Configurable link opener** — set `opener` in config to customize how URLs"
46: "are opened" | "are opened"
47: "• Image rendering — inline images via Kitty graphics protocol (Ghostty," | "- **Image rendering** — inline images via Kitty graphics protocol (Ghostty,"
48: "Kitty, WezTerm)" | "Kitty, WezTerm)"
49: "• HTML tag support —  <img>  and  <a>  HTML tags are rendered as images and" | "- **HTML tag support** — `<img>` and `<a>` HTML tags are rendered as images and"
50: "links alongside native markdown syntax" | "links alongside native markdown syntax"
52: "## Installation" | "## Installation"
54: "Download the latest release 🔗 https://github.com/grota/gomd/releases/latest" | "Download the [latest release](https://github.com/grota/gomd/releases/latest)"
55: "from this project." | "from this project."
57: "Or via go install:" | "Or via go install:"
59: "go install github.com/grota/gomd/cmd/gomd@latest" | "```bash\ngo install github.com/grota/gomd/cmd/gomd@latest\n```"
61: "Or build from source:" | "Or build from source:"
63: "git clone https://github.com/grota/gomd" | "git clone https://github.com/grota/gomd"
64: "cd gomd" | "cd gomd"
65: "go build ./cmd/gomd" | "go build ./cmd/gomd"
67: "Requires Go 1.22+." | "Requires Go 1.22+."
69: "## Usage" | "## Usage"
71: "### Interactive TUI" | "### Interactive TUI"
73: "gomd file.md          # open a file" | "gomd file.md          # open a file"
74: "gomd some/dir/        # open first .md in directory" | "gomd some/dir/        # open first .md in directory"
75: "cat file.md | gomd    # read from stdin" | "cat file.md | gomd    # read from stdin"
76: "gomd                  # print help" | "gomd                  # print help"
78: "### Render Mode" | "### Render Mode"
80: "Render mode ( -r  /  --render ) outputs rendered markdown to stdout without" | "Render mode (`-r` / `--render`) outputs rendered markdown to stdout without"
81: "launching the TUI. Useful for piping, previewing, or printing." | "launching the TUI. Useful for piping, previewing, or printing."
83: "gomd file.md -r                        # render to stdout with background" | "gomd file.md -r                        # render to stdout with background"
84: "gomd file.md -r --disable-background   # render without background (for" | "gomd file.md -r --disable-background   # render without background (for"
85: "piping)" | "piping)"
86: "gomd file.md -r --images               # render with inline images" | "gomd file.md -r --images               # render with inline images"
88: "Background color is emitted by default so the output looks correct in the" | "Background color is emitted by default so the output looks correct in the"
89: "terminal. Use  --disable-background  when piping to other tools or files," | "terminal. Use `--disable-background` when piping to other tools or files,"
90: "since ANSI background sequences can interfere with downstream processing." | "since ANSI background sequences can interfere with downstream processing."
92: "Images are rendered as  alt 🖼️ path  and links as  text 🔗 url . HTML  <img>" | "Images are rendered as `alt 🖼️ path` and links as `text 🔗 url`. HTML `<img>`"
93: "and  <a>  tags are converted to their markdown equivalents before rendering." | "and `<a>` tags are converted to their markdown equivalents before rendering."
95: "#### Key bindings" | "#### Key bindings"
97: " Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
98: "──────────────────────────────┼─────────────────────────────────────────────" | ""
99: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
100: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
101: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
102: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
103: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
104: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
105: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
106: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
107: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
108: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
109: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
110: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
111: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
112: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
113: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
114: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
115: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
116: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
118: "#### Interactive mode ( i )" | "#### Interactive mode (`i`)"
120: "Cycles through selectable nodes in the current section: fenced code blocks," | "Cycles through selectable nodes in the current section: fenced code blocks,"
121: "inline code spans, and links. Press  m  to cycle sub-modes." | "inline code spans, and links. Press `m` to cycle sub-modes."
123: " Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
124: "────────────────────┼───────────────────────────────────────────────────────" | ""
125: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
126: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
127: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
128: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
129: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
130: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
131: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
132: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
134: "#### Jump mode ( f )" | "#### Jump mode (`f`)"
136: "Shows short labels next to all selectable elements. Type a label to jump" | "Shows short labels next to all selectable elements. Type a label to jump"
137: "directly to that element (enters interactive ALL mode). Press  Esc  to" | "directly to that element (enters interactive ALL mode). Press `Esc` to"
138: "cancel." | "cancel."
140: "### CLI mode" | "### CLI mode"
142: "gomd -l file.md                  # list headings" | "gomd -l file.md                  # list headings"
143: "gomd --tree file.md              # heading tree" | "gomd --tree file.md              # heading tree"
144: "gomd -L 2 file.md                # only h2 headings" | "gomd -L 2 file.md                # only h2 headings"
145: "gomd --filter \"install\" file.md  # filter by text" | "gomd --filter \"install\" file.md  # filter by text"
146: "gomd --count file.md             # count by level" | "gomd --count file.md             # count by level"
147: "gomd -s \"Installation\" file.md   # extract section" | "gomd -s \"Installation\" file.md   # extract section"
149: "## Configuration" | "## Configuration"
151: "Config file:  ~/.config/gomd/config.toml" | "Config file: `~/.config/gomd/config.toml`"
153: "All settings are optional — only specify what you want to override." | "All settings are optional — only specify what you want to override."
155: "[ui]" | "[ui]"
156: "theme = \"OceanDark\"       # OceanDark | Dracula | Nord | Gruvbox |" | "theme = \"OceanDark\"       # OceanDark | Dracula | Nord | Gruvbox |"
157: "TokyoNight (or a Ghostty theme name)" | "TokyoNight (or a Ghostty theme name)"
158: "compact_tree = false      # compact sidebar tree" | "compact_tree = false      # compact sidebar tree"
159: "opener = \"xdg-open\"       # command to open URLs (macOS: \"open\")" | "opener = \"xdg-open\"       # command to open URLs (macOS: \"open\")"
160: "ghostty_theme_directory = \"\"  # path to Ghostty themes (e.g.," | "ghostty_theme_directory = \"\"  # path to Ghostty themes (e.g.,"
161: "\"/usr/share/ghostty/themes/\")" | "\"/usr/share/ghostty/themes/\")"
163: "[ui.theme_override]" | "[ui.theme_override]"
164: "# Override individual colors of the active theme (hex values)." | "# Override individual colors of the active theme (hex values)."
165: "# Any field left empty keeps the theme default." | "# Any field left empty keeps the theme default."
166: "# border = \"#504945\"" | "# border = \"#504945\""
167: "# selected = \"#3c3836\"" | "# selected = \"#3c3836\""
168: "# heading1 = \"#fabd2f\"" | "# heading1 = \"#fabd2f\""
169: "# heading2 = \"#b8bb26\"" | "# heading2 = \"#b8bb26\""
170: "# heading3 = \"#83a598\"" | "# heading3 = \"#83a598\""
171: "# headingn = \"#928374\"" | "# headingn = \"#928374\""
172: "# background = \"#282828\"" | "# background = \"#282828\""
173: "# foreground = \"#ebdbb2\"" | "# foreground = \"#ebdbb2\""
174: "# statusbar = \"#3c3836\"" | "# statusbar = \"#3c3836\""
175: "# highlight = \"#fe8019\"" | "# highlight = \"#fe8019\""
176: "# code = \"#1d2021\"" | "# code = \"#1d2021\""
177: "# search = \"#fb4934\"" | "# search = \"#fb4934\""
178: "# nodesel = \"#8ec07c\"" | "# nodesel = \"#8ec07c\""
180: "[terminal]" | "[terminal]"
181: "color_mode = \"auto\"       # auto | truecolor | 256 | ansi | none" | "color_mode = \"auto\"       # auto | truecolor | 256 | ansi | none"
183: "[images]" | "[images]"
184: "enabled = true" | "enabled = true"
186: "[content]" | "[content]"
187: "syntax_highlighting = true" | "syntax_highlighting = true"
189: "[keys]" | "[keys]"
190: "# Shared" | "# Shared"
191: "quit = \"q\"" | "quit = \"q\""
192: "help = \"?\"" | "help = \"?\""
193: "theme_picker = \"T\"" | "theme_picker = \"T\""
194: "toggle_focus = \"tab\"" | "toggle_focus = \"tab\""
195: "toggle_sidebar = \"w\"" | "toggle_sidebar = \"w\""
196: "reload = \"r\"" | "reload = \"r\""
197: "edit = \"e\"" | "edit = \"e\""
199: "# Sidebar navigation" | "# Sidebar navigation"
200: "sidebar_down = \"j,down\"" | "sidebar_down = \"j,down\""
201: "sidebar_up = \"k,up\"" | "sidebar_up = \"k,up\""
202: "sidebar_top = \"g\"" | "sidebar_top = \"g\""
203: "sidebar_bottom = \"G\"" | "sidebar_bottom = \"G\""
204: "sidebar_search = \"/\"" | "sidebar_search = \"/\""
205: "next_match = \"n\"" | "next_match = \"n\""
206: "prev_match = \"N\"" | "prev_match = \"N\""
208: "# Content navigation" | "# Content navigation"
209: "scroll_down = \"j,down\"" | "scroll_down = \"j,down\""
210: "scroll_up = \"k,up\"" | "scroll_up = \"k,up\""
211: "scroll_half_down = \"ctrl+d,ctrl+f,pgdown, \"" | "scroll_half_down = \"ctrl+d,ctrl+f,pgdown, \""
212: "scroll_half_up = \"ctrl+u,ctrl+b,pgup\"" | "scroll_half_up = \"ctrl+u,ctrl+b,pgup\""
213: "content_top = \"g\"" | "content_top = \"g\""
214: "content_bottom = \"G\"" | "content_bottom = \"G\""
215: "content_search = \"/\"" | "content_search = \"/\""
216: "content_next_match = \"n\"" | "content_next_match = \"n\""
217: "content_prev_match = \"N\"" | "content_prev_match = \"N\""
218: "jump = \"f\"" | "jump = \"f\""
219: "nav_back = \"ctrl+o\"" | "nav_back = \"ctrl+o\""
220: "nav_forward = \"ctrl+i\"" | "nav_forward = \"ctrl+i\""
222: "# Node-select mode" | "# Node-select mode"
223: "node_select = \"i\"" | "node_select = \"i\""
224: "node_next = \"j,down,tab\"" | "node_next = \"j,down,tab\""
225: "node_prev = \"k,up,shift+tab\"" | "node_prev = \"k,up,shift+tab\""
226: "node_copy = \"y\"" | "node_copy = \"y\""
227: "node_open = \"enter\"" | "node_open = \"enter\""
228: "node_exit = \"esc,q,i\"" | "node_exit = \"esc,q,i\""
230: "Key bindings use comma-separated values for multiple keys mapped to the same" | "Key bindings use comma-separated values for multiple keys mapped to the same"
231: "action. Key names follow bubbletea conventions (e.g.,  ctrl+d ,  shift+tab ," | "action. Key names follow bubbletea conventions (e.g., `ctrl+d`, `shift+tab`,"
232: "pgdown ,  esc )." | "`pgdown`, `esc`)."
234: "## Theme resolution" | "## Theme resolution"
236: "gomd resolves the active theme in three steps:" | "gomd resolves the active theme in three steps:"
238: "1. Built-in lookup — if  theme  matches a built-in name (OceanDark, Nord," | "1. **Built-in lookup** — if `theme` matches a built-in name (OceanDark, Nord,"
239: "Dracula, Gruvbox, TokyoNight), that theme is used." | "Dracula, Gruvbox, TokyoNight), that theme is used."
240: "2. Ghostty fallback — if the name doesn't match a built-in theme and" | "2. **Ghostty fallback** — if the name doesn't match a built-in theme and"
241: "ghostty_theme_directory  is set, gomd looks for a file with that name in the" | "`ghostty_theme_directory` is set, gomd looks for a file with that name in the"
242: "directory and converts the Ghostty color palette to a gomd theme. The" | "directory and converts the Ghostty color palette to a gomd theme. The"
243: "mapping is:" | "mapping is:"
244: "•  background  /  foreground  → direct" | "   - `background` / `foreground` → direct"
245: "• palette 0 → Selected, StatusBar, Code" | "   - palette 0 → Selected, StatusBar, Code"
246: "• palette 1 → Search (red)" | "   - palette 1 → Search (red)"
247: "• palette 2 → Heading2 (green)" | "   - palette 2 → Heading2 (green)"
248: "• palette 3 → Heading1 (yellow)" | "   - palette 3 → Heading1 (yellow)"
249: "• palette 4 → Heading3 (blue)" | "   - palette 4 → Heading3 (blue)"
250: "• palette 6 → NodeSel (cyan)" | "   - palette 6 → NodeSel (cyan)"
251: "• palette 8 → Border, HeadingN (bright black)" | "   - palette 8 → Border, HeadingN (bright black)"
252: "• palette 11 → Highlight (bright yellow)" | "   - palette 11 → Highlight (bright yellow)"
253: "3. Color overrides — any non-empty field in  [ui.theme_override]  replaces" | "3. **Color overrides** — any non-empty field in `[ui.theme_override]` replaces"
254: "the corresponding color, regardless of whether the base theme is built-in or" | "the corresponding color, regardless of whether the base theme is built-in or"
255: "loaded from Ghostty." | "loaded from Ghostty."
257: "If the theme name doesn't match a built-in and no Ghostty directory is" | "If the theme name doesn't match a built-in and no Ghostty directory is"
258: "configured (or the file isn't found), gomd falls back to OceanDark." | "configured (or the file isn't found), gomd falls back to OceanDark."
260: "## Input resolution" | "## Input resolution"
262: "When no file argument is given,  gomd  reads from stdin if piped, otherwise" | "When no file argument is given, `gomd` reads from stdin if piped, otherwise"
263: "prints help and exits." | "prints help and exits."
265: "### Images" | "### Images"
267: "gomd can render inline images using the Kitty graphics protocol\nhttps://sw.kovidgoyal.net/kitty/graphics-protocol/" | "gomd can render inline images using the [Kitty graphics protocol](https://sw.kovidgoyal.net/kitty/graphics-protocol/)"
268: "Kitty graphics protocol\nhttps://sw.kovidgoyal.net/kitty/graphics-protocol/. This works in terminals" | "[Kitty graphics protocol](https://sw.kovidgoyal.net/kitty/graphics-protocol/). This works in terminals"
269: "that support it: Ghostty, Kitty, and WezTerm." | "that support it: **Ghostty**, **Kitty**, and **WezTerm**."
271: "Image rendering is controlled by three settings, evaluated in this order:" | "Image rendering is controlled by three settings, evaluated in this order:"
273: "1.  --no-images  flag: always disables images (highest priority)." | "1. **`--no-images`** flag: always disables images (highest priority)."
274: "2.  --images  flag: always enables images, even if the terminal can't be" | "2. **`--images`** flag: always enables images, even if the terminal can't be"
275: "auto-detected (e.g. inside tmux)." | "auto-detected (e.g. inside tmux)."
276: "3. Config file ( ~/.config/gomd/config.toml ): the  [images]  section sets" | "3. **Config file** (`~/.config/gomd/config.toml`): the `[images]` section sets"
277: "the default." | "the default."
279: "[images]" | "[images]"
280: "enabled = true   # default" | "enabled = true   # default"
282: "When none of the flags are passed,  enabled  from the config is used. If" | "When none of the flags are passed, `enabled` from the config is used. If"
283: "enabled = true , gomd auto-detects whether the terminal supports Kitty" | "`enabled = true`, gomd auto-detects whether the terminal supports Kitty"
284: "graphics (by checking  TERM_PROGRAM  and related env vars). If detection" | "graphics (by checking `TERM_PROGRAM` and related env vars). If detection"
285: "succeeds, images are rendered automatically." | "succeeds, images are rendered automatically."
287: "tmux: inside tmux, terminal detection fails because  TERM_PROGRAM  is  tmux" | "**tmux**: inside tmux, terminal detection fails because `TERM_PROGRAM` is `tmux`"
288: ". Use  --images  to force image rendering. Images are sent via tmux's DCS" | ". Use `--images` to force image rendering. Images are sent via tmux's DCS"
289: "passthrough (requires  allow-passthrough on  in your tmux config). Note:" | "passthrough (requires `allow-passthrough on` in your tmux config). Note:"
290: "images rendered inside tmux are \"sticky\" — they remain painted on screen" | "images rendered inside tmux are \"sticky\" — they remain painted on screen"
291: "even when scrolling or switching tmux windows, because tmux manages text" | "even when scrolling or switching tmux windows, because tmux manages text"
292: "cells but not graphics placements. Run  reset  in the terminal to clear" | "cells but not graphics placements. Run `reset` in the terminal to clear"
293: "them. This is a known limitation of tmux's passthrough mechanism." | "them. This is a known limitation of tmux's passthrough mechanism."
295: "Only local image files are supported (relative or absolute paths). URLs are" | "Only local image files are supported (relative or absolute paths). URLs are"
296: "not fetched — they display as text placeholders." | "not fetched — they display as text placeholders."
298: "## License" | "## License"
300: "MIT" | "MIT"
######## section 0
== rendered

  ## Screenshots
  normal mode:
  gomd screenshot 1 📷 https://github.com/user-attachments/assets/0f0db041-
  0dc3-4d35-8ee1-d89701707aff

  --------
  jump mode:
  gomd screenshot 2 📷 https://github.com/user-attachments/assets/ba477053-
  48d4-4380-b682-8f2bf39b72b2

  --------
  interactive mode:
  gomd screenshot 3 📷 https://github.com/user-attachments/assets/d0c03603-
  662a-4092-ad15-e03c8c1f6aaa
== nodes
link   src 6-6 col 0-144  shown 3-4 col 2-75  "gomd screenshot 1 📷 https://github.com/user-attachments/assets/0f0db041-0dc3-4d35-8ee1-d89701707aff -> https://github.com/user-attachments/assets/0f0db041-0dc3-4d35-8ee1-d89701707aff"
link   src 14-14 col 0-144  shown 8-9 col 2-75  "gomd screenshot 2 📷 https://github.com/user-attachments/assets/ba477053-48d4-4380-b682-8f2bf39b72b2 -> https://github.com/user-attachments/assets/ba477053-48d4-4380-b682-8f2bf39b72b2"
link   src 22-22 col 0-144  shown 13-14 col 2-75  "gomd screenshot 3 📷 https://github.com/user-attachments/assets/d0c03603-662a-4092-ad15-e03c8c1f6aaa -> https://github.com/user-attachments/assets/d0c03603-662a-4092-ad15-e03c8c1f6aaa"
== copy per line
1: "## Screenshots" | "## Screenshots"
2: "normal mode:" | "<div align=\"center\">\nnormal mode:\n</div>"
3: "gomd screenshot 1 📷 https://github.com/user-attachments/assets/0f0db041-\n0dc3-4d35-8ee1-d89701707aff" | "<img width=\"1536\" height=\"1022\" alt=\"gomd screenshot 1\" src=\"https://github.com/user-attachments/assets/0f0db041-0dc3-4d35-8ee1-d89701707aff\" />"
4: "gomd screenshot 1 📷 https://github.com/user-attachments/assets/0f0db041-\n0dc3-4d35-8ee1-d89701707aff" | "<img width=\"1536\" height=\"1022\" alt=\"gomd screenshot 1\" src=\"https://github.com/user-attachments/assets/0f0db041-0dc3-4d35-8ee1-d89701707aff\" />"
6: "--------" | ""
7: "jump mode:" | "<div align=\"center\">\njump mode:\n</div>"
8: "gomd screenshot 2 📷 https://github.com/user-attachments/assets/ba477053-\n48d4-4380-b682-8f2bf39b72b2" | "<img width=\"1536\" height=\"1020\" alt=\"gomd screenshot 2\" src=\"https://github.com/user-attachments/assets/ba477053-48d4-4380-b682-8f2bf39b72b2\" />"
9: "gomd screenshot 2 📷 https://github.com/user-attachments/assets/ba477053-\n48d4-4380-b682-8f2bf39b72b2" | "<img width=\"1536\" height=\"1020\" alt=\"gomd screenshot 2\" src=\"https://github.com/user-attachments/assets/ba477053-48d4-4380-b682-8f2bf39b72b2\" />"
11: "--------" | ""
12: "interactive mode:" | "<div align=\"center\">\ninteractive mode:\n</div>"
13: "gomd screenshot 3 📷 https://github.com/user-attachments/assets/d0c03603-\n662a-4092-ad15-e03c8c1f6aaa" | "<img width=\"1536\" height=\"1020\" alt=\"gomd screenshot 3\" src=\"https://github.com/user-attachments/assets/d0c03603-662a-4092-ad15-e03c8c1f6aaa\" />"
14: "gomd screenshot 3 📷 https://github.com/user-attachments/assets/d0c03603-\n662a-4092-ad15-e03c8c1f6aaa" | "<img width=\"1536\" height=\"1020\" alt=\"gomd screenshot 3\" src=\"https://github.com/user-attachments/assets/d0c03603-662a-4092-ad15-e03c8c1f6aaa\" />"
######## section 1
== rendered

  ## Features

  • Interactive TUI — dual-pane layout: outline sidebar + rendered content
  • Vim-style navigation —  j / k ,  gg / G ,  /  search
  • Fuzzy search — sidebar heading search uses fuzzy matching (characters in
  order)
  • Breadcrumb bar — title shows current heading path (e.g.,  gomd — file.md >
  Section > Subsection )
  • Interactive mode ( i ) — cycle through code blocks, inline code spans, and
  links; copy with  y , open links with  Enter  (works in any sub-mode)
  • Jump mode ( f ) — EasyMotion-style labels to quickly jump to any
  selectable element
  • Navigation history —  Ctrl+O  /  Ctrl+I  for back/forward heading
  navigation
  • Internal link following —  Enter  on  #anchor  links jumps to the
  referenced heading
  • Mouse support — click sidebar headings, scroll content with mouse wheel
  • Responsive sidebar — width auto-adjusts based on heading text length
  • Multiple themes —  T  to cycle (OceanDark, Dracula, Nord, Solarized, …)
  • CLI mode — list, tree, count, extract sections non-interactively
  • File watcher — auto-reloads on change
  • Editor integration —  e  opens the source file in  $EDITOR
  • Configurable link opener — set  opener  in config to customize how URLs
  are opened
  • Image rendering — inline images via Kitty graphics protocol (Ghostty,
  Kitty, WezTerm)
  • HTML tag support —  <img>  and  <a>  HTML tags are rendered as images and
  links alongside native markdown syntax
== nodes
inline src 3-3 col 31-34  shown 4-4 col 28-29  "j"
inline src 3-3 col 35-38  shown 4-4 col 32-33  "k"
inline src 3-3 col 40-44  shown 4-4 col 37-39  "gg"
inline src 3-3 col 45-48  shown 4-4 col 42-43  "G"
inline src 3-3 col 50-53  shown 4-4 col 47-48  "/"
inline src 5-5 col 65-106  shown 7-8 col 62-78  "gomd — file.md > Section > Subsection"
inline src 6-6 col 24-27  shown 9-9 col 23-24  "i"
inline src 6-6 col 100-103  shown 10-10 col 20-21  "y"
inline src 6-6 col 121-128  shown 10-10 col 41-46  "Enter"
inline src 7-7 col 17-20  shown 11-11 col 16-17  "f"
inline src 8-8 col 29-37  shown 13-13 col 26-32  "Ctrl+O"
inline src 8-8 col 40-48  shown 13-13 col 37-43  "Ctrl+I"
inline src 9-9 col 34-41  shown 15-15 col 31-36  "Enter"
inline src 9-9 col 45-54  shown 15-15 col 42-49  "#anchor"
inline src 12-12 col 26-29  shown 19-19 col 23-24  "T"
inline src 15-15 col 29-32  shown 22-22 col 26-27  "e"
inline src 15-15 col 58-67  shown 22-22 col 55-62  "$EDITOR"
inline src 16-16 col 39-47  shown 23-23 col 36-42  "opener"
inline src 18-18 col 27-34  shown 27-27 col 24-29  "<img>"
inline src 18-18 col 39-44  shown 27-27 col 36-39  "<a>"
== copy per line
1: "## Features" | "## Features"
3: "• Interactive TUI — dual-pane layout: outline sidebar + rendered content" | "- **Interactive TUI** — dual-pane layout: outline sidebar + rendered content"
4: "• Vim-style navigation —  j / k ,  gg / G ,  /  search" | "- **Vim-style navigation** — `j`/`k`, `gg`/`G`, `/` search"
5: "• Fuzzy search — sidebar heading search uses fuzzy matching (characters in" | "- **Fuzzy search** — sidebar heading search uses fuzzy matching (characters in"
6: "order)" | "order)"
7: "• Breadcrumb bar — title shows current heading path (e.g.,  gomd — file.md >\nSection > Subsection" | "- **Breadcrumb bar** — title shows current heading path (e.g., `gomd — file.md > Section > Subsection`"
8: "gomd — file.md >\nSection > Subsection )" | "`gomd — file.md > Section > Subsection`)"
9: "• Interactive mode ( i ) — cycle through code blocks, inline code spans, and" | "- **Interactive mode** (`i`) — cycle through code blocks, inline code spans, and"
10: "links; copy with  y , open links with  Enter  (works in any sub-mode)" | "links; copy with `y`, open links with `Enter` (works in any sub-mode)"
11: "• Jump mode ( f ) — EasyMotion-style labels to quickly jump to any" | "- **Jump mode** (`f`) — EasyMotion-style labels to quickly jump to any"
12: "selectable element" | "selectable element"
13: "• Navigation history —  Ctrl+O  /  Ctrl+I  for back/forward heading" | "- **Navigation history** — `Ctrl+O` / `Ctrl+I` for back/forward heading"
14: "navigation" | "navigation"
15: "• Internal link following —  Enter  on  #anchor  links jumps to the" | "- **Internal link following** — `Enter` on `#anchor` links jumps to the"
16: "referenced heading" | "referenced heading"
17: "• Mouse support — click sidebar headings, scroll content with mouse wheel" | "- **Mouse support** — click sidebar headings, scroll content with mouse wheel"
18: "• Responsive sidebar — width auto-adjusts based on heading text length" | "- **Responsive sidebar** — width auto-adjusts based on heading text length"
19: "• Multiple themes —  T  to cycle (OceanDark, Dracula, Nord, Solarized, …)" | "- **Multiple themes** — `T` to cycle (OceanDark, Dracula, Nord, Solarized, …)"
20: "• CLI mode — list, tree, count, extract sections non-interactively" | "- **CLI mode** — list, tree, count, extract sections non-interactively"
21: "• File watcher — auto-reloads on change" | "- **File watcher** — auto-reloads on change"
22: "• Editor integration —  e  opens the source file in  $EDITOR" | "- **Editor integration** — `e` opens the source file in `$EDITOR`"
23: "• Configurable link opener — set  opener  in config to customize how URLs" | "- **Configurable link opener** — set `opener` in config to customize how URLs"
24: "are opened" | "are opened"
25: "• Image rendering — inline images via Kitty graphics protocol (Ghostty," | "- **Image rendering** — inline images via Kitty graphics protocol (Ghostty,"
26: "Kitty, WezTerm)" | "Kitty, WezTerm)"
27: "• HTML tag support —  <img>  and  <a>  HTML tags are rendered as images and" | "- **HTML tag support** — `<img>` and `<a>` HTML tags are rendered as images and"
28: "links alongside native markdown syntax" | "links alongside native markdown syntax"
######## section 2
== rendered

  ## Installation

  Download the latest release 🔗 https://github.com/grota/gomd/releases/latest
  from this project.

  Or via go install:

    go install github.com/grota/gomd/cmd/gomd@latest

  Or build from source:

    git clone https://github.com/grota/gomd
    cd gomd
    go build ./cmd/gomd

  Requires Go 1.22+.
== nodes
link   src 2-2 col 13-76  shown 3-3 col 15-29  "latest release -> https://github.com/grota/gomd/releases/latest"
block  src 6-8 col 0-0  shown 8-8 col -1--1  "go install github.com/grota/gomd/cmd/gomd@latest"
block  src 12-16 col 0-0  shown 12-14 col -1--1  "git clone https://github.com/grota/gomd\ncd gomd\ngo build ./cmd/gomd"
== copy per line
1: "## Installation" | "## Installation"
3: "Download the latest release 🔗 https://github.com/grota/gomd/releases/latest" | "Download the [latest release](https://github.com/grota/gomd/releases/latest)"
4: "from this project." | "from this project."
6: "Or via go install:" | "Or via go install:"
8: "go install github.com/grota/gomd/cmd/gomd@latest" | "```bash\ngo install github.com/grota/gomd/cmd/gomd@latest\n```"
10: "Or build from source:" | "Or build from source:"
12: "git clone https://github.com/grota/gomd" | "git clone https://github.com/grota/gomd"
13: "cd gomd" | "cd gomd"
14: "go build ./cmd/gomd" | "go build ./cmd/gomd"
16: "Requires Go 1.22+." | "Requires Go 1.22+."
######## section 3
== rendered

  ## Usage

  ### Interactive TUI

    gomd file.md          # open a file
    gomd some/dir/        # open first .md in directory
    cat file.md | gomd    # read from stdin
    gomd                  # print help

  ### Render Mode

  Render mode ( -r  /  --render ) outputs rendered markdown to stdout without
  launching the TUI. Useful for piping, previewing, or printing.

    gomd file.md -r                        # render to stdout with background
    gomd file.md -r --disable-background   # render without background (for
  piping)
    gomd file.md -r --images               # render with inline images

  Background color is emitted by default so the output looks correct in the
  terminal. Use  --disable-background  when piping to other tools or files,
  since ANSI background sequences can interfere with downstream processing.

  Images are rendered as  alt 🖼️ path  and links as  text 🔗 url . HTML  <img>
  and  <a>  tags are converted to their markdown equivalents before rendering.

  #### Key bindings

   Key                          │ Action
  ──────────────────────────────┼─────────────────────────────────────────────
    j  /  ↓                     │ Next heading
    k  /  ↑                     │ Previous heading
    gg  /  G                    │ Jump to root / last heading
    H  /  M  /  L               │ Jump to top / mid / bottom of visible area
    zz  /  zt  /  zb            │ Center / top / bottom selection in viewport
    /                           │ Search headings (fuzzy)
    Tab                         │ Switch focus (sidebar ↔ content)
    w                           │ Toggle sidebar
    i                           │ Enter interactive mode
    f                           │ Jump mode (EasyMotion labels)
    Ctrl+O                      │ Navigate back
    Ctrl+I                      │ Navigate forward
    e                           │ Open in  $EDITOR
    T                           │ Cycle theme
    ?                           │ Help
    q  /  Ctrl+C                │ Quit
   Mouse click                  │ Select sidebar heading
   Scroll wheel                 │ Scroll content

  #### Interactive mode ( i )

  Cycles through selectable nodes in the current section: fenced code blocks,
  inline code spans, and links. Press  m  to cycle sub-modes.

   Key                │ Action
  ────────────────────┼───────────────────────────────────────────────────────
    j  /  k           │ Next / previous node
    m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)
    H  /  M  /  L     │ Jump to top / mid / bottom visible node
    zz  /  zt  /  zb  │ Center / top / bottom node in viewport
    y                 │ Copy node content to clipboard and exit
    Enter             │ Open link (any sub-mode; external: browser;  #anchor
                      │ : jump to heading)
    Esc               │ Exit interactive mode

  #### Jump mode ( f )

  Shows short labels next to all selectable elements. Type a label to jump
  directly to that element (enters interactive ALL mode). Press  Esc  to
  cancel.

  ### CLI mode

    gomd -l file.md                  # list headings
    gomd --tree file.md              # heading tree
    gomd -L 2 file.md                # only h2 headings
    gomd --filter "install" file.md  # filter by text
    gomd --count file.md             # count by level
    gomd -s "Installation" file.md   # extract section
== nodes
block  src 4-9 col 0-0  shown 5-8 col -1--1  "gomd file.md          # open a file\ngomd some/dir/        # open first .md in directory\ncat file.md | gomd    # read from stdin\ngomd                  # print help"
inline src 13-13 col 13-17  shown 12-12 col 16-18  "-r"
inline src 13-13 col 20-30  shown 12-12 col 23-31  "--render"
block  src 15-19 col 0-0  shown 15-18 col -1--1  "gomd file.md -r                        # render to stdout with background\ngomd file.md -r --disable-background   # render without background (for piping)\ngomd file.md -r --images               # render with inline images"
inline src 21-21 col 88-110  shown 21-21 col 17-37  "--disable-background"
inline src 23-23 col 23-41  shown 24-24 col 26-37  "alt 🖼️ path"
inline src 23-23 col 55-70  shown 24-24 col 53-64  "text 🔗 url"
inline src 23-23 col 77-84  shown 24-24 col 73-78  "<img>"
inline src 23-23 col 89-94  shown 25-25 col 7-10  "<a>"
inline src 29-29 col 2-5  shown 31-31 col 4-5  "j"
inline src 29-29 col 8-13  shown 31-31 col 10-11  "↓"
inline src 30-30 col 2-5  shown 32-32 col 4-5  "k"
inline src 30-30 col 8-13  shown 32-32 col 10-11  "↑"
inline src 31-31 col 2-6  shown 33-33 col 4-6  "gg"
inline src 31-31 col 9-12  shown 33-33 col 11-12  "G"
inline src 32-32 col 2-5  shown 34-34 col 4-5  "H"
inline src 32-32 col 8-11  shown 34-34 col 10-11  "M"
inline src 32-32 col 14-17  shown 34-34 col 16-17  "L"
inline src 33-33 col 2-6  shown 35-35 col 4-6  "zz"
inline src 33-33 col 9-13  shown 35-35 col 11-13  "zt"
inline src 33-33 col 16-20  shown 35-35 col 18-20  "zb"
inline src 34-34 col 2-5  shown 36-36 col 4-5  "/"
inline src 35-35 col 2-7  shown 37-37 col 4-7  "Tab"
inline src 36-36 col 2-5  shown 38-38 col 4-5  "w"
inline src 37-37 col 2-5  shown 39-39 col 4-5  "i"
inline src 38-38 col 2-5  shown 40-40 col 4-5  "f"
inline src 39-39 col 2-10  shown 41-41 col 4-10  "Ctrl+O"
inline src 40-40 col 2-10  shown 42-42 col 4-10  "Ctrl+I"
inline src 41-41 col 2-5  shown 43-43 col 4-5  "e"
inline src 41-41 col 31-40  shown 43-43 col 43-50  "$EDITOR"
inline src 42-42 col 2-5  shown 44-44 col 4-5  "T"
inline src 43-43 col 2-5  shown 45-45 col 4-5  "?"
inline src 44-44 col 2-5  shown 46-46 col 4-5  "q"
inline src 44-44 col 8-16  shown 46-46 col 10-16  "Ctrl+C"
inline src 48-48 col 23-26  shown 50-50 col 26-27  "i"
inline src 50-50 col 112-115  shown 53-53 col 39-40  "m"
inline src 54-54 col 2-5  shown 57-57 col 4-5  "j"
inline src 54-54 col 8-11  shown 57-57 col 10-11  "k"
inline src 55-55 col 2-5  shown 58-58 col 4-5  "m"
inline src 56-56 col 2-5  shown 59-59 col 4-5  "H"
inline src 56-56 col 8-11  shown 59-59 col 10-11  "M"
inline src 56-56 col 14-17  shown 59-59 col 16-17  "L"
inline src 57-57 col 2-6  shown 60-60 col 4-6  "zz"
inline src 57-57 col 9-13  shown 60-60 col 11-13  "zt"
inline src 57-57 col 16-20  shown 60-60 col 18-20  "zb"
inline src 58-58 col 2-5  shown 61-61 col 4-5  "y"
inline src 59-59 col 2-9  shown 62-62 col 4-9  "Enter"
inline src 59-59 col 67-76  shown 62-62 col 69-76  "#anchor"
inline src 60-60 col 2-7  shown 64-64 col 4-7  "Esc"
inline src 62-62 col 16-19  shown 66-66 col 19-20  "f"
inline src 64-64 col 135-140  shown 69-69 col 65-68  "Esc"
block  src 68-75 col 0-0  shown 74-79 col -1--1  "gomd -l file.md                  # list headings\ngomd --tree file.md              # heading tree\ngomd -L 2 file.md                # only h2 headings\ngomd --filter \"install\" file.md  # filter by text\ngomd --count file.md             # count by level\ngomd -s \"Installation\" file.md   # extract section"
== copy per line
1: "## Usage" | "## Usage"
3: "### Interactive TUI" | "### Interactive TUI"
5: "gomd file.md          # open a file" | "gomd file.md          # open a file"
6: "gomd some/dir/        # open first .md in directory" | "gomd some/dir/        # open first .md in directory"
7: "cat file.md | gomd    # read from stdin" | "cat file.md | gomd    # read from stdin"
8: "gomd                  # print help" | "gomd                  # print help"
10: "### Render Mode" | "### Render Mode"
12: "Render mode ( -r  /  --render ) outputs rendered markdown to stdout without" | "Render mode (`-r` / `--render`) outputs rendered markdown to stdout without"
13: "launching the TUI. Useful for piping, previewing, or printing." | "launching the TUI. Useful for piping, previewing, or printing."
15: "gomd file.md -r                        # render to stdout with background" | "gomd file.md -r                        # render to stdout with background"
16: "gomd file.md -r --disable-background   # render without background (for" | "gomd file.md -r --disable-background   # render without background (for"
17: "piping)" | "piping)"
18: "gomd file.md -r --images               # render with inline images" | "gomd file.md -r --images               # render with inline images"
20: "Background color is emitted by default so the output looks correct in the" | "Background color is emitted by default so the output looks correct in the"
21: "terminal. Use  --disable-background  when piping to other tools or files," | "terminal. Use `--disable-background` when piping to other tools or files,"
22: "since ANSI background sequences can interfere with downstream processing." | "since ANSI background sequences can interfere with downstream processing."
24: "Images are rendered as  alt 🖼️ path  and links as  text 🔗 url . HTML  <img>" | "Images are rendered as `alt 🖼️ path` and links as `text 🔗 url`. HTML `<img>`"
25: "and  <a>  tags are converted to their markdown equivalents before rendering." | "and `<a>` tags are converted to their markdown equivalents before rendering."
27: "#### Key bindings" | "#### Key bindings"
29: " Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
30: "──────────────────────────────┼─────────────────────────────────────────────" | ""
31: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
32: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
33: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
34: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
35: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
36: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
37: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
38: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
39: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
40: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
41: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
42: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
43: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
44: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
45: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
46: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
47: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
48: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
50: "#### Interactive mode ( i )" | "#### Interactive mode (`i`)"
52: "Cycles through selectable nodes in the current section: fenced code blocks," | "Cycles through selectable nodes in the current section: fenced code blocks,"
53: "inline code spans, and links. Press  m  to cycle sub-modes." | "inline code spans, and links. Press `m` to cycle sub-modes."
55: " Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
56: "────────────────────┼───────────────────────────────────────────────────────" | ""
57: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
58: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
59: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
60: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
61: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
62: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
63: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
64: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
66: "#### Jump mode ( f )" | "#### Jump mode (`f`)"
68: "Shows short labels next to all selectable elements. Type a label to jump" | "Shows short labels next to all selectable elements. Type a label to jump"
69: "directly to that element (enters interactive ALL mode). Press  Esc  to" | "directly to that element (enters interactive ALL mode). Press `Esc` to"
70: "cancel." | "cancel."
72: "### CLI mode" | "### CLI mode"
74: "gomd -l file.md                  # list headings" | "gomd -l file.md                  # list headings"
75: "gomd --tree file.md              # heading tree" | "gomd --tree file.md              # heading tree"
76: "gomd -L 2 file.md                # only h2 headings" | "gomd -L 2 file.md                # only h2 headings"
77: "gomd --filter \"install\" file.md  # filter by text" | "gomd --filter \"install\" file.md  # filter by text"
78: "gomd --count file.md             # count by level" | "gomd --count file.md             # count by level"
79: "gomd -s \"Installation\" file.md   # extract section" | "gomd -s \"Installation\" file.md   # extract section"
######## section 4
== rendered

  ### Interactive TUI

    gomd file.md          # open a file
    gomd some/dir/        # open first .md in directory
    cat file.md | gomd    # read from stdin
    gomd                  # print help
== nodes
block  src 2-7 col 0-0  shown 3-6 col -1--1  "gomd file.md          # open a file\ngomd some/dir/        # open first .md in directory\ncat file.md | gomd    # read from stdin\ngomd                  # print help"
== copy per line
1: "### Interactive TUI" | "### Interactive TUI"
3: "gomd file.md          # open a file" | "gomd file.md          # open a file"
4: "gomd some/dir/        # open first .md in directory" | "gomd some/dir/        # open first .md in directory"
5: "cat file.md | gomd    # read from stdin" | "cat file.md | gomd    # read from stdin"
6: "gomd                  # print help" | "gomd                  # print help"
######## section 5
== rendered

  ### Render Mode

  Render mode ( -r  /  --render ) outputs rendered markdown to stdout without
  launching the TUI. Useful for piping, previewing, or printing.

    gomd file.md -r                        # render to stdout with background
    gomd file.md -r --disable-background   # render without background (for
  piping)
    gomd file.md -r --images               # render with inline images

  Background color is emitted by default so the output looks correct in the
  terminal. Use  --disable-background  when piping to other tools or files,
  since ANSI background sequences can interfere with downstream processing.

  Images are rendered as  alt 🖼️ path  and links as  text 🔗 url . HTML  <img>
  and  <a>  tags are converted to their markdown equivalents before rendering.

  #### Key bindings

   Key                          │ Action
  ──────────────────────────────┼─────────────────────────────────────────────
    j  /  ↓                     │ Next heading
    k  /  ↑                     │ Previous heading
    gg  /  G                    │ Jump to root / last heading
    H  /  M  /  L               │ Jump to top / mid / bottom of visible area
    zz  /  zt  /  zb            │ Center / top / bottom selection in viewport
    /                           │ Search headings (fuzzy)
    Tab                         │ Switch focus (sidebar ↔ content)
    w                           │ Toggle sidebar
    i                           │ Enter interactive mode
    f                           │ Jump mode (EasyMotion labels)
    Ctrl+O                      │ Navigate back
    Ctrl+I                      │ Navigate forward
    e                           │ Open in  $EDITOR
    T                           │ Cycle theme
    ?                           │ Help
    q  /  Ctrl+C                │ Quit
   Mouse click                  │ Select sidebar heading
   Scroll wheel                 │ Scroll content

  #### Interactive mode ( i )

  Cycles through selectable nodes in the current section: fenced code blocks,
  inline code spans, and links. Press  m  to cycle sub-modes.

   Key                │ Action
  ────────────────────┼───────────────────────────────────────────────────────
    j  /  k           │ Next / previous node
    m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)
    H  /  M  /  L     │ Jump to top / mid / bottom visible node
    zz  /  zt  /  zb  │ Center / top / bottom node in viewport
    y                 │ Copy node content to clipboard and exit
    Enter             │ Open link (any sub-mode; external: browser;  #anchor
                      │ : jump to heading)
    Esc               │ Exit interactive mode

  #### Jump mode ( f )

  Shows short labels next to all selectable elements. Type a label to jump
  directly to that element (enters interactive ALL mode). Press  Esc  to
  cancel.
== nodes
inline src 2-2 col 13-17  shown 3-3 col 16-18  "-r"
inline src 2-2 col 20-30  shown 3-3 col 23-31  "--render"
block  src 4-8 col 0-0  shown 6-9 col -1--1  "gomd file.md -r                        # render to stdout with background\ngomd file.md -r --disable-background   # render without background (for piping)\ngomd file.md -r --images               # render with inline images"
inline src 10-10 col 88-110  shown 12-12 col 17-37  "--disable-background"
inline src 12-12 col 23-41  shown 15-15 col 26-37  "alt 🖼️ path"
inline src 12-12 col 55-70  shown 15-15 col 53-64  "text 🔗 url"
inline src 12-12 col 77-84  shown 15-15 col 73-78  "<img>"
inline src 12-12 col 89-94  shown 16-16 col 7-10  "<a>"
inline src 18-18 col 2-5  shown 22-22 col 4-5  "j"
inline src 18-18 col 8-13  shown 22-22 col 10-11  "↓"
inline src 19-19 col 2-5  shown 23-23 col 4-5  "k"
inline src 19-19 col 8-13  shown 23-23 col 10-11  "↑"
inline src 20-20 col 2-6  shown 24-24 col 4-6  "gg"
inline src 20-20 col 9-12  shown 24-24 col 11-12  "G"
inline src 21-21 col 2-5  shown 25-25 col 4-5  "H"
inline src 21-21 col 8-11  shown 25-25 col 10-11  "M"
inline src 21-21 col 14-17  shown 25-25 col 16-17  "L"
inline src 22-22 col 2-6  shown 26-26 col 4-6  "zz"
inline src 22-22 col 9-13  shown 26-26 col 11-13  "zt"
inline src 22-22 col 16-20  shown 26-26 col 18-20  "zb"
inline src 23-23 col 2-5  shown 27-27 col 4-5  "/"
inline src 24-24 col 2-7  shown 28-28 col 4-7  "Tab"
inline src 25-25 col 2-5  shown 29-29 col 4-5  "w"
inline src 26-26 col 2-5  shown 30-30 col 4-5  "i"
inline src 27-27 col 2-5  shown 31-31 col 4-5  "f"
inline src 28-28 col 2-10  shown 32-32 col 4-10  "Ctrl+O"
inline src 29-29 col 2-10  shown 33-33 col 4-10  "Ctrl+I"
inline src 30-30 col 2-5  shown 34-34 col 4-5  "e"
inline src 30-30 col 31-40  shown 34-34 col 43-50  "$EDITOR"
inline src 31-31 col 2-5  shown 35-35 col 4-5  "T"
inline src 32-32 col 2-5  shown 36-36 col 4-5  "?"
inline src 33-33 col 2-5  shown 37-37 col 4-5  "q"
inline src 33-33 col 8-16  shown 37-37 col 10-16  "Ctrl+C"
inline src 37-37 col 23-26  shown 41-41 col 26-27  "i"
inline src 39-39 col 112-115  shown 44-44 col 39-40  "m"
inline src 43-43 col 2-5  shown 48-48 col 4-5  "j"
inline src 43-43 col 8-11  shown 48-48 col 10-11  "k"
inline src 44-44 col 2-5  shown 49-49 col 4-5  "m"
inline src 45-45 col 2-5  shown 50-50 col 4-5  "H"
inline src 45-45 col 8-11  shown 50-50 col 10-11  "M"
inline src 45-45 col 14-17  shown 50-50 col 16-17  "L"
inline src 46-46 col 2-6  shown 51-51 col 4-6  "zz"
inline src 46-46 col 9-13  shown 51-51 col 11-13  "zt"
inline src 46-46 col 16-20  shown 51-51 col 18-20  "zb"
inline src 47-47 col 2-5  shown 52-52 col 4-5  "y"
inline src 48-48 col 2-9  shown 53-53 col 4-9  "Enter"
inline src 48-48 col 67-76  shown 53-53 col 69-76  "#anchor"
inline src 49-49 col 2-7  shown 55-55 col 4-7  "Esc"
inline src 51-51 col 16-19  shown 57-57 col 19-20  "f"
inline src 53-53 col 135-140  shown 60-60 col 65-68  "Esc"
== copy per line
1: "### Render Mode" | "### Render Mode"
3: "Render mode ( -r  /  --render ) outputs rendered markdown to stdout without" | "Render mode (`-r` / `--render`) outputs rendered markdown to stdout without"
4: "launching the TUI. Useful for piping, previewing, or printing." | "launching the TUI. Useful for piping, previewing, or printing."
6: "gomd file.md -r                        # render to stdout with background" | "gomd file.md -r                        # render to stdout with background"
7: "gomd file.md -r --disable-background   # render without background (for" | "gomd file.md -r --disable-background   # render without background (for"
8: "piping)" | "piping)"
9: "gomd file.md -r --images               # render with inline images" | "gomd file.md -r --images               # render with inline images"
11: "Background color is emitted by default so the output looks correct in the" | "Background color is emitted by default so the output looks correct in the"
12: "terminal. Use  --disable-background  when piping to other tools or files," | "terminal. Use `--disable-background` when piping to other tools or files,"
13: "since ANSI background sequences can interfere with downstream processing." | "since ANSI background sequences can interfere with downstream processing."
15: "Images are rendered as  alt 🖼️ path  and links as  text 🔗 url . HTML  <img>" | "Images are rendered as `alt 🖼️ path` and links as `text 🔗 url`. HTML `<img>`"
16: "and  <a>  tags are converted to their markdown equivalents before rendering." | "and `<a>` tags are converted to their markdown equivalents before rendering."
18: "#### Key bindings" | "#### Key bindings"
20: " Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
21: "──────────────────────────────┼─────────────────────────────────────────────" | ""
22: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
23: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
24: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
25: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
26: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
27: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
28: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
29: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
30: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
31: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
32: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
33: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
34: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
35: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
36: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
37: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
38: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
39: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
41: "#### Interactive mode ( i )" | "#### Interactive mode (`i`)"
43: "Cycles through selectable nodes in the current section: fenced code blocks," | "Cycles through selectable nodes in the current section: fenced code blocks,"
44: "inline code spans, and links. Press  m  to cycle sub-modes." | "inline code spans, and links. Press `m` to cycle sub-modes."
46: " Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
47: "────────────────────┼───────────────────────────────────────────────────────" | ""
48: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
49: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
50: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
51: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
52: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
53: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
54: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
55: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
57: "#### Jump mode ( f )" | "#### Jump mode (`f`)"
59: "Shows short labels next to all selectable elements. Type a label to jump" | "Shows short labels next to all selectable elements. Type a label to jump"
60: "directly to that element (enters interactive ALL mode). Press  Esc  to" | "directly to that element (enters interactive ALL mode). Press `Esc` to"
61: "cancel." | "cancel."
######## section 6
== rendered

  #### Key bindings

   Key                          │ Action
  ──────────────────────────────┼─────────────────────────────────────────────
    j  /  ↓                     │ Next heading
    k  /  ↑                     │ Previous heading
    gg  /  G                    │ Jump to root / last heading
    H  /  M  /  L               │ Jump to top / mid / bottom of visible area
    zz  /  zt  /  zb            │ Center / top / bottom selection in viewport
    /                           │ Search headings (fuzzy)
    Tab                         │ Switch focus (sidebar ↔ content)
    w                           │ Toggle sidebar
    i                           │ Enter interactive mode
    f                           │ Jump mode (EasyMotion labels)
    Ctrl+O                      │ Navigate back
    Ctrl+I                      │ Navigate forward
    e                           │ Open in  $EDITOR
    T                           │ Cycle theme
    ?                           │ Help
    q  /  Ctrl+C                │ Quit
   Mouse click                  │ Select sidebar heading
   Scroll wheel                 │ Scroll content
== nodes
inline src 4-4 col 2-5  shown 5-5 col 4-5  "j"
inline src 4-4 col 8-13  shown 5-5 col 10-11  "↓"
inline src 5-5 col 2-5  shown 6-6 col 4-5  "k"
inline src 5-5 col 8-13  shown 6-6 col 10-11  "↑"
inline src 6-6 col 2-6  shown 7-7 col 4-6  "gg"
inline src 6-6 col 9-12  shown 7-7 col 11-12  "G"
inline src 7-7 col 2-5  shown 8-8 col 4-5  "H"
inline src 7-7 col 8-11  shown 8-8 col 10-11  "M"
inline src 7-7 col 14-17  shown 8-8 col 16-17  "L"
inline src 8-8 col 2-6  shown 9-9 col 4-6  "zz"
inline src 8-8 col 9-13  shown 9-9 col 11-13  "zt"
inline src 8-8 col 16-20  shown 9-9 col 18-20  "zb"
inline src 9-9 col 2-5  shown 10-10 col 4-5  "/"
inline src 10-10 col 2-7  shown 11-11 col 4-7  "Tab"
inline src 11-11 col 2-5  shown 12-12 col 4-5  "w"
inline src 12-12 col 2-5  shown 13-13 col 4-5  "i"
inline src 13-13 col 2-5  shown 14-14 col 4-5  "f"
inline src 14-14 col 2-10  shown 15-15 col 4-10  "Ctrl+O"
inline src 15-15 col 2-10  shown 16-16 col 4-10  "Ctrl+I"
inline src 16-16 col 2-5  shown 17-17 col 4-5  "e"
inline src 16-16 col 31-40  shown 17-17 col 43-50  "$EDITOR"
inline src 17-17 col 2-5  shown 18-18 col 4-5  "T"
inline src 18-18 col 2-5  shown 19-19 col 4-5  "?"
inline src 19-19 col 2-5  shown 20-20 col 4-5  "q"
inline src 19-19 col 8-16  shown 20-20 col 10-16  "Ctrl+C"
== copy per line
1: "#### Key bindings" | "#### Key bindings"
3: " Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
4: "──────────────────────────────┼─────────────────────────────────────────────" | ""
5: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
6: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
7: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
8: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
9: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
10: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
11: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
12: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
13: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
14: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
15: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
16: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
17: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
18: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
19: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
20: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
21: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
22: "Key                          │ Action\n──────────────────────────────┼─────────────────────────────────────────────\n  j  /  ↓                     │ Next heading\n  k  /  ↑                     │ Previous heading\n  gg  /  G                    │ Jump to root / last heading\n  H  /  M  /  L               │ Jump to top / mid / bottom of visible area\n  zz  /  zt  /  zb            │ Center / top / bottom selection in viewport\n  /                           │ Search headings (fuzzy)\n  Tab                         │ Switch focus (sidebar ↔ content)\n  w                           │ Toggle sidebar\n  i                           │ Enter interactive mode\n  f                           │ Jump mode (EasyMotion labels)\n  Ctrl+O                      │ Navigate back\n  Ctrl+I                      │ Navigate forward\n  e                           │ Open in  $EDITOR\n  T                           │ Cycle theme\n  ?                           │ Help\n  q  /  Ctrl+C                │ Quit\n Mouse click                  │ Select sidebar heading\n Scroll wheel                 │ Scroll content" | "| Key                | Action                                      |\n| ------------------ | ------------------------------------------- |\n| `j` / `↓`          | Next heading                                |\n| `k` / `↑`          | Previous heading                            |\n| `gg` / `G`         | Jump to root / last heading                 |\n| `H` / `M` / `L`    | Jump to top / mid / bottom of visible area  |\n| `zz` / `zt` / `zb` | Center / top / bottom selection in viewport |\n| `/`                | Search headings (fuzzy)                     |\n| `Tab`              | Switch focus (sidebar ↔ content)            |\n| `w`                | Toggle sidebar                              |\n| `i`                | Enter interactive mode                      |\n| `f`                | Jump mode (EasyMotion labels)               |\n| `Ctrl+O`           | Navigate back                               |\n| `Ctrl+I`           | Navigate forward                            |\n| `e`                | Open in `$EDITOR`                           |\n| `T`                | Cycle theme                                 |\n| `?`                | Help                                        |\n| `q` / `Ctrl+C`     | Quit                                        |\n| Mouse click        | Select sidebar heading                      |\n| Scroll wheel       | Scroll content                              |"
######## section 7
== rendered

  #### Interactive mode ( i )

  Cycles through selectable nodes in the current section: fenced code blocks,
  inline code spans, and links. Press  m  to cycle sub-modes.

   Key                │ Action
  ────────────────────┼───────────────────────────────────────────────────────
    j  /  k           │ Next / previous node
    m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)
    H  /  M  /  L     │ Jump to top / mid / bottom visible node
    zz  /  zt  /  zb  │ Center / top / bottom node in viewport
    y                 │ Copy node content to clipboard and exit
    Enter             │ Open link (any sub-mode; external: browser;  #anchor
                      │ : jump to heading)
    Esc               │ Exit interactive mode
== nodes
inline src 0-0 col 23-26  shown 1-1 col 26-27  "i"
inline src 2-2 col 112-115  shown 4-4 col 39-40  "m"
inline src 6-6 col 2-5  shown 8-8 col 4-5  "j"
inline src 6-6 col 8-11  shown 8-8 col 10-11  "k"
inline src 7-7 col 2-5  shown 9-9 col 4-5  "m"
inline src 8-8 col 2-5  shown 10-10 col 4-5  "H"
inline src 8-8 col 8-11  shown 10-10 col 10-11  "M"
inline src 8-8 col 14-17  shown 10-10 col 16-17  "L"
inline src 9-9 col 2-6  shown 11-11 col 4-6  "zz"
inline src 9-9 col 9-13  shown 11-11 col 11-13  "zt"
inline src 9-9 col 16-20  shown 11-11 col 18-20  "zb"
inline src 10-10 col 2-5  shown 12-12 col 4-5  "y"
inline src 11-11 col 2-9  shown 13-13 col 4-9  "Enter"
inline src 11-11 col 67-76  shown 13-13 col 69-76  "#anchor"
inline src 12-12 col 2-7  shown 15-15 col 4-7  "Esc"
== copy per line
1: "#### Interactive mode ( i )" | "#### Interactive mode (`i`)"
3: "Cycles through selectable nodes in the current section: fenced code blocks," | "Cycles through selectable nodes in the current section: fenced code blocks,"
4: "inline code spans, and links. Press  m  to cycle sub-modes." | "inline code spans, and links. Press `m` to cycle sub-modes."
6: " Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
7: "────────────────────┼───────────────────────────────────────────────────────" | ""
8: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
9: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
10: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
11: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
12: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
13: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
14: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
15: "Key                │ Action\n────────────────────┼───────────────────────────────────────────────────────\n  j  /  k           │ Next / previous node\n  m                 │ Cycle sub-mode (ALL → CODE → INLINE → LINKS)\n  H  /  M  /  L     │ Jump to top / mid / bottom visible node\n  zz  /  zt  /  zb  │ Center / top / bottom node in viewport\n  y                 │ Copy node content to clipboard and exit\n  Enter             │ Open link (any sub-mode; external: browser;  #anchor\n                    │ : jump to heading)\n  Esc               │ Exit interactive mode" | "| Key                | Action                                                    |\n| ------------------ | --------------------------------------------------------- |\n| `j` / `k`          | Next / previous node                                      |\n| `m`                | Cycle sub-mode (ALL → CODE → INLINE → LINKS)              |\n| `H` / `M` / `L`    | Jump to top / mid / bottom visible node                   |\n| `zz` / `zt` / `zb` | Center / top / bottom node in viewport                    |\n| `y`                | Copy node content to clipboard and exit                   |\n| `Enter`            | Open link (any sub-mode; external: browser; `#anchor`: jump to heading) |\n| `Esc`              | Exit interactive mode                                     |"
######## section 8
== rendered

  #### Jump mode ( f )

  Shows short labels next to all selectable elements. Type a label to jump
  directly to that element (enters interactive ALL mode). Press  Esc  to
  cancel.
== nodes
inline src 0-0 col 16-19  shown 1-1 col 19-20  "f"
inline src 2-2 col 135-140  shown 4-4 col 65-68  "Esc"
== copy per line
1: "#### Jump mode ( f )" | "#### Jump mode (`f`)"
3: "Shows short labels next to all selectable elements. Type a label to jump" | "Shows short labels next to all selectable elements. Type a label to jump"
4: "directly to that element (enters interactive ALL mode). Press  Esc  to" | "directly to that element (enters interactive ALL mode). Press `Esc` to"
5: "cancel." | "cancel."
######## section 9
== rendered

  ### CLI mode

    gomd -l file.md                  # list headings
    gomd --tree file.md              # heading tree
    gomd -L 2 file.md                # only h2 headings
    gomd --filter "install" file.md  # filter by text
    gomd --count file.md             # count by level
    gomd -s "Installation" file.md   # extract section
== nodes
block  src 2-9 col 0-0  shown 3-8 col -1--1  "gomd -l file.md                  # list headings\ngomd --tree file.md              # heading tree\ngomd -L 2 file.md                # only h2 headings\ngomd --filter \"install\" file.md  # filter by text\ngomd --count file.md             # count by level\ngomd -s \"Installation\" file.md   # extract section"
== copy per line
1: "### CLI mode" | "### CLI mode"
3: "gomd -l file.md                  # list headings" | "gomd -l file.md                  # list headings"
4: "gomd --tree file.md              # heading tree" | "gomd --tree file.md              # heading tree"
5: "gomd -L 2 file.md                # only h2 headings" | "gomd -L 2 file.md                # only h2 headings"
6: "gomd --filter \"install\" file.md  # filter by text" | "gomd --filter \"install\" file.md  # filter by text"
7: "gomd --count file.md             # count by level" | "gomd --count file.md             # count by level"
8: "gomd -s \"Installation\" file.md   # extract section" | "gomd -s \"Installation\" file.md   # extract section"
######## section 10
== rendered

  ## Configuration

  Config file:  ~/.config/gomd/config.toml

  All settings are optional — only specify what you want to override.

    [ui]
    theme = "OceanDark"       # OceanDark | Dracula | Nord | Gruvbox |
  TokyoNight (or a Ghostty theme name)
    compact_tree = false      # compact sidebar tree
    opener = "xdg-open"       # command to open URLs (macOS: "open")
    ghostty_theme_directory = ""  # path to Ghostty themes (e.g.,
  "/usr/share/ghostty/themes/")

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

  Key bindings use comma-separated values for multiple keys mapped to the same
  action. Key names follow bubbletea conventions (e.g.,  ctrl+d ,  shift+tab ,
  pgdown ,  esc ).
== nodes
inline src 2-2 col 13-41  shown 3-3 col 16-42  "~/.config/gomd/config.toml"
block  src 6-79 col 0-0  shown 7-80 col -1--1  "[ui]\ntheme = \"OceanDark\"       # OceanDark | Dracula | Nord | Gruvbox | TokyoNight (or a Ghostty theme name)\ncompact_tree = false      # compact sidebar tree\nopener = \"xdg-open\"       # command to open URLs (macOS: \"open\")\nghostty_theme_directory = \"\"  # path to Ghostty themes (e.g., \"/usr/share/ghostty/themes/\")\n\n[ui.theme_override]\n# Override individual colors of the active theme (hex values).\n# Any field left empty keeps the theme default.\n# border = \"#504945\"\n# selected = \"#3c3836\"\n# heading1 = \"#fabd2f\"\n# heading2 = \"#b8bb26\"\n# heading3 = \"#83a598\"\n# headingn = \"#928374\"\n# background = \"#282828\"\n# foreground = \"#ebdbb2\"\n# statusbar = \"#3c3836\"\n# highlight = \"#fe8019\"\n# code = \"#1d2021\"\n# search = \"#fb4934\"\n# nodesel = \"#8ec07c\"\n\n[terminal]\ncolor_mode = \"auto\"       # auto | truecolor | 256 | ansi | none\n\n[images]\nenabled = true\n\n[content]\nsyntax_highlighting = true\n\n[keys]\n# Shared\nquit = \"q\"\nhelp = \"?\"\ntheme_picker = \"T\"\ntoggle_focus = \"tab\"\ntoggle_sidebar = \"w\"\nreload = \"r\"\nedit = \"e\"\n\n# Sidebar navigation\nsidebar_down = \"j,down\"\nsidebar_up = \"k,up\"\nsidebar_top = \"g\"\nsidebar_bottom = \"G\"\nsidebar_search = \"/\"\nnext_match = \"n\"\nprev_match = \"N\"\n\n# Content navigation\nscroll_down = \"j,down\"\nscroll_up = \"k,up\"\nscroll_half_down = \"ctrl+d,ctrl+f,pgdown, \"\nscroll_half_up = \"ctrl+u,ctrl+b,pgup\"\ncontent_top = \"g\"\ncontent_bottom = \"G\"\ncontent_search = \"/\"\ncontent_next_match = \"n\"\ncontent_prev_match = \"N\"\njump = \"f\"\nnav_back = \"ctrl+o\"\nnav_forward = \"ctrl+i\"\n\n# Node-select mode\nnode_select = \"i\"\nnode_next = \"j,down,tab\"\nnode_prev = \"k,up,shift+tab\"\nnode_copy = \"y\"\nnode_open = \"enter\"\nnode_exit = \"esc,q,i\""
inline src 81-81 col 131-139  shown 83-83 col 57-63  "ctrl+d"
inline src 81-81 col 141-152  shown 83-83 col 67-76  "shift+tab"
inline src 81-81 col 154-162  shown 84-84 col 2-8  "pgdown"
inline src 81-81 col 164-169  shown 84-84 col 12-15  "esc"
== copy per line
1: "## Configuration" | "## Configuration"
3: "Config file:  ~/.config/gomd/config.toml" | "Config file: `~/.config/gomd/config.toml`"
5: "All settings are optional — only specify what you want to override." | "All settings are optional — only specify what you want to override."
7: "[ui]" | "[ui]"
8: "theme = \"OceanDark\"       # OceanDark | Dracula | Nord | Gruvbox |" | "theme = \"OceanDark\"       # OceanDark | Dracula | Nord | Gruvbox |"
9: "TokyoNight (or a Ghostty theme name)" | "TokyoNight (or a Ghostty theme name)"
10: "compact_tree = false      # compact sidebar tree" | "compact_tree = false      # compact sidebar tree"
11: "opener = \"xdg-open\"       # command to open URLs (macOS: \"open\")" | "opener = \"xdg-open\"       # command to open URLs (macOS: \"open\")"
12: "ghostty_theme_directory = \"\"  # path to Ghostty themes (e.g.," | "ghostty_theme_directory = \"\"  # path to Ghostty themes (e.g.,"
13: "\"/usr/share/ghostty/themes/\")" | "\"/usr/share/ghostty/themes/\")"
15: "[ui.theme_override]" | "[ui.theme_override]"
16: "# Override individual colors of the active theme (hex values)." | "# Override individual colors of the active theme (hex values)."
17: "# Any field left empty keeps the theme default." | "# Any field left empty keeps the theme default."
18: "# border = \"#504945\"" | "# border = \"#504945\""
19: "# selected = \"#3c3836\"" | "# selected = \"#3c3836\""
20: "# heading1 = \"#fabd2f\"" | "# heading1 = \"#fabd2f\""
21: "# heading2 = \"#b8bb26\"" | "# heading2 = \"#b8bb26\""
22: "# heading3 = \"#83a598\"" | "# heading3 = \"#83a598\""
23: "# headingn = \"#928374\"" | "# headingn = \"#928374\""
24: "# background = \"#282828\"" | "# background = \"#282828\""
25: "# foreground = \"#ebdbb2\"" | "# foreground = \"#ebdbb2\""
26: "# statusbar = \"#3c3836\"" | "# statusbar = \"#3c3836\""
27: "# highlight = \"#fe8019\"" | "# highlight = \"#fe8019\""
28: "# code = \"#1d2021\"" | "# code = \"#1d2021\""
29: "# search = \"#fb4934\"" | "# search = \"#fb4934\""
30: "# nodesel = \"#8ec07c\"" | "# nodesel = \"#8ec07c\""
32: "[terminal]" | "[terminal]"
33: "color_mode = \"auto\"       # auto | truecolor | 256 | ansi | none" | "color_mode = \"auto\"       # auto | truecolor | 256 | ansi | none"
35: "[images]" | "[images]"
36: "enabled = true" | "enabled = true"
38: "[content]" | "[content]"
39: "syntax_highlighting = true" | "syntax_highlighting = true"
41: "[keys]" | "[keys]"
42: "# Shared" | "# Shared"
43: "quit = \"q\"" | "quit = \"q\""
44: "help = \"?\"" | "help = \"?\""
45: "theme_picker = \"T\"" | "theme_picker = \"T\""
46: "toggle_focus = \"tab\"" | "toggle_focus = \"tab\""
47: "toggle_sidebar = \"w\"" | "toggle_sidebar = \"w\""
48: "reload = \"r\"" | "reload = \"r\""
49: "edit = \"e\"" | "edit = \"e\""
51: "# Sidebar navigation" | "# Sidebar navigation"
52: "sidebar_down = \"j,down\"" | "sidebar_down = \"j,down\""
53: "sidebar_up = \"k,up\"" | "sidebar_up = \"k,up\""
54: "sidebar_top = \"g\"" | "sidebar_top = \"g\""
55: "sidebar_bottom = \"G\"" | "sidebar_bottom = \"G\""
56: "sidebar_search = \"/\"" | "sidebar_search = \"/\""
57: "next_match = \"n\"" | "next_match = \"n\""
58: "prev_match = \"N\"" | "prev_match = \"N\""
60: "# Content navigation" | "# Content navigation"
61: "scroll_down = \"j,down\"" | "scroll_down = \"j,down\""
62: "scroll_up = \"k,up\"" | "scroll_up = \"k,up\""
63: "scroll_half_down = \"ctrl+d,ctrl+f,pgdown, \"" | "scroll_half_down = \"ctrl+d,ctrl+f,pgdown, \""
64: "scroll_half_up = \"ctrl+u,ctrl+b,pgup\"" | "scroll_half_up = \"ctrl+u,ctrl+b,pgup\""
65: "content_top = \"g\"" | "content_top = \"g\""
66: "content_bottom = \"G\"" | "content_bottom = \"G\""
67: "content_search = \"/\"" | "content_search = \"/\""
68: "content_next_match = \"n\"" | "content_next_match = \"n\""
69: "content_prev_match = \"N\"" | "content_prev_match = \"N\""
70: "jump = \"f\"" | "jump = \"f\""
71: "nav_back = \"ctrl+o\"" | "nav_back = \"ctrl+o\""
72: "nav_forward = \"ctrl+i\"" | "nav_forward = \"ctrl+i\""
74: "# Node-select mode" | "# Node-select mode"
75: "node_select = \"i\"" | "node_select = \"i\""
76: "node_next = \"j,down,tab\"" | "node_next = \"j,down,tab\""
77: "node_prev = \"k,up,shift+tab\"" | "node_prev = \"k,up,shift+tab\""
78: "node_copy = \"y\"" | "node_copy = \"y\""
79: "node_open = \"enter\"" | "node_open = \"enter\""
80: "node_exit = \"esc,q,i\"" | "node_exit = \"esc,q,i\""
82: "Key bindings use comma-separated values for multiple keys mapped to the same" | "Key bindings use comma-separated values for multiple keys mapped to the same"
83: "action. Key names follow bubbletea conventions (e.g.,  ctrl+d ,  shift+tab ," | "action. Key names follow bubbletea conventions (e.g., `ctrl+d`, `shift+tab`,"
84: "pgdown ,  esc )." | "`pgdown`, `esc`)."
######## section 11
== rendered

  ## Theme resolution

  gomd resolves the active theme in three steps:

  1. Built-in lookup — if  theme  matches a built-in name (OceanDark, Nord,
  Dracula, Gruvbox, TokyoNight), that theme is used.
  2. Ghostty fallback — if the name doesn't match a built-in theme and
  ghostty_theme_directory  is set, gomd looks for a file with that name in the
  directory and converts the Ghostty color palette to a gomd theme. The
  mapping is:
    •  background  /  foreground  → direct
    • palette 0 → Selected, StatusBar, Code
    • palette 1 → Search (red)
    • palette 2 → Heading2 (green)
    • palette 3 → Heading1 (yellow)
    • palette 4 → Heading3 (blue)
    • palette 6 → NodeSel (cyan)
    • palette 8 → Border, HeadingN (bright black)
    • palette 11 → Highlight (bright yellow)
  3. Color overrides — any non-empty field in  [ui.theme_override]  replaces
  the corresponding color, regardless of whether the base theme is built-in or
  loaded from Ghostty.

  If the theme name doesn't match a built-in and no Ghostty directory is
  configured (or the file isn't found), gomd falls back to OceanDark.
== nodes
inline src 4-4 col 30-37  shown 5-5 col 27-32  "theme"
inline src 5-5 col 75-100  shown 8-8 col 2-25  "ghostty_theme_directory"
inline src 6-6 col 5-17  shown 11-11 col 7-17  "background"
inline src 6-6 col 20-32  shown 11-11 col 22-32  "foreground"
inline src 15-15 col 50-71  shown 20-20 col 47-66  "[ui.theme_override]"
== copy per line
1: "## Theme resolution" | "## Theme resolution"
3: "gomd resolves the active theme in three steps:" | "gomd resolves the active theme in three steps:"
5: "1. Built-in lookup — if  theme  matches a built-in name (OceanDark, Nord," | "1. **Built-in lookup** — if `theme` matches a built-in name (OceanDark, Nord,"
6: "Dracula, Gruvbox, TokyoNight), that theme is used." | "Dracula, Gruvbox, TokyoNight), that theme is used."
7: "2. Ghostty fallback — if the name doesn't match a built-in theme and" | "2. **Ghostty fallback** — if the name doesn't match a built-in theme and"
8: "ghostty_theme_directory  is set, gomd looks for a file with that name in the" | "`ghostty_theme_directory` is set, gomd looks for a file with that name in the"
9: "directory and converts the Ghostty color palette to a gomd theme. The" | "directory and converts the Ghostty color palette to a gomd theme. The"
10: "mapping is:" | "mapping is:"
11: "•  background  /  foreground  → direct" | "   - `background` / `foreground` → direct"
12: "• palette 0 → Selected, StatusBar, Code" | "   - palette 0 → Selected, StatusBar, Code"
13: "• palette 1 → Search (red)" | "   - palette 1 → Search (red)"
14: "• palette 2 → Heading2 (green)" | "   - palette 2 → Heading2 (green)"
15: "• palette 3 → Heading1 (yellow)" | "   - palette 3 → Heading1 (yellow)"
16: "• palette 4 → Heading3 (blue)" | "   - palette 4 → Heading3 (blue)"
17: "• palette 6 → NodeSel (cyan)" | "   - palette 6 → NodeSel (cyan)"
18: "• palette 8 → Border, HeadingN (bright black)" | "   - palette 8 → Border, HeadingN (bright black)"
19: "• palette 11 → Highlight (bright yellow)" | "   - palette 11 → Highlight (bright yellow)"
20: "3. Color overrides — any non-empty field in  [ui.theme_override]  replaces" | "3. **Color overrides** — any non-empty field in `[ui.theme_override]` replaces"
21: "the corresponding color, regardless of whether the base theme is built-in or" | "the corresponding color, regardless of whether the base theme is built-in or"
22: "loaded from Ghostty." | "loaded from Ghostty."
24: "If the theme name doesn't match a built-in and no Ghostty directory is" | "If the theme name doesn't match a built-in and no Ghostty directory is"
25: "configured (or the file isn't found), gomd falls back to OceanDark." | "configured (or the file isn't found), gomd falls back to OceanDark."
######## section 12
== rendered

  ## Input resolution

  When no file argument is given,  gomd  reads from stdin if piped, otherwise
  prints help and exits.

  ### Images

  gomd can render inline images using the Kitty graphics protocol
  https://sw.kovidgoyal.net/kitty/graphics-protocol/. This works in terminals
  that support it: Ghostty, Kitty, and WezTerm.

  Image rendering is controlled by three settings, evaluated in this order:

  1.  --no-images  flag: always disables images (highest priority).
  2.  --images  flag: always enables images, even if the terminal can't be
  auto-detected (e.g. inside tmux).
  3. Config file ( ~/.config/gomd/config.toml ): the  [images]  section sets
  the default.

    [images]
    enabled = true   # default

  When none of the flags are passed,  enabled  from the config is used. If
  enabled = true , gomd auto-detects whether the terminal supports Kitty
  graphics (by checking  TERM_PROGRAM  and related env vars). If detection
  succeeds, images are rendered automatically.

  tmux: inside tmux, terminal detection fails because  TERM_PROGRAM  is  tmux
  . Use  --images  to force image rendering. Images are sent via tmux's DCS
  passthrough (requires  allow-passthrough on  in your tmux config). Note:
  images rendered inside tmux are "sticky" — they remain painted on screen
  even when scrolling or switching tmux windows, because tmux manages text
  cells but not graphics placements. Run  reset  in the terminal to clear
  them. This is a known limitation of tmux's passthrough mechanism.

  Only local image files are supported (relative or absolute paths). URLs are
  not fetched — they display as text placeholders.
== nodes
inline src 2-2 col 32-38  shown 3-3 col 35-39  "gomd"
link   src 6-6 col 40-117  shown 8-8 col 42-65  "Kitty graphics protocol -> https://sw.kovidgoyal.net/kitty/graphics-protocol/"
inline src 10-10 col 5-18  shown 14-14 col 6-17  "--no-images"
inline src 11-11 col 5-15  shown 15-15 col 6-14  "--images"
inline src 12-12 col 20-48  shown 17-17 col 19-45  "~/.config/gomd/config.toml"
inline src 12-12 col 55-65  shown 17-17 col 54-62  "[images]"
block  src 14-17 col 0-0  shown 20-21 col -1--1  "[images]\nenabled = true   # default"
inline src 19-19 col 35-44  shown 23-23 col 38-45  "enabled"
inline src 19-19 col 73-89  shown 24-24 col 2-16  "enabled = true"
inline src 19-19 col 167-181  shown 25-25 col 25-37  "TERM_PROGRAM"
inline src 21-21 col 56-70  shown 28-28 col 55-67  "TERM_PROGRAM"
inline src 21-21 col 74-80  shown 28-28 col 73-77  "tmux"
inline src 21-21 col 86-96  shown 29-29 col 9-17  "--images"
inline src 21-21 col 176-198  shown 30-30 col 25-45  "allow-passthrough on"
inline src 21-21 col 414-421  shown 33-33 col 42-47  "reset"
== copy per line
1: "## Input resolution" | "## Input resolution"
3: "When no file argument is given,  gomd  reads from stdin if piped, otherwise" | "When no file argument is given, `gomd` reads from stdin if piped, otherwise"
4: "prints help and exits." | "prints help and exits."
6: "### Images" | "### Images"
8: "gomd can render inline images using the Kitty graphics protocol\nhttps://sw.kovidgoyal.net/kitty/graphics-protocol/" | "gomd can render inline images using the [Kitty graphics protocol](https://sw.kovidgoyal.net/kitty/graphics-protocol/)"
9: "Kitty graphics protocol\nhttps://sw.kovidgoyal.net/kitty/graphics-protocol/. This works in terminals" | "[Kitty graphics protocol](https://sw.kovidgoyal.net/kitty/graphics-protocol/). This works in terminals"
10: "that support it: Ghostty, Kitty, and WezTerm." | "that support it: **Ghostty**, **Kitty**, and **WezTerm**."
12: "Image rendering is controlled by three settings, evaluated in this order:" | "Image rendering is controlled by three settings, evaluated in this order:"
14: "1.  --no-images  flag: always disables images (highest priority)." | "1. **`--no-images`** flag: always disables images (highest priority)."
15: "2.  --images  flag: always enables images, even if the terminal can't be" | "2. **`--images`** flag: always enables images, even if the terminal can't be"
16: "auto-detected (e.g. inside tmux)." | "auto-detected (e.g. inside tmux)."
17: "3. Config file ( ~/.config/gomd/config.toml ): the  [images]  section sets" | "3. **Config file** (`~/.config/gomd/config.toml`): the `[images]` section sets"
18: "the default." | "the default."
20: "[images]" | "[images]"
21: "enabled = true   # default" | "enabled = true   # default"
23: "When none of the flags are passed,  enabled  from the config is used. If" | "When none of the flags are passed, `enabled` from the config is used. If"
24: "enabled = true , gomd auto-detects whether the terminal supports Kitty" | "`enabled = true`, gomd auto-detects whether the terminal supports Kitty"
25: "graphics (by checking  TERM_PROGRAM  and related env vars). If detection" | "graphics (by checking `TERM_PROGRAM` and related env vars). If detection"
26: "succeeds, images are rendered automatically." | "succeeds, images are rendered automatically."
28: "tmux: inside tmux, terminal detection fails because  TERM_PROGRAM  is  tmux" | "**tmux**: inside tmux, terminal detection fails because `TERM_PROGRAM` is `tmux`"
29: ". Use  --images  to force image rendering. Images are sent via tmux's DCS" | ". Use `--images` to force image rendering. Images are sent via tmux's DCS"
30: "passthrough (requires  allow-passthrough on  in your tmux config). Note:" | "passthrough (requires `allow-passthrough on` in your tmux config). Note:"
31: "images rendered inside tmux are \"sticky\" — they remain painted on screen" | "images rendered inside tmux are \"sticky\" — they remain painted on screen"
32: "even when scrolling or switching tmux windows, because tmux manages text" | "even when scrolling or switching tmux windows, because tmux manages text"
33: "cells but not graphics placements. Run  reset  in the terminal to clear" | "cells but not graphics placements. Run `reset` in the terminal to clear"
34: "them. This is a known limitation of tmux's passthrough mechanism." | "them. This is a known limitation of tmux's passthrough mechanism."
36: "Only local image files are supported (relative or absolute paths). URLs are" | "Only local image files are supported (relative or absolute paths). URLs are"
37: "not fetched — they display as text placeholders." | "not fetched — they display as text placeholders."
######## section 13
== rendered

  ### Images

  gomd can render inline images using the Kitty graphics protocol
  https://sw.kovidgoyal.net/kitty/graphics-protocol/. This works in terminals
  that support it: Ghostty, Kitty, and WezTerm.

  Image rendering is controlled by three settings, evaluated in this order:

  1.  --no-images  flag: always disables images (highest priority).
  2.  --images  flag: always enables images, even if the terminal can't be
  auto-detected (e.g. inside tmux).
  3. Config file ( ~/.config/gomd/config.toml ): the  [images]  section sets
  the default.

    [images]
    enabled = true   # default

  When none of the flags are passed,  enabled  from the config is used. If
  enabled = true , gomd auto-detects whether the terminal supports Kitty
  graphics (by checking  TERM_PROGRAM  and related env vars). If detection
  succeeds, images are rendered automatically.

  tmux: inside tmux, terminal detection fails because  TERM_PROGRAM  is  tmux
  . Use  --images  to force image rendering. Images are sent via tmux's DCS
  passthrough (requires  allow-passthrough on  in your tmux config). Note:
  images rendered inside tmux are "sticky" — they remain painted on screen
  even when scrolling or switching tmux windows, because tmux manages text
  cells but not graphics placements. Run  reset  in the terminal to clear
  them. This is a known limitation of tmux's passthrough mechanism.

  Only local image files are supported (relative or absolute paths). URLs are
  not fetched — they display as text placeholders.
== nodes
link   src 2-2 col 40-117  shown 3-3 col 42-65  "Kitty graphics protocol -> https://sw.kovidgoyal.net/kitty/graphics-protocol/"
inline src 6-6 col 5-18  shown 9-9 col 6-17  "--no-images"
inline src 7-7 col 5-15  shown 10-10 col 6-14  "--images"
inline src 8-8 col 20-48  shown 12-12 col 19-45  "~/.config/gomd/config.toml"
inline src 8-8 col 55-65  shown 12-12 col 54-62  "[images]"
block  src 10-13 col 0-0  shown 15-16 col -1--1  "[images]\nenabled = true   # default"
inline src 15-15 col 35-44  shown 18-18 col 38-45  "enabled"
inline src 15-15 col 73-89  shown 19-19 col 2-16  "enabled = true"
inline src 15-15 col 167-181  shown 20-20 col 25-37  "TERM_PROGRAM"
inline src 17-17 col 56-70  shown 23-23 col 55-67  "TERM_PROGRAM"
inline src 17-17 col 74-80  shown 23-23 col 73-77  "tmux"
inline src 17-17 col 86-96  shown 24-24 col 9-17  "--images"
inline src 17-17 col 176-198  shown 25-25 col 25-45  "allow-passthrough on"
inline src 17-17 col 414-421  shown 28-28 col 42-47  "reset"
== copy per line
1: "### Images" | "### Images"
3: "gomd can render inline images using the Kitty graphics protocol\nhttps://sw.kovidgoyal.net/kitty/graphics-protocol/" | "gomd can render inline images using the [Kitty graphics protocol](https://sw.kovidgoyal.net/kitty/graphics-protocol/)"
4: "Kitty graphics protocol\nhttps://sw.kovidgoyal.net/kitty/graphics-protocol/. This works in terminals" | "[Kitty graphics protocol](https://sw.kovidgoyal.net/kitty/graphics-protocol/). This works in terminals"
5: "that support it: Ghostty, Kitty, and WezTerm." | "that support it: **Ghostty**, **Kitty**, and **WezTerm**."
7: "Image rendering is controlled by three settings, evaluated in this order:" | "Image rendering is controlled by three settings, evaluated in this order:"
9: "1.  --no-images  flag: always disables images (highest priority)." | "1. **`--no-images`** flag: always disables images (highest priority)."
10: "2.  --images  flag: always enables images, even if the terminal can't be" | "2. **`--images`** flag: always enables images, even if the terminal can't be"
11: "auto-detected (e.g. inside tmux)." | "auto-detected (e.g. inside tmux)."
12: "3. Config file ( ~/.config/gomd/config.toml ): the  [images]  section sets" | "3. **Config file** (`~/.config/gomd/config.toml`): the `[images]` section sets"
13: "the default." | "the default."
15: "[images]" | "[images]"
16: "enabled = true   # default" | "enabled = true   # default"
18: "When none of the flags are passed,  enabled  from the config is used. If" | "When none of the flags are passed, `enabled` from the config is used. If"
19: "enabled = true , gomd auto-detects whether the terminal supports Kitty" | "`enabled = true`, gomd auto-detects whether the terminal supports Kitty"
20: "graphics (by checking  TERM_PROGRAM  and related env vars). If detection" | "graphics (by checking `TERM_PROGRAM` and related env vars). If detection"
21: "succeeds, images are rendered automatically." | "succeeds, images are rendered automatically."
23: "tmux: inside tmux, terminal detection fails because  TERM_PROGRAM  is  tmux" | "**tmux**: inside tmux, terminal detection fails because `TERM_PROGRAM` is `tmux`"
24: ". Use  --images  to force image rendering. Images are sent via tmux's DCS" | ". Use `--images` to force image rendering. Images are sent via tmux's DCS"
25: "passthrough (requires  allow-passthrough on  in your tmux config). Note:" | "passthrough (requires `allow-passthrough on` in your tmux config). Note:"
26: "images rendered inside tmux are \"sticky\" — they remain painted on screen" | "images rendered inside tmux are \"sticky\" — they remain painted on screen"
27: "even when scrolling or switching tmux windows, because tmux manages text" | "even when scrolling or switching tmux windows, because tmux manages text"
28: "cells but not graphics placements. Run  reset  in the terminal to clear" | "cells but not graphics placements. Run `reset` in the terminal to clear"
29: "them. This is a known limitation of tmux's passthrough mechanism." | "them. This is a known limitation of tmux's passthrough mechanism."
31: "Only local image files are supported (relative or absolute paths). URLs are" | "Only local image files are supported (relative or absolute paths). URLs are"
32: "not fetched — they display as text placeholders." | "not fetched — they display as text placeholders."
######## section 14
== rendered

  ## License

  MIT
== nodes
== copy per line
1: "## License" | "## License"
3: "MIT" | "MIT"
