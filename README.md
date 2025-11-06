# node-module-man

Scan and clean `node_modules` across a folder recursively. Comes with a fast scanner, an interactive TUI for selecting directories to delete, and a scriptable CLI for CI or batch workflows.

![preview](./screenshot/1.png)

## Features

- Fast, concurrent scan for `node_modules` (depth limiting, excludes, symlink options)
- TUI: live streaming results, filter, sort, multi-select, select-all/invert, confirm + delete, progress view
- CLI: JSON output, non-interactive deletion with `--yes`, batch delete from JSON
- Dry-run mode for safe validation, graceful cancellation during scanning/deleting

## Requirements

- Go 1.19+
- macOS/Linux/Windows (tested in CI-friendly setups)

## Build & Install

- One-off build (current OS/arch):
  - `go build ./cmd/node-module-man`
- Make targets:
  - `make build` → macOS arm64+amd64 into `dist/`
  - `make all` → Mac/Linux/Windows binaries into `dist/`
  - `make version VERSION=1.2.3` → same as `make build` but with version injection
- Cross-platform via script:
  - macOS (arm64+amd64): `./build.sh`
  - Common OS targets: `./build.sh all`
- Release artifacts (archives + checksums):
  - `VERSION=1.2.3 ./build.sh release`
  - Produces `dist/node-module-man_${VERSION}_${GOOS}_${ARCH}.*` and `dist/checksums_${VERSION}.txt`.
- Common tasks via Makefile:
  - `make build` • `make test` • `make fmt` • `make version VERSION=1.2.3`

Version injection: binaries can embed a version string via `-ldflags`.
- `VERSION=1.2.3 ./build.sh` or `make version VERSION=1.2.3`

## Quick Start (TUI)

- Scan current directory and open TUI:
  - `./node-module-man`

TUI key bindings (press `?` in the app for help):
- `↑/k`, `↓/j`: move cursor
- `space`/`x`: toggle delete selection `[x]`
- `z`: toggle compress selection `[z]`
- `A` / `X` / `ctrl+a`: mark all `[x]` (filtered view)
- `Z`: mark all `[z]` (filtered view)
- `R`: invert marks (z→·, x→·, ·→x)
- `s`: toggle sort field (size/path)
- `r`: reverse sort
- `/`: filter list (type to refine; Enter to confirm; Esc to clear)
- Navigation: `gg`/`G` jump to top/bottom; `Home`/`End`; `ctrl+f`/`ctrl+b` page
- `d` or `enter`: perform action — delete if any `[x]`, or compress if any `[z]`
- `?`: toggle help
- `q/esc`: quit; cancels ongoing scan/delete/compress

## CLI Usage

- Disable TUI and print results (table or JSON):
  - Table: `./node-module-man --tui=false -p .`
  - JSON: `./node-module-man --tui=false --json -p .`

Key flags:
- `--path, -p`: root path to scan (default `.`)
- `--concurrency, -c`: workers for size calculations (default: CPU cores)
- `--max-depth, -m`: max directory depth to traverse (`-1` unlimited)
- `--exclude, -x`: repeatable glob/pattern to exclude (matches path or basename)
- `--follow-symlinks, -L`: follow symlinked directories
- `--dry-run, -d`: simulate deletion (no files removed)
- `--compress-json`, `--compress-stdin`: compress targets from JSON
- `--out-dir`: output directory for zip archives (default: alongside source)
- `--delete-after`: delete originals after compress (default: true)
- `--version`: print version and exit

### Delete (non-interactive)

Delete selected targets non-interactively with `--yes`. Input targets via JSON file or stdin:

- From file: `./node-module-man --tui=false --delete-json targets.json --yes`
- From stdin: `cat targets.json | ./node-module-man --tui=false --delete-stdin --yes`
- Add `--json` to get a machine-readable summary for CI.

Accepted JSON formats:
```json
["/abs/path/one", "/abs/path/two"]
```
```json
[{"path":"/abs/path/one","size":123}, {"path":"/abs/path/two"}]
```
```json
{"targets": ["/abs/path/one", {"path":"/abs/path/two","size":2048}]}
```

## Examples

- Scan current path (table output):
  - `./node-module-man --tui=false -p .`

- Scan as JSON, follow symlinks, exclude examples:
  - `./node-module-man --tui=false --json -p . -L -x '*/examples/*'`

- Delete from a JSON file (non‑interactive):
  - `./node-module-man --tui=false --delete-json targets.json --yes`

- Compress from stdin to a custom folder, keep originals:
  - `cat targets.json | ./node-module-man --tui=false --compress-stdin --out-dir ./archives --delete-after=false --yes`

- Compress from a JSON file (default deletes originals after success):
  - `./node-module-man --tui=false --compress-json targets.json --yes`

Sample `targets.json` inputs accepted by both delete and compress:

```json
[
  "/abs/path/to/project-a/node_modules",
  {"path": "/abs/path/to/project-b/node_modules", "size": 123456}
]
```

or wrapped in an object:

```json
{
  "targets": [
    "/abs/path/to/project-a/node_modules",
    {"path": "/abs/path/to/project-b/node_modules"}
  ]
}
```

Deletion summary (when `--json`):
```json
{
  "Successes": [{"Path": "/abs/path/one", "Size": 123}],
  "Failures": [{"Path": "/abs/path/two", "Err": "permission denied"}],
  "Freed": 123
}
```

## Excludes & Filters

- `--exclude` patterns are matched against full path and basename.
- Simple wildcard forms like `*/examples/*` are supported.
- Examples:
  - Skip all `node_modules` inside `packages/a`: `--exclude '*/packages/a/*'`
  - Skip by basename (skip everything named `node_modules`): `--exclude node_modules`

## Performance & Symlinks

- Concurrency defaults to `runtime.NumCPU()`; tune via `--concurrency`.
- Limit traversal with `--max-depth` to avoid deep directory walks.
- Do not follow symlinked directories by default; enable with `-L/--follow-symlinks`.

## Safety Notes

- TUI requires confirmation before delete/press `y`; compression confirm is shown but delete-after is default.
- CLI delete/compact requires `--yes` to proceed without prompt.
- Use `--dry-run` during validation to simulate deletions safely.

Compression specifics:
- Archives are `.zip` with a top-level `node_modules` folder (extracts cleanly).
- Symlinks inside `node_modules` are skipped for portability.
- By default, originals are removed after successful compression; disable with `--delete-after=false`.

## Development

- Project structure:
  - `cmd/node-module-man/` — entrypoint (CLI/TUI)
  - `internal/scanner/` — discovery + size computation
  - `internal/tui/` — Bubble Tea model and list UI
  - `internal/deleter/` — concurrent deletion with progress and dry‑run
  - `pkg/utils/` — helpers (byte formatting)
  - `scripts/` — utilities (e.g., fixtures)
  - `docs/` — PRD, plan, progress, known issues

## Tests

- Run: `go test ./...`
- Scanner tests cover discovery, depth, excludes, symlinks; deleter covers dry‑run and cancel.

## Fixtures (optional)

Generate test fixtures with a Node script (no network by default):
- `node scripts/make-test-fixtures.js --count 4`
- Add `--no-install` to skip `npm install` if the script supports it.

## Roadmap & Progress

- See `docs/features/todos.md` for planned items.
- See `docs/progress.md` for completed milestones.
- Known issues live under `docs/issues/`.

## Acknowledgements

- Built with the Charm stack: Bubble Tea and Lipgloss.
