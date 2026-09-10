# Contributing to cacheriff

Thanks for your interest in contributing to cacheriff! This document covers how to set up your development environment and the guidelines for submitting changes.

## Development setup

Go 1.25.13 or later is required (see `go.mod`).

```sh
git clone https://github.com/<your-fork>/cacheriff.git
cd cacheriff
go build ./...
```

## Build, test, and run

```sh
go build ./...               # build
go vet ./...                  # static analysis
go test ./... -race -cover    # tests
gofmt -l .                     # formatting check (fix with gofmt -w .)
go run .                       # run the TUI
```

These are the same checks run in CI (`.github/workflows/ci.yml`), so make sure they pass locally before opening a PR.

## Before submitting a pull request

- `gofmt -l .` reports no differences
- `go vet ./...` passes with no warnings
- `go test ./... -race -cover` passes
- New behavior is covered by tests
- Commit messages clearly describe the change

## Code layout

- `internal/driver` — driver implementations for each supported package manager (npm, pnpm, yarn, bun, deno, cargo, go). Shared logic lives in `base.go`; the interface is defined in `driver.go`.
- `internal/ui` — the Bubble Tea-based terminal UI (model, view, key bindings).
- `internal/theme` — color scheme definitions. Values are plain hex strings with no dependency on the rendering library.
- `internal/config` — loading of user configuration, matching `config.example.yml`.
- `internal/platform` — OS detection and other platform-specific logic.
- `internal/textwrap` — text wrapping and escaping for TUI rendering.

## Adding a new package manager driver

1. Create a new file under `internal/driver` (e.g. `foo.go`) that embeds `base` and implements the `driver.Driver` interface. Use an existing implementation (`npm.go`, `cargo.go`, etc.) as a reference.
2. Register the new driver's constructor in `All()` in `driver.go`.
3. Add corresponding tests (`foo_test.go`).
4. Make sure `Available()` can be called safely even when the underlying CLI isn't installed.

## Reporting issues

Please include reproduction steps, expected behavior, actual behavior, and your OS/Go version.

## License

By contributing to cacheriff, you agree that your contributions will be licensed under the MIT License (see `LICENSE`).
