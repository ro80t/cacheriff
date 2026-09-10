---
name: go-development
description: Go language conventions used in cacheriff (error handling, testing, concurrency, comments). Use when writing or reviewing Go code anywhere in this repository.
---

# Go development conventions

cacheriff is a small, dependency-light Go CLI/TUI (module `cacheriff`, Go
1.25.13). These are the conventions actually followed in this codebase —
match them rather than defaulting to generic Go style.

## Comments

Default to none. Only add a comment when the *why* isn't derivable from the
code — a hidden constraint, a CLI quirk being worked around, a security
invariant. Don't restate what a well-named function/type already says. See
`AGENTS.md` for the fuller rationale and a catalog of non-obvious behavior
that used to be inline comments.

## Error handling

- Wrap errors with `fmt.Errorf("%s: %w", context, err)`, always including
  enough context to identify *what* failed (a command line, a file path) —
  see `internal/driver/base.go`'s `runOutput`/`runCombined` and `cmdLine`.
- A missing file is often not an error — see `config.Load()` treating
  `os.IsNotExist` as "use defaults", and `singleDirCacheEntries` returning
  `(nil, nil)` when a cache directory doesn't exist yet. Distinguish "isn't
  there yet" from "failed to read" using `os.IsNotExist`.
- Prefer a sentinel return value that's part of the type's normal contract
  over an error when absence is expected: `Entry.Size == -1` means
  "unknown", not an error; `LocalPackages` returns `(nil, nil)` when a
  package manager has nothing to report for a project, not an error.

## Testing

- Table-driven tests are the norm (`tests := []struct{...}{...}` then a
  `for _, tt := range tests` loop) — see any `*_test.go` in
  `internal/driver` or `internal/textwrap`.
- Use real, captured CLI output as test fixtures (e.g. actual `npm ls -g
  --json` or `cargo install --list` output), not synthetic-looking data —
  it catches format assumptions that made-up data wouldn't.
- Use `t.TempDir()` and `t.Setenv()` for filesystem/environment isolation
  rather than manual cleanup.
- New behavior needs test coverage; see `.agents/skills/pre-pr-check` for
  the full verification sequence.

## Concurrency

- Sizing multiple cache directories is done concurrently with
  `sync.WaitGroup` (see `base.go`'s `sizeCacheDirs`) because some caches
  hold huge numbers of small files and walking them serially is slow. Use
  the same pattern (goroutine per independent unit, `wg.Wait()`, write into
  pre-sized slices by index to avoid a mutex) for similar "size/fetch N
  independent things" work.
- Long filesystem walks check `ctx.Err()` on every entry (`dirSize`) so a
  caller's timeout aborts promptly instead of running to completion
  regardless of the deadline.
- Background work in the UI runs as a `tea.Cmd` closure with its own
  `context.WithTimeout`, tagged with a monotonically increasing generation
  counter so a stale result can be discarded if the user has since moved
  on — see `.agents/skills/bubbletea-tui`.

## Security

- Any user-controlled or driver-reported string (a package name) that
  becomes a CLI argument must go through `textwrap.EscapeArg` first — see
  `internal/textwrap/escape.go` and the "Security" section of `AGENTS.md`
  for why (argument injection, not shell injection, is the actual risk).
- Never build a command line as a single shell string; always pass args as
  a `[]string` to `exec.CommandContext`.

## Style

- `gofmt` is enforced by CI — run `gofmt -w .` before finishing, not by
  hand-formatting.
- `go vet` must be clean.
- Keep drivers and UI code free of framework-specific abstractions beyond
  what's needed; this repo favors small structs with embedding (`base`)
  over interfaces-for-their-own-sake.

See also `.agents/skills/pre-pr-check` (verification loop),
`.agents/skills/go-mod-dependencies` (dependency changes),
`.agents/skills/bubbletea-tui` and `.agents/skills/lipgloss-styling` (the
`internal/ui` layer).
