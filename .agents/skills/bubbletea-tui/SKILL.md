---
name: bubbletea-tui
description: Bubble Tea patterns used in cacheriff's TUI (internal/ui). Use when adding UI state, key bindings, async commands, or otherwise touching the Model/Update/View flow.
---

# Bubble Tea patterns in cacheriff

`internal/ui` is a single `Model` (`model.go`) built on
[Bubble Tea](https://github.com/charmbracelet/bubbletea), using
`bubbles/key`, `bubbles/help`, `bubbles/spinner`, and `bubbles/viewport`
from the same Charm ecosystem. Styling is layered on top with lipgloss —
see `.agents/skills/lipgloss-styling`.

## File layout

- `model.go` — the `Model` struct, `Init`/`Update`, and all key-handling
  (`handleKey` dispatches to `handleSidebarKey`/`handleMainKey`/
  `handleConfirmKey` depending on focus/modal state).
- `view.go` — `View()` and all rendering (layout math, panel rendering,
  the confirmation modal).
- `keys.go` — the `keyMap` (implements `help.KeyMap` via `ShortHelp`/
  `FullHelp`).
- `messages.go` — `tea.Msg` types returned by async commands.
- `commands.go` — the `tea.Cmd` closures that do background work (driver
  calls, uninstall).
- `styles.go` — lipgloss styles, theme-driven (see
  `.agents/skills/lipgloss-styling`).

## Adding UI state

- New persistent state goes on `Model` as a field, mutated only through
  value-receiver methods that return a new `Model` (Bubble Tea's usual
  immutable-update convention) — see `setScope`, `movePackageCursor`.
  `ensureCursorVisible` is the one exception (pointer receiver) because it
  only needs to touch the viewport's scroll offset, not swap the whole
  model.
- If the new state affects what's rendered, also update
  `m.viewport.SetContent(m.renderMainContent())` at the point where the
  state changes — the viewport's content isn't automatically kept in sync
  with `Model` fields.

## Async work: the `gen` counter pattern

Anything that shells out to a driver runs in the background as a
`tea.Cmd` (see `loadDriverDataCmd`, `removePackageCmd` in `commands.go`),
not inline in `Update`, since Bubble Tea's `Update` must return quickly.

Each such command is tagged with a `gen int` that increments every time
the user starts a new one of that kind (`m.loadGen++`, `m.removeGen++`).
The resulting message (`driverDataMsg`, `packageRemovedMsg` in
`messages.go`) carries that same `gen`. In `Update`, compare
`msg.gen != m.loadGen` (or `removeGen`) and discard the message if it
doesn't match — that's what makes it safe for the user to select a
different driver mid-load without a stale result clobbering the new
selection. Follow this pattern for any new async operation; don't assume
only one can be in flight.

Each async `tea.Cmd` gets its own `context.WithTimeout` (`loadTimeout` =
2 minutes) and a stored `context.CancelFunc` (`loadCancel`/`removeCancel`)
so starting a new operation, or quitting, can cancel an in-flight one —
see `selectDriver` and the `Quit`/`ctrl+c` handling in `handleKey`/
`handleConfirmKey`.

## Key bindings

Add new bindings to `keyMap` in `keys.go` following the existing
`key.NewBinding(key.WithKeys(...), key.WithHelp(...))` shape, then wire
them into `ShortHelp`/`FullHelp` so they show up in the help bar (`?` to
toggle `m.help.ShowAll`). Dispatch them in the appropriate `handle*Key`
method based on `m.focus` — don't add a case to the top-level `handleKey`
switch unless the binding should work regardless of which panel is
focused (like `Tab`, `Help`, `Quit` do).

## Layout and text wrapping

`view.go`'s `computeLayout` is the single source of truth for panel
pixel budgets; both `applyLayout` (sizes the viewport) and `View` (renders
panels) read from it — don't recompute widths/heights ad hoc elsewhere.
Any text rendered into a bordered panel must go through
`textwrap.ContentLine` to the panel's real content width first (see
`.agents/skills/lipgloss-styling` for why lipgloss won't do this for you).

## Verifying a UI change

There's no headless test harness for the TUI in this repo — after a
change, run it (`go run .` or `.agents/skills/pre-pr-check` first) and
exercise the affected panel/keys manually, since `go build`/`go vet`/
`go test` won't catch a layout or rendering regression.
