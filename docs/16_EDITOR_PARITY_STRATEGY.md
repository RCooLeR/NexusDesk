# Native Editor Parity Strategy

Date: 2026-05-28

This document records the Native Parity Beta decision for replacing Monaco/Wails editor behavior in the Fyne-native app.

## Decision

For Native Parity Beta, NexusDesk accepts the current native editor strategy:

- safe Fyne text editing remains the primary editable surface;
- the read-only Syntax mirror provides bounded token highlighting, line numbers, active-line highlighting, and cursor-aware token/symbol status;
- the Document Map replaces Monaco minimap value with jumpable symbols, markers, merge conflicts, and long-file anchors;
- native outline, go-to-symbol, local definition, bounded workspace definition fallback, and bounded references search provide editor navigation without starting external language servers;
- live draft diagnostics and saved-file Problems scans cover parser/syntax safety for supported local formats.

This means active-editor inline token styling is not a Native Parity Beta blocker. It remains a post-beta enhancement only if it preserves safe editing, accessibility, packaging reliability, and responsiveness.

## Why

The Wails/Monaco implementation delivered strong highlighting and language-worker value, but it depended on browser/webview packaging and frontend worker behavior that the Fyne migration is intentionally removing.

The native replacement is not a one-for-one Monaco clone. It is a production-safe desktop strategy:

- editing stays simple, predictable, and rollback-safe;
- token analysis is bounded and cannot freeze large files;
- navigation features are explicit and testable in Go services;
- no language server or web worker starts during workspace open;
- future richer editor engines must prove packaging and accessibility before becoming default.

## Native Parity Beta Acceptance

The editor is acceptable for Native Parity Beta when these remain true:

- normal text/code edit, save, revert, close, pin, and dirty-state workflows work without data loss;
- find/replace, formatting, breadcrumbs, outline, go-to-symbol, local definition, workspace definition fallback, references, Syntax mirror, Document Map, and draft diagnostics remain available;
- saved-file Problems scans continue to cover JSON, Go, YAML, TOML, and XML syntax diagnostics;
- language-action readiness surfaces available, fallback, planned, and unavailable behavior clearly;
- LSP and active inline syntax styling are documented as post-beta enhancements, not hidden blockers.

## Post-Beta Milestones

- Spike editable-widget inline syntax styling only if it preserves safe editing, accessibility, and performance.
- Prototype one packaged LSP provider behind an explicit feature flag with cancellation and failure isolation.
- Expand semantic diagnostics only after local syntax scans and fallback navigation remain stable.
- Revisit embedded editor options only after a focused spike proves packaging, accessibility, and local-first safety.
