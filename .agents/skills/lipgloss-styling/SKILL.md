---
name: lipgloss-styling
description: lipgloss styling and theming conventions used in cacheriff (internal/ui/styles.go, internal/theme). Use when adding or changing styles, colors, borders, or panel layout.
---

# lipgloss styling in cacheriff

Styling lives in two layers: `internal/theme` (pure color data, no
lipgloss dependency) and `internal/ui/styles.go` (lipgloss `Style` values
built from that theme). Rendering that uses these styles is spread across
`internal/ui/view.go` — see `.agents/skills/bubbletea-tui` for the
Model/View side.

## Theme vs styles

- `internal/theme.Theme` holds plain hex-string color roles (`Primary`,
  `ActiveBorder`, `InactiveBorder`, `Muted`, `Faint`, `Error`, `Success`)
  with **no** import of lipgloss — deliberately, so the same struct could
  back a non-lipgloss renderer later (the package doc mentions a possible
  Neovim port via `guifg`/`guibg`). Don't add a lipgloss type to this
  package.
- `internal/ui/styles.go` converts those hex strings to `lipgloss.Color`
  and builds every `lipgloss.Style` used by the UI, all inside `SetTheme`.
  `SetTheme` must run exactly once, before the program starts (`main.go`
  calls it right after loading config) — styles are package-level `var`s,
  not recomputed per-render.
- A user's `internal/config` override (`gui.theme` in their YAML config)
  is applied via `theme.Override.Apply(theme.Default)` before `SetTheme`
  runs — see `internal/config/config.go`'s `Config.Theme()`.

## Adding a new style

1. If it needs a new *semantic* color role (not just a new combination of
   existing ones), add a field to `theme.Theme` and `theme.Override`
   (both, field-for-field — `Override.Apply` also needs a case for it),
   and document what it's for in the field's doc comment (these *are*
   worth documenting, since they're a public contract with user config
   files).
2. Build the `lipgloss.Style` in `SetTheme` in `styles.go`, following the
   existing pattern (`lipgloss.NewStyle().Foreground(colorX)...`).
3. Reference it from `view.go`; don't construct one-off
   `lipgloss.NewStyle()` calls scattered through render functions unless
   truly one-off (the confirmation modal's inline styles in
   `renderConfirmModal` are the accepted exception, since they're
   error-colored and used nowhere else).

## Panels, width, and height — known gotchas

- `panelStyle` (the shared base for every bordered panel) uses
  `Border(lipgloss.RoundedBorder())` and `Padding(0, 1)` — horizontal
  padding only, no vertical. That asymmetry matters for layout math: see
  `view.go`'s `computeLayout`, where **widths** subtract border + padding
  (4 columns) but **heights** only subtract the border (2 rows).
- `Style.Width()` already accounts for the style's own horizontal padding
  internally — don't subtract padding again when computing the value you
  pass to `.Width()`.
- **lipgloss does not clip overflow.** `Style.Height()` sets a minimum,
  not a maximum — a line wider than the panel's content width will grow
  the rendered box past its declared height and misalign it against
  neighboring panels (`lipgloss.JoinHorizontal`/`JoinVertical` don't fix
  this after the fact). Always pre-wrap text to the real content width
  with `textwrap.ContentLine` before handing it to a styled box; see
  `cacheRowLines`/`packageRowLines` in `view.go` for the pattern.

## Composing layout

Use `lipgloss.JoinHorizontal`/`JoinVertical` to combine already-rendered
blocks (sidebar + main panel, header + body + footer) rather than
manually padding strings. Use `lipgloss.Place` for centering an overlay
like the confirm modal over the existing body.

## Colors

`lipgloss.Color(hex)` takes a plain hex string. This repo's default theme
targets a lazygit-like minimal palette (white/green/gray/red — see
`theme.Default`); a user's config can override any subset via
`gui.theme.*Color` keys. When picking a new default color, keep the
palette small and check it reads clearly against both the sidebar's
unfocused-gray and the focused-green border, not just in isolation.
