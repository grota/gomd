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

## Known bugs

- [ ] **Link URLs with some non-ASCII characters break the rendering.**
      Glamour writes a link's URL raw into an OSC 8 hyperlink.
      `charmbracelet/x/ansi` (v0.11.7, still in v0.11.9) reads byte 0x9C as
      the C1 string terminator even inside UTF-8, so a URL holding "✓"
      (E2 9C 93), Cyrillic "М" (D0 9C) or Greek "Μ" (CE 9C) ends the escape
      early: the hyperlink target is corrupt and the rest of the URL shows
      as text, repeated by glamour's word wrap. The source map then loses its
      place for the rest of the document (gopls `doc/contributing.md`: 41 of
      its nodes).

      **Decision: wait for upstream**, then bump `x/ansi` and rerun the corpus
      benchmark (with the fix, misses dropped from 242 to 160). Upstream
      issue: https://github.com/charmbracelet/x/issues/848. Fix PR:
      https://github.com/charmbracelet/x/pull/946, which covers only
      `Parser`; our commit extending it to `Strip`, `Wrap`, `Truncate`,
      `DecodeSequence` and the rest is
      https://github.com/grota/x/commit/3498cdf146b20a2f108b1f2a6e52955277627761
      (offered in
      https://github.com/charmbracelet/x/pull/946#issuecomment-6533146504).
      Check that whatever merges covers those functions. A `replace`
      directive is not an option: `go install ...@latest` rejects it. A gomd
      workaround would percent-encode those characters in link destinations
      before rendering.

## Unconfirmed checks

Earlier notes left these open, and nothing records a result.

- [ ] **Ghostty upstream issue.** Ghostty 1.3.1 answers DSR 996 with "light"
      whenever its config uses a single theme, because
      `Config.changeConditionalState` returns early. Search for an existing
      issue or open one. The workaround is a `light:A,dark:B` theme pair in the
      Ghostty config; with it, gomd switches live in an attached tmux
      (confirmed 2026-10-13).
