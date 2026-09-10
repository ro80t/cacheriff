# cacheriff

A lazygit-inspired terminal UI for finding and cleaning up package manager
caches, globally installed packages, and per-project install artifacts.

Supports npm, pnpm, yarn, bun, deno, cargo, and Go.

## Install

```sh
go install github.com/ro80t/cacheriff@latest
```

Or build from source — see [CONTRIBUTING.md](.github/CONTRIBUTING.md).

## Usage

```sh
cacheriff
```

Run it from a project directory to also see that project's local packages
(e.g. `node_modules` contents) alongside each package manager's shared,
machine-wide cache and globally installed packages.

| Key | Action |
|-----|--------|
| `↑`/`k`, `↓`/`j` | Move selection |
| `enter` | Select a package manager |
| `tab` | Switch focus between panels |
| `←`/`h`, `→`/`l` | Switch between global/local packages |
| `d` | Uninstall the selected global package |
| `esc` | Back to the sidebar |
| `?` | Toggle help |
| `q`, `ctrl+c` | Quit |

## Configuration

cacheriff reads an optional YAML config file for color theme overrides,
following lazygit's convention: `<user config dir>/cacheriff/config.yml`,
or a path set via `CCF_CONFIG_FILE`. See
[config.example.yml](config.example.yml) for the available options.

## Contributing

See [CONTRIBUTING.md](.github/CONTRIBUTING.md).

## License

[MIT](LICENSE)
