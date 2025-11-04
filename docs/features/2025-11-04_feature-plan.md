# Feature Plan — November 4, 2025

This plan outlines the next set of features and refinements for node-module-man. Coding will begin after review and sign‑off.

## Scope & Goals
- Persist user preferences (path, excludes, sort) and restore across runs.
- Manage excludes interactively within the TUI (add/remove patterns).
- Improve deletion UX (per‑item progress, aggregate ETA, retry for failures).
- Performance controls (IO/concurrency limits, large-dir capping/expand-on-demand).
- Reliability improvements (Windows long-path guard, permission error messaging).
- Release infrastructure (basic CI build/test, checksums).

Non‑goals (this iteration)
- Full theme system and extensive styling revamp (tracked separately).
- Cross‑platform “move to trash” (to scope in follow‑up once platform nuances are settled).

## Current Status (context)
- Implemented: TUI help panel, live filtering (`/`), CLI batch delete from JSON, `--yes`, `--version`, build version injection. Tests green.

## Milestones

- TUI Pagination/Virtualization
  - Paginate/virtualize large lists for smooth scrolling (windowed render + ctrl+f/ctrl+b paging, Home/End, gg/G).
  - Acceptance: Scrolling is smooth on large result sets; key bindings work as documented; selection and delete flows remain correct.

- M2: Excludes Manager (TUI)
  - New TUI pane to view current patterns, add new, remove existing.
  - Apply changes immediately to scanning (trigger rescan).
  - Persist to config.
  - Acceptance: add/remove reflects in scan results; changes saved; tests for exclude logic.

- M3: Deletion UX Enhancements
  - Show per‑item progress and aggregate ETA during deletion.
  - Failure panel listing errors with a “retry failed” option.
  - Acceptance: progress/ETA visible; retry only attempts failed items; summaries correct.

- M4 (pending): Performance Controls
  - CLI/TUI option to set IO/concurrency (separate from CPU workers if needed).
  - Cap very large directories initially; expand on demand in TUI before computing full size.
  - Acceptance: controls visibly change throughput; large dirs remain collapsed until expanded.

- M5 (pending): Reliability & Safety
  - Guard Windows long paths (prefixing or friendly errors); improve permission error messages.
  - Acceptance: operations avoid cryptic failures; clear messages with actionable guidance.

## Design Notes
- Config loader
  - Resolve OS‑specific base config dir (`os.UserConfigDir()` in Go 1.19+).
  - Merge order: defaults < config file < CLI flags.
- Selections restore
  - Identify items by absolute path; if not present, ignore.
- Excludes manager
  - Use a simple modal/popup: list patterns; keys: `a` (add), `del` (remove), `enter` (edit).
  - Reuse existing exclude matcher; persist after confirmation.
- Deletion ETA
  - Based on rolling average time per directory (bytes or directory count). Show coarse estimate.
- Large-dir capping
  - Add a “collapsed” state for directories above threshold; compute size only on expand.
- Error handling
  - Wrap permission and long-path errors with friendly strings; do not crash.

## Implementation Steps (high level)
1) Config package with load/save, schema, OS‑path helpers
2) Wire config into CLI/TUI bootstrap (load, merge, apply)
3) Add selection restore logic keyed by absolute paths
4) TUI excludes manager (modal) and rescan wiring
5) Deletion progress: extend `deleter.Progress` and TUI render with ETA
6) Large-dir capping: scanner fast pass + on‑demand expand
7) Reliability: error wrapping for perms/long paths
8) CI workflow yaml (build/test/checksums)

## Testing Plan
- Unit tests
  - Config load/save roundtrip; merge precedence.
  - Scanner: excludes, depth, symlink follow, large‑dir capping toggles.
  - Deleter: progress stream, retry failed only, dry‑run preserved.
- TUI model tests (logic-only)
  - Excludes manager add/remove state transitions.
  - Selection restore after rescan mapping.
- Manual checks
  - TUI paging, filter/search, deletes with ETA, retry flow.

## Risks & Mitigations
- Platform differences (paths, trash/long‑path): gate features, handle per‑OS.
- ETA accuracy: display as estimate with clear labeling; avoid blocking UI.
- Large-dir capping UX: ensure discoverability (collapsed indicator + hint).

## Rollout
- Behind flags where appropriate; dogfood internally first.
- Update README with new flags/panels; add screenshots/GIFs.
- Tag minor release once stable.
