# NexusDesk v1 Performance Review

Date: 2026-05-30

Status: local release performance review complete for the current v1 candidate. This review validates deterministic performance controls and regression coverage in the current Windows development environment. It does not replace platform CI, clean-machine smoke, or profiling on representative customer repositories.

## Scope

This review covers the v1 performance risks called out in the plan and tracker:

- Startup and workspace-open timing capture.
- Assistant streaming and activity rendering under long sessions.
- Editor save responsiveness, save state, cursor/scroll preservation, and large-file diff fallback.
- Workspace search streaming, cancellation, result caps, binary/container skips, and large-workspace behavior.
- Metadata store concurrency, WAL mode, busy timeout, and cached connection strategy.
- External connector pool reuse, invalidation, cancellation, and bounded idle lifetime.
- Data grid, artifact listing, generated artifact counts, and quick performance smoke scenarios.
- Diagnostics visibility for over-budget startup or workspace-open timings.

Out of scope for this review:

- GPU, font, and window-manager differences across macOS/Linux desktops.
- Clean-machine startup and folder-open timing on release hardware.
- Very large real-world repositories beyond the deterministic synthetic fixtures.
- Network latency and provider model latency.
- Independent profiling by a beta user.

## Evidence Reviewed

- `internal/services/perf/harness.go` defines deterministic performance smoke scenarios for shell redraw model, activity log model, data grid model, large search, and large artifacts.
- `internal/services/perf/harness_test.go` verifies all five scenarios are covered, scenario budgets are recorded, search result caps are honored, canceled contexts stop profiling, and fixture cleanup is the default.
- `internal/services/perf/benchmark_test.go` provides a repeatable quick-profile benchmark harness.
- `internal/services/perf/timings.go` and `timings_test.go` verify startup/workspace timing recording, retention, copies, and over-budget classification.
- `internal/ui/shell/performance.go`, `diagnostics_panel.go`, and diagnostics tests surface captured timings, warnings, health cards, and recommended actions for over-budget startup or folder-open work.
- Assistant, activity, search, editor, metadata, connector, jobs, git, workspace, and shell tests cover coalesced rendering, capped buffers, off-UI save behavior, bounded diffs, large search fixtures, cancellation, WAL/busy-timeout behavior, pool reuse, log caps, timeout classes, and UI timing diagnostics.

## Findings

No known local P0 performance defect was found in the reviewed evidence.

The current candidate has the main v1 performance controls in place:

- Workspace open is designed to stay cheap and side-effect-free, with startup/workspace timing records captured for diagnostics.
- Diagnostics turns over-budget startup or workspace-open timings into visible health-card warnings and recommended actions.
- Assistant and activity rendering use coalesced or bounded updates instead of rebuilding unbounded markdown on every event.
- Editor save flow runs off the UI path, shows save state, preserves editor affordances, and uses bounded diff fallback for large files.
- Search uses streaming bounded reads, binary/container pre-skip behavior, result caps, cancellation, stale-search suppression, and synthetic large-workspace tests.
- Metadata storage uses WAL, foreign keys, busy timeout, one cached SQLite handle, and concurrent read/write coverage.
- External connector access uses versioned pool reuse, invalidation on profile changes, context-aware borrowing/querying, bounded pool lifetimes, and diagnostics.
- Durable job logs and issue-report excerpts are capped and redacted, avoiding unbounded UI/report growth.
- The dedicated quick-profile harness exercises shell, activity, data, search, and artifact scenarios with explicit budgets.

## Residual Release Blockers

This review does not clear release blockers that require external execution environments:

- Full CI is still not proven for the latest release candidate.
- Full platform smoke is still open.
- Windows clean-machine launch, workspace, edit/save/revert, assistant, data/artifact, and diagnostics smokes are still open.
- macOS and Linux package artifacts, CI, and clean-machine smokes are still open.
- Cross-platform protected-secret smoke is still open.
- Accessibility release review, release notes publication, signing, and beta install validation are still open.

## Decision

Performance review status for the current local candidate: pass with release blockers.

The reviewed implementation and focused verification support closing the P0 performance review item because the current local candidate has deterministic performance controls, diagnostics visibility, and regression coverage for the v1 risk areas. Production release remains blocked until platform CI/smoke, signing, beta validation, accessibility review, and release-note publication are completed or explicitly dispositioned in the tracker.
