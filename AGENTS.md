# Repository Guidelines

This repo contains a Go TUI/CLI tool for scanning and cleaning `node_modules`. Follow these concise conventions to contribute effectively.

## Project Structure & Module Organization
- `cmd/node-module-man/` — CLI/TUI entrypoint.
- `internal/scanner/` — discovery + size computation (concurrency, excludes).
- `internal/tui/` — Bubble Tea models, list UI, deletion flow.
- `internal/deleter/` — concurrent deletion with progress and dry‑run.
- `pkg/utils/` — helpers (e.g., byte formatting).
- `scripts/` — utilities (e.g., `make-test-fixtures.js`).
- `docs/` — PRD, plan, progress, known issues.

## Build, Test, and Development Commands
- Build macOS binaries: `./build.sh` (arm64 + amd64 to `dist/`).
- One‑off build: `go build ./cmd/node-module-man`.
- Run TUI (default enabled): `./node-module-man -p .`.
- Run CLI (disable TUI): `./node-module-man --tui=false --json`.
- Tests: `go test ./...`.
- Create fixtures: `node scripts/make-test-fixtures.js --count 4` (optional `--no-install`).

## Coding Style & Naming Conventions
- Go 1.19+. Use `gofmt` defaults; no custom linters required.
- Packages: lowercase; exported identifiers use `CamelCase`.
- Files: keep modules small and focused; avoid wide, unrelated diffs.
- TUI: prefer small, pure functions; avoid styling full multi‑line blocks.

## Testing Guidelines
- Use Go’s `testing` package; name tests `TestXxx` in `*_test.go`.
- Cover scanner (discovery, depth, excludes, symlinks) and deleter (dry‑run, cancel).
- Keep fixtures under temp dirs in tests; do not rely on network.
- Run: `go test ./...` before opening a PR.

## Commit & Pull Request Guidelines
- Commits: imperative mood, scoped (e.g., `tui: fix selection cursor`).
- PRs: include summary, motivation, screenshots/GIFs for UI, and steps to test.
- Link related issues in `docs/issues/` when applicable; update `docs/progress.md` for feature milestones.

## Security & Configuration Tips
- Destructive ops: default to `--dry-run` during validation; confirm before delete.
- Symlinks: prefer not to follow; enable explicitly with `-L/--follow-symlinks`.
- Avoid running as root; handle permission errors gracefully.

## Agent Memory & MUSE Loop
- Follow Think → Act → Reflect for substantive changes; consult `MUSE.md`.
- Record durable learnings in `memory/`:
  - `procedural.mem.md`: reusable steps/SOPs (e.g., fixing list indentation).
  - `strategic.mem.md`: high‑level choices and trade‑offs (styling strategy).
  - `tool.mem.md`: concrete tool usage/quirks (bubbletea/list, lipgloss).
- When you fix a recurring issue, add a short note under `docs/issues/` and update the relevant memory file.
