---
name: github-actions-ci
description: cacheriff's GitHub Actions CI workflow and Dependabot config. Use when modifying .github/workflows, adding a CI job or step, or diagnosing a CI failure.
---

# GitHub Actions CI

## Current workflow

`.github/workflows/ci.yml` defines a single job, `test` (display name
"Build, Vet & Test"), on `ubuntu-latest`, triggered on push and pull
request to `main`. Steps, in order:

1. `actions/checkout@v7`
2. `actions/setup-go@v7` with `go-version-file: go.mod` (so the Go version
   tracks `go.mod`'s `go` directive — see `.agents/skills/go-mod-dependencies`)
   and `cache: true`
3. `gofmt -l .` — fails the build if it prints any file names
4. `go vet ./...`
5. `go build ./...`
6. `go test ./... -race -cover`

`permissions: contents: read` is set at the workflow level — the minimum
needed to check out the repo. Don't broaden it unless a step genuinely
needs more (e.g. writing PR comments).

This is exactly the sequence `.agents/skills/pre-pr-check` runs locally —
the two must stay in sync. If you change one, change the other.

## Dependabot

`.github/dependabot.yml` runs two weekly update checks:
- `gomod` at `/` — Go module updates, grouped into one PR
  (`groups.go-dependencies.patterns: ["*"]`) rather than one PR per
  dependency, capped at 10 open PRs.
- `github-actions` at `/` — keeps the workflow's own action versions
  (`actions/checkout`, `actions/setup-go`) current.

## Adding a CI step

- Keep it fast and deterministic — this repo has no integration tests that
  need network access or external services; don't add either without a
  strong reason.
- Put a new check where it fails fastest: formatting/linting before
  build, build before test.
- If a step is Windows-specific behavior worth testing (this project cares
  about Windows: see `platform.Windows`, the `.exe` fallback in
  `cargoBinSize`, path separator handling in `internal/textwrap`), consider
  a matrix (`strategy.matrix.os`) rather than assuming `ubuntu-latest`
  coverage is sufficient — but check with a human before expanding CI
  scope/cost.

## Diagnosing a CI failure

1. Reproduce locally first with the exact same commands
   (`.agents/skills/pre-pr-check`) — nearly everything CI does is
   reproducible without pushing.
2. `gofmt -l .` failures: run `gofmt -w .`, don't hand-fix whitespace.
3. `go vet` failures: fix the underlying issue; don't suppress.
4. Flaky `-race` failures point at a real data race — see the concurrency
   patterns in `.agents/skills/go-development` for how this repo avoids
   them (indexed writes into pre-sized slices, no shared mutable state
   across goroutines without a `sync.WaitGroup` boundary).
5. Use `gh run list` / `gh run view --log-failed` to inspect a run that
   already happened on GitHub without leaving the terminal.
