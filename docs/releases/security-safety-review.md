# NexusDesk v1 Security And Safety Review

Date: 2026-05-30

Status: release security/safety review complete for the current v1 candidate. This review did not sign artifacts, replace clean-machine platform smoke testing, or prove public distribution trust.

## Scope

This review covers the current Fyne-native NexusDesk codebase and release docs for:

- Workspace filesystem reads, previews, writes, rollback, path containment, and workspace-open behavior.
- Agent and assistant tool execution, mutation reporting, approval gates, audit output, timeouts, cancellation, output caps, and repeated-tool-loop safety.
- Tool catalog risk controls for implemented and planned tools.
- Secret storage, credential references, redaction, connector transport state, and plaintext warnings.
- Local data and external database read-only guardrails.
- Jobs, durable logs, issue reports, long-running workflow visibility, and cancellation.
- Generated artifacts, destructive artifact flows, lineage, rollback snapshots, and regeneration cancellation.
- Release trust evidence, update visibility, obsolete web-runtime references, and documentation alignment.

Out of scope for this review:

- Code signing certificate custody and signing ceremony verification.
- Apple notarization, Linux package repository trust, or platform-store review.
- Clean-machine platform smoke results that require external machines.
- Penetration testing by an independent reviewer.
- Five-user beta install validation.

## Evidence Reviewed

- `internal/services/security/threat_model.go` defines release controls for approval, audit, rollback or mitigation, redaction, rooted scope, connector scope, timeout, cancellation, output caps, durable jobs, secret isolation, artifact lineage, sandboxing, no workspace-open side effects, preview, and user-visible status.
- `internal/services/security/threat_model_test.go` verifies risky controls, future high-impact threat families, durable and visible controls, approval requirements, and immutable control metadata.
- `internal/services/tools/catalog_test.go` verifies implemented tools are cataloged, planned tools are not executable, high-risk tools have approval plus rollback/mitigation controls, and catalog validation rejects missing controls.
- Agent tests verify mutation honesty, timeout propagation, tool audit behavior, and repeated-tool-loop safety stops.
- Task tests verify discovered tasks use argv-style execution, reject shell-interpreter shortcuts, reject shell payloads in task names, clamp terminal timeouts, and route terminal execution through approved integration.
- Workspace tests verify rooted path safety, symlink escape rejection, partial preview save blocking, rollback creation/application, unsafe target rejection, ambiguous encoding save blocking, and search binary/cap behavior.
- Connector, settings, protected-secret, LLM, operations, issue-report, diagnostics, and UI tests verify redaction, protected credential references, plaintext transport warnings, provider error redaction, protected-secret backend visibility, and issue-report redaction.
- Release evidence tests and scripts verify manifests, SBOM, provenance, artifact hashes, manual update visibility, and release trust diagnostics.
- Documentation and repository sweeps removed obsolete web-runtime references and stale web-layout task skip paths.

## Findings

No known local P0 security or safety defect was found in the reviewed evidence.

The current candidate has strong local controls for high-risk behavior:

- Workspace mutation is rooted, previewed, approval-gated where applicable, rollback-backed, audited, and blocked for partial or ambiguous previews.
- Agent tool results carry mutation signals used for final verification rather than relying on fragile tool-name guesses.
- High-risk implemented tools require approval and audit controls, and planned high-risk tool families are documented as non-executable roadmap contracts until design approval and tests exist.
- Terminal/task execution is constrained to known task definitions or approved, rooted, timeout-bound, output-capped one-shot execution.
- Secrets and credentials are kept behind protected-secret references where supported, with redaction applied to UI, logs, diagnostics, and issue reports.
- Database and connector flows default toward read-only or encrypted behavior, with explicit development-only plaintext states surfaced and audited.
- Long-running jobs expose visible status, durable logs, cancellation, timeout behavior, and redacted support evidence.
- Generated artifact destructive flows have rollback snapshots, lineage, source freshness, and cancellation coverage.
- Workspace open policy blocks hidden slow jobs and hidden model/tool work.
- Release trust evidence exists for Windows artifacts through manifests, SBOM, provenance, hash verification, and diagnostics.

## Residual Release Blockers

This review does not clear the remaining release blockers:

- Windows executable and installer signing are not complete.
- Windows clean-machine launch, workspace, edit/save/revert, assistant, data/artifact, and diagnostics smokes are missing.
- macOS and Linux clean-machine smokes are missing.
- Five-user beta install validation and feedback triage are still open.

## Decision

Security/safety review status for the current local candidate: pass with release blockers.

The reviewed code and tests support closing the P0 security/safety review item because the current local candidate has no known unaddressed P0 safety defect in the covered areas. The app is still not production-release complete until signing, clean-machine platform smoke, beta validation, and final P0/P1 disposition are completed or explicitly deferred in the tracker.
