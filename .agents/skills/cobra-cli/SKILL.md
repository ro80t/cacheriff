---
name: cobra-cli
description: Cobra CLI setup in cacheriff (main.go). Use when adding flags, subcommands, or touching --help/--version output.
---

# Cobra CLI in cacheriff

`main.go` wraps the TUI launch in a single [Cobra](https://github.com/spf13/cobra)
root command (`Use: "cacheriff"`). Cobra buys `--help` and `--version`
generation for free; there's no subcommand tree today because cacheriff has
exactly one thing to do — open the TUI.

## Layout

- `main()` builds the `*cobra.Command`, then calls `root.Execute()`.
- `run(cmd *cobra.Command, args []string) error` holds the actual behavior
  (config load, `ui.SetTheme`, `tea.NewProgram(...).Run()`) and is wired up
  as `RunE`, so a version bump or a future subcommand doesn't disturb it.
- `version` is a package-level `var` (default `"dev"`), overridable at
  build/release time via `-ldflags "-X main.version=..."`. Nothing in this
  repo sets that flag yet (see `.agents/skills/github-actions-ci` before
  wiring a release build that does).
- `SilenceUsage: true` keeps a runtime TUI error from also dumping the
  flag usage block — that's only useful for a flag-parsing error, which
  Cobra already prints on its own.

## Adding a flag or subcommand

Only add one when it does something the interactive TUI can't — e.g. a
non-interactive `list`/`clean` mode. Don't add a subcommand just to mirror
a TUI keybinding; that's what the TUI is for. A new flag goes on `root`
via `root.Flags()`/`root.PersistentFlags()` and is read inside `run`.

## Checking it

`go build -o cacheriff.exe .` then `cacheriff.exe --help` and `--version`.
Cobra's own flag/help handling doesn't need app-level tests; only test
logic you write inside `run` or a new subcommand's `RunE`.
