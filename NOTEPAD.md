# gomd notepad

Open work only. Finished items are removed; git history has the details.

## Potential improvements

- [ ] **Syntax colours from each theme's terminal palette.** Fenced code uses
      glamour's fixed dark and light palettes. `ensureContrast` in
      `internal/tui/theme.go` pushes each colour towards black or white until it
      reaches 4.5:1 on the code background (3:1 for comments). Mapping tokens to
      the theme's ANSI colours (blue keywords, green strings, cyan numbers)
      would make code match the theme, with `ensureContrast` kept as the safety
      net. `Theme` holds only some palette slots and no magenta, so Ghostty
      themes need the full palette kept and the built-in Ocean themes need a
      hand-picked one. Decide after test-driving the 2026-10-13 colour changes.

## Unconfirmed checks

Earlier notes left these open, and nothing records a result.

- [ ] **Ghostty upstream issue.** Ghostty 1.3.1 answers DSR 996 with "light"
      whenever its config uses a single theme, because
      `Config.changeConditionalState` returns early. Search for an existing
      issue or open one. The workaround is a `light:A,dark:B` theme pair in the
      Ghostty config; with it, gomd switches live in an attached tmux
      (confirmed 2026-10-13).
