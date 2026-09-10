---
name: go-mod-dependencies
description: Managing Go module dependencies in cacheriff (go.mod/go.sum). Use when adding, updating, or removing a dependency, or investigating a build/version mismatch.
---

# Go module dependencies

cacheriff's module is intentionally small: `go.mod` currently has four
direct dependencies —
[`bubbles`](https://github.com/charmbracelet/bubbles),
[`bubbletea`](https://github.com/charmbracelet/bubbletea),
[`lipgloss`](https://github.com/charmbracelet/lipgloss) (the Charm TUI
stack, see `.agents/skills/bubbletea-tui` and
`.agents/skills/lipgloss-styling`), and
[`goccy/go-yaml`](https://github.com/goccy/go-yaml) (config file parsing,
`internal/config`). Everything else in `go.mod` is `// indirect` —
transitive dependencies of those four.

## Adding a dependency

1. Ask whether it's actually needed — this project favors the standard
   library (`internal/textwrap`, `internal/platform`, and most of
   `internal/driver` use nothing beyond it). Only add a module for real
   leverage (a TUI framework, a YAML parser), not for something a handful
   of stdlib lines would cover.
2. `go get <module>@<version>` from the repo root.
3. `go mod tidy` to update `go.sum` and prune unused indirect entries.
4. Run the full check sequence (`.agents/skills/pre-pr-check`) — a new
   dependency can shift `go vet` or introduce build tag issues.

## Updating a dependency

- `go get <module>@latest` (or a specific version) then `go mod tidy`.
- Dependabot already opens weekly PRs for both `gomod` and
  `github-actions` ecosystems (`.github/dependabot.yml`), grouping all Go
  dependency updates into one PR (`groups.go-dependencies.patterns: ["*"]`).
  Prefer reviewing/merging those over manually bumping versions unless
  something is urgent.
- After any bump, re-run `.agents/skills/pre-pr-check` — the Charm
  libraries in particular have had breaking changes across major versions.

## Removing a dependency

1. Remove the import and all usages.
2. `go mod tidy` — it will drop the now-unused entry from `go.mod`/`go.sum`
   automatically. Don't hand-edit `go.sum`.

## `go.mod`'s `go` directive

`go 1.25.13` pins the toolchain version. CI resolves its Go version from
this file (`go-version-file: go.mod` in `.github/workflows/ci.yml`), so
bumping it also bumps what CI builds/tests with — do that deliberately, not
as a side effect of an unrelated change.

## Checking what changed

`go mod tidy` and `go mod verify` are safe, read-mostly operations. Use
`go list -m all` to see the full resolved dependency graph, or `go mod why
<module>` to see why something transitive is pulled in — useful before
deciding whether an indirect dependency bump actually matters.
