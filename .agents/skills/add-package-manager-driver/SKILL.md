---
name: add-package-manager-driver
description: Scaffold a new package-manager driver for cacheriff (internal/driver). Use when asked to add support for a new package manager (e.g. "add a driver for X", "support Homebrew/winget/composer/pip/gem/etc").
---

# Add a package manager driver

cacheriff reports and cleans up caches, global packages, and local
(per-project) packages for each package manager it knows about. Each one is
a `Driver` implementation under `internal/driver`.

## Steps

1. **Read `internal/driver/driver.go`** for the `Driver` interface and
   `internal/driver/base.go` for the shared plumbing every driver embeds
   (`base` gives you `ID`, `Name`, `Available`, `SupportedOS`,
   `LocalArtifactDirNames`, `LocalInstallDir`, plus helpers `runOutput`,
   `runCombined`, `singleDirCacheEntries`, `sizeCacheDirs`,
   `unsupportedKindErr`, `dirSize`).

2. **Pick a reference driver** whose shape matches the new one:
   - A cache that lives under one directory, with a CLI command that
     prints its own global install dir and a list command → `npm.go`.
   - Several named cache subdirectories under a shared home dir → `cargo.go`
     or `go.go`.
   - No per-project install directory (dependencies live only in a shared
     cache) → leave `localDir` unset in the constructor, like `cargo.go`,
     `go.go`, and `deno.go` do; `LocalPackages` should return `(nil, nil)`
     in that case, and `LocalInstallDir` will already return `("", false)`
     via `base`.

3. **Create `internal/driver/<name>.go`.** Embed `base` in a
   `<name>Driver` struct. Implement:
   - `CacheDir` / `CacheEntries` — where the shared cache lives and how to
     size it (`singleDirCacheEntries` for one directory,
     `sizeCacheDirs` for several, sized concurrently — that matters when a
     cache holds many small files).
   - `GlobalInstallDir` / `GlobalPackages` — where global installs live and
     how to list them. If the tool has no reliable "list installed"
     command, look at how `deno.go` falls back to scanning shim scripts, or
     how `go.go` uses `go version -m` to read embedded build info.
   - `LocalPackages` — the project's direct dependencies, resolved to a
     path under the shared cache (or under the local install dir) so they
     can be sized.
   - `Remove` — prefer the package manager's own uninstall/clean command
     over deleting files directly, so its own metadata/lockfiles stay
     consistent. Pass any user-controlled name (e.g. a package name)
     through `textwrap.EscapeArg` before using it as a CLI argument — see
     `internal/textwrap/escape.go` for why.

4. **Watch for CLI quirks** — many package manager CLIs exit non-zero on
   partial/warning conditions while still printing valid JSON (see npm's
   and Go's drivers), or don't include everything you'd expect in one
   command's output. Prefer parsing what the tool actually prints over
   assuming a "clean" API.

5. **Register the driver** in `All()` in `driver.go`.

6. **Add `internal/driver/<name>_test.go`** covering at least the output
   parsing function(s) with realistic captured CLI output, following the
   style of the existing `*_test.go` files (table-driven, real sample
   output as a comment/string literal, not synthetic-looking data).

7. **Verify**: `go build ./...`, `go vet ./...`,
   `go test ./... -race -cover`, `gofmt -l .`.

See `AGENTS.md` for the deeper rationale behind these conventions, and
`.github/CONTRIBUTING.md` for the human-facing contributor doc.
