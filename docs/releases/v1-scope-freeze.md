# NexusDesk v1 Scope Freeze

Date: 2026-05-30
Status: frozen release scope

This document freezes the v1 product scope so production-readiness work can converge on a stable target. New ideas may be captured as post-v1 candidates, but they must not expand the v1 release bar unless the tracker is deliberately reopened.

## v1 Release Promise

NexusDesk v1 is a Fyne-native, local-first desktop workbench for safe workspace editing, source-grounded assistant use, approval-gated agent tools, local data inspection, durable artifacts, diagnostics, and trustworthy release evidence.

The v1 release is complete when the documented core flow works on the supported release targets and the remaining release trust, smoke, security, performance, accessibility, and documentation gates are closed with evidence.

## In Scope

- Native Fyne workbench with compact editor-centered layout, rails, left tool windows, right assistant, status bar, command palette, and first-run onboarding.
- Workspace open, recent workspaces, lazy navigation, safe file preview/edit/save/revert, rollback metadata, bounded large-file handling, search, problems, and Git inspection.
- Assistant Ask and Agent modes with provider setup, model routing, bounded streaming, source citations, source coverage warnings, chat history, and artifact save paths.
- Deterministic local agent tools that are implemented, bounded, approval-gated by risk, audited, and reversible where practical.
- Data workflows for common local files, SQLite, read-only external connectors, SQL/query history, result export, chart/dashboard artifacts, and connector diagnostics.
- Artifact workflows for chat answers, reports, notebooks, charts, dashboards, DOCX, PPTX, runbooks, lineage, freshness, regeneration, archive, restore, delete, and rollback-aware destructive operations.
- Jobs, approvals, agent audit, activity, diagnostics, crash recovery, redacted issue reports, protected secret status, release trust diagnostics, and app-data cleanup guidance.
- Repeatable release packaging with version metadata, hashes, manifest, SBOM, provenance, private-beta installer notes, and platform-specific trust documentation.

## Deferred Post-v1

- Cloud workspaces, accounts, team collaboration, sync, telemetry by default, silent auto-update, and plugin marketplace distribution.
- Autonomous high-risk mutations, production database mutation, Docker/system mutation, and any unreviewed background indexing on workspace open.
- Browser automation, interactive terminal sessions, pull-request platform tools, MCP/plugin invocation, scheduled automations, semantic search, connector sync jobs, and image/screenshot understanding as executable agent tools.
- Full multi-language IDE parity, broad LSP/code-action support, cloud model hosting, and multi-user administration.
- Public update channels beyond documented manual release checks.

## Release Blockers That Do Not Expand Scope

The following remaining gates must be closed or explicitly deferred for the release decision, but they do not add product features:

- Windows executable and installer signing.
- macOS app/package production and signing/notarization.
- Linux package production and trust evidence.
- Linux and macOS CI/test/package smoke evidence.
- Cross-platform protected-secret smoke evidence.
- Windows, macOS, and Linux clean-machine smoke evidence.
- Uninstall and app-data cleanup smoke evidence.
- Five-user beta install test and feedback triage.
- Final security, performance, accessibility, file data-loss, plaintext-secret, workspace-open side-effect, artifact/hash, docs-behavior, and release-notes verification.

## Change Control

Any change that adds a new v1 feature must update this document, the feature inventory, and the execution tracker in the same change. If the change is not required to satisfy the frozen v1 promise, it should be recorded as post-v1 work instead.
