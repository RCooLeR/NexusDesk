# NexusDesk Performance Profiling Harness

Date: 2026-05-28

NexusDesk needs measured performance before production release, especially as the Fyne shell grows toward a JetBrains-like workbench. The first native profiling harness lives in `nexus-app/internal/services/perf` and is intentionally framework-free so it can run in CI, release-candidate smoke, and developer machines without launching a desktop window.

## What It Covers

The quick profile exercises representative hot paths:

- shell redraw model: materializes tab/status-style labels;
- activity log model: appends and caps long activity history;
- data grid model: creates a bounded CSV fixture and runs a capped dataset query;
- large search: creates a synthetic workspace and runs bounded path/content search;
- large artifacts: creates generated artifact records and lists them through the artifact browser service.

This is not a replacement for manual UI profiling. It is a deterministic service-level smoke harness that catches obvious scaling regressions before they become desktop-window issues.

## How To Run

Run the focused package:

```powershell
cd nexus-app
go test ./internal/services/perf
```

Run the benchmark recipe when comparing changes:

```powershell
cd nexus-app
go test ./internal/services/perf -bench . -benchmem
```

Run full validation after performance-sensitive changes:

```powershell
cd nexus-app
go test ./...
go build .
cd ..
git diff --check
```

Remove generated binaries such as `nexus-app/nexusdesk.exe` after build validation.

## Harness Behavior

- The harness requires a scratch parent directory.
- It creates a temporary fixture workspace under that parent.
- It cleans the fixture by default.
- Callers can request retained fixtures for manual inspection.
- It honors canceled contexts.
- It reports scenario duration, item counts, budget, pass/fail flag, and detail text.

## Current Budgets

The first budgets are generous smoke thresholds, not release SLAs:

- shell redraw model: 150 ms;
- activity log model: 150 ms;
- data grid model: 300 ms;
- large search: 600 ms;
- large artifacts: 900 ms.

These should be tightened only after baseline data is collected on representative Windows, macOS, and Linux machines.

## Production Follow-Ups

- Add startup and folder-open timing logs to Diagnostics.
- Add manual UI profiling recipes for shell redraw, tab switching, Data grid scrolling, artifact browsing, and Diagnostics export.
- Add large workspace, large CSV/query/grid, long chat/agent session, and large artifact directory fixtures to release-candidate smoke.
- Capture memory snapshots during private-beta release candidates.
- Keep slow future workflows such as OCR, dump imports, connector syncs, long indexing, packaged exports, and long agent runs routed through durable jobs before UI exposure.
