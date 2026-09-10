---
name: pre-pr-check
description: Run cacheriff's CI checks locally (gofmt, go vet, go build, go test) before considering a change finished or opening a PR. Use proactively after making Go code changes in this repository.
---

# Pre-PR check

Before treating a change to this repository as done, run the same checks
CI runs (`.github/workflows/ci.yml`), in this order:

```sh
gofmt -l .                    # 1. formatting — must print nothing
go vet ./...                   # 2. static analysis — must be clean
go build ./...                 # 3. build — must succeed
go test ./... -race -cover     # 4. tests — must pass
```

If `gofmt -l .` prints file names, run `gofmt -w .` to fix them and re-check
rather than hand-editing whitespace/formatting.

If any step fails, fix the root cause rather than working around it (e.g.
don't silence `go vet` findings without understanding them, don't skip a
failing test).

New behavior should come with test coverage — see existing `*_test.go`
files in `internal/driver` and `internal/textwrap` for the table-driven
style used throughout this repo.
