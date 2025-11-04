# TODOs

- TUI polish
  - [x] Add help panel (`?`) with key bindings.
  - [x] Add filter/search (press `/`, live filtering, quick jump to matches).
  - [x] Paginate/virtualize large lists for smooth scrolling (windowed render + ctrl+f/ctrl+b, Home/End, gg/G).
  - [ ] Improve layout and styling using `lipgloss` (headers, colors, spacing).

- Selection + state
  - Persist last used settings (path, excludes, sort) to a config file.
  - Restore selections after rescans when possible (by absolute path).

- Scanner & performance
  - Configurable IO/concurrency limits; adapt based on CPU/IO.
  - Skip or cap very large directories until user expands them.

- Excludes & filters
  - Manage excludes inside TUI (add/remove patterns interactively).
  - Support exact-path excludes and presets (e.g., `**/examples/**`).

- Deletion UX
  - Show per-item progress and aggregate ETA during deletion.
  - Optional “move to trash” instead of permanent delete (platform-specific).
  - Detailed error panel for failures with retry option.

- CLI enhancements
  - [x] `--yes` non-interactive deletion confirmation for CLI mode.
  - [x] Accept targets from JSON input to batch-delete without TUI (`--delete-json` or `--delete-stdin`).
  - [x] Output machine-readable summaries for CI scripts (`--json`).

- Reliability & safety
  - [x] Cancel scanning on `q` with graceful cleanup (context cancel before quit).
  - [ ] Guard against long Windows paths and permission errors; friendly messages.

- DX & release
  - [x] Add `Makefile` for common tasks.
  - [ ] Add CI workflow for cross-platform builds and checksums.
  - [x] Add version injection via `-ldflags` and `--version` flag.
