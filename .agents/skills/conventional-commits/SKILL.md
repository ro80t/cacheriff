---
name: conventional-commits
description: Write commit messages in Conventional Commits format for cacheriff. Use whenever creating a git commit in this repository.
---

# Conventional Commits

Format: `<type>[optional scope]: <description>`

Types used in this repo's history: `feat`, `fix`, `refactor`, `docs`,
`config`, `chore`. Pick the one that matches the change; don't invent new
types.

- `feat` — new behavior (a driver, a UI feature, a config option)
- `fix` — bug fix
- `refactor` — code change with no behavior change
- `docs` — README/CONTRIBUTING/AGENTS.md/comments only
- `config` — config file handling/schema changes
- `chore` — everything else (deps, CI, tooling)

Optional scope in parentheses when it narrows the change to one area, e.g.
`feat(driver): add homebrew support`, `fix(ui): ...`. Skip it when the
type alone is clear.

Description: imperative mood, lowercase, no trailing period — "add x", not
"added x" or "adds x".

Breaking change: `!` after the type/scope (`feat!: ...`) or a `BREAKING
CHANGE:` footer, whichever fits the amount of explanation needed.

## Body

Only add a body when the *why* isn't obvious from the diff — same bar as
code comments in this repo (see `AGENTS.md`). A one-line `feat: add bun
driver` needs nothing more. Reserve the body for the reasoning a reviewer
couldn't get from the diff alone.

## What not to do

- Don't title-case or capitalize the description (`feat: Add X` → `feat:
  add x`).
- Don't bundle unrelated changes under one type; split into separate
  commits instead.
- Don't use `feat` for things that aren't user/API-facing — a pure
  internal refactor is `refactor`, even if it happens to touch a lot of
  files.
