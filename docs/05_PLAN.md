# Production Plan

Status: canonical production plan.

This plan sequences NexusDesk from the current native foundation to a production-ready desktop release. It prioritizes safety and data-loss prevention first, then performance, audit correctness, UI coherence, packaging, and beta readiness.

## 1. Planning Assumptions

- The active product is the Fyne-native app under `nexus-app/`.
- Documentation in `docs/` is the single source of truth.
- Services remain framework-free.
- The UI shell remains native.
- Workspace open remains cheap and side-effect-free.
- Risky actions require approval, audit, redaction, cancellation, and rollback or mitigation where practical.
- Slow workflows route through jobs.
- Planned tools stay non-executable until their safety design is implemented.
- Release work is not complete until clean-machine smoke passes.

## 2. Current Assessment

The app already has a strong foundation:

- Native desktop app foundation is mostly in place.
- Service architecture is strong and testable.
- Workbench, editor, Git, data, assistant, agent, tools, artifacts, jobs, settings, and diagnostics have broad coverage.
- The remaining risk is concentrated in safety hardening, performance under long sessions, UI coherence, packaging trust, and end-to-end smoke.

Planning estimates:

- Native app foundation: about 98% complete.
- Core feature coverage for private beta: about 95% complete.
- UI target polish: about 70% complete.
- Safety/performance hardening: about 80% complete.
- Packaging/release trust: about 55% complete.
- Overall production readiness: about 80% complete.

These are planning estimates, not release guarantees.

## 3. Phase 0: Documentation Reset And Product Alignment

Goal: make the plan unambiguous and remove planning clutter.

Milestones:

- Canonical docs exist and are detailed.
- The tracker has explicit checkboxes for remaining work.
- Old planning references are removed from active docs.
- Root tracker points to the canonical tracker.
- Developers can start a new slice by reading `docs/README.md` and `docs/06_TRACKER.md`.

Exit criteria:

- `docs/` contains only the canonical Markdown set needed for planning.
- `git diff --check` passes for docs changes.
- No obsolete app/runtime references appear in docs.

## 4. Phase 1: Safety Lockdown

Goal: close known data-loss, credential-exposure, SSRF, and command-injection risks before UI or feature expansion.

Milestones:

### 1.1 File preview and save safety

- Add top-level truncation metadata to file previews.
- Disable save for partial previews unless a safe full-file edit path exists.
- Show clear UI banner for truncated files.
- Include truncation state in assistant/tool observations.
- Add regression tests for large/truncated files.

### 1.2 Archive/container parser caps

- Add XLSX decompression caps.
- Add DOCX/PPTX container caps where relevant.
- Limit per-entry and total uncompressed bytes.
- Add regression tests for malicious compressed containers.

### 1.3 Protected secret hardening

- Ensure macOS secret storage never places secret values in command arguments.
- Ensure Windows DPAPI buffers are kept alive across syscalls.
- Ensure Linux protected storage failure is explicit and tested.
- Add platform-specific tests or smoke instructions.

### 1.4 Network and database safety

- Harden `web_fetch` against DNS rebinding.
- Default external database connections to encrypted transport.
- Add explicit audited development-only plaintext opt-in.
- Show resolved transport mode in UI/diagnostics.

### 1.5 Tool mutation honesty

- Ensure every mutating tool sets `Mutated: true` accurately.
- Make final agent verification trust tool result mutation flags.
- Add tests for formatting, Git mutations, artifact regeneration, conflict resolution, and project memory updates.

### 1.6 Task and terminal execution safety

- Ensure task execution uses argv, not shell-string execution.
- Keep one-shot terminal commands rooted, argv-only, approval-gated, bounded, and audited.
- Add tests with malicious task names and shell metacharacters.

Exit criteria:

- All Phase 1 tracker items are checked.
- Tests exist for every closed safety risk.
- Threat/safety docs and tracker are updated.
- `go test ./...` passes in CI.

## 5. Phase 2: Performance Floor

Goal: make the app feel fast on real projects and long sessions.

Milestones:

### 2.1 Streaming and activity rendering

- Coalesce assistant streaming UI updates.
- Parse final markdown once after stream completion.
- Coalesce agent events.
- Convert activity rendering to incremental list-style updates.
- Add stress tests or profiling harness for long streams.

### 2.2 Editor save performance

- Move save/diff/rollback off the UI thread.
- Show saving state.
- Avoid rebuilding the whole editor on save.
- Preserve cursor and scroll where practical.
- Replace large-file LCS diff with bounded hunk-based diff.

### 2.3 Search performance

- Replace preview-based content search with streaming byte-level search.
- Skip known binaries before opening files.
- Add cancellation/singleflight for search-while-typing.
- Stream results into UI.
- Add large-repo search benchmark or test fixture.

### 2.4 Metadata and rollback performance

- Enable SQLite WAL and busy timeout.
- Configure metadata connection usage.
- Surface journal mode in diagnostics.
- Add rollback storage diagnostics.
- Add content-addressed or deduplicated rollback storage if needed.

### 2.5 Connector performance

- Reuse external database pools with short TTL.
- Invalidate pools on profile changes.
- Preserve cancellation.
- Surface connection reuse and transport status in diagnostics.

Exit criteria:

- Workspace open target is met.
- Search first-result target is met.
- Editor save does not visibly freeze.
- Streaming stays smooth for long answers.
- Metadata panels do not stall during agent runs.

## 6. Phase 3: Correctness And Audit Honesty

Goal: make the app's recorded history match reality.

Milestones:

### 3.1 SQL guard correctness

- Share a token-aware read-only SQL analyzer between dataset SQL and database connectors.
- Add fuzz tests and string-literal regression cases.
- Keep database and dataset query behavior consistent.

### 3.2 Agent bounds and history

- Add per-run wall-clock limit.
- Add context/token-aware history packing.
- Add long-loop stress test.
- Surface timeout clearly in UI and audit.

### 3.3 Job logs and long output

- Raise visible log tail.
- Persist full logs to job files.
- Add open-full-log action.
- Include job logs in redacted issue reports where appropriate.

### 3.4 Artifact mutation rollback

- Add rollback/audit parity for artifact archive, restore, delete, and regenerate.
- Snapshot previous artifact state before destructive actions.
- Expose artifact rollback or recovery path.

### 3.5 Git robustness

- Use differentiated timeouts for quick/status/diff/history/blame operations.
- Add defensive Git environment variables.
- Detect and explain ownership/safety errors.
- Preserve no-network/no-fetch default.

### 3.6 Encoding honesty

- Add better charset detection.
- Mark ambiguous encodings.
- Disable save until explicit encoding choice when ambiguity is risky.
- Preserve round-trip safety.

Exit criteria:

- Audit records are accurate for all user-visible mutations.
- Long jobs are debuggable without terminal fallback.
- SQL safety is token-aware.
- Agent runs are bounded and explain why they stopped.

## 7. Phase 4: JetBrains-Style UI Refactor

Goal: make the app match the target UI: professional, dense, native, resize-safe, and editor-first.

Milestones:

### 4.1 Shell controller extraction

- Introduce controller structure for major panels.
- Shrink `View` to layout registry and cross-controller coordination.
- Add typed event bus for shell events.
- Split oversized panel files.

### 4.2 Tool-window framework

- Implement typed tool-window registry.
- Left rail renders from registry.
- Right rail/internal assistant panes render from registry.
- Problems, Search, Git, Tasks, Jobs, Audit, Diagnostics, and Activity render as left-sidebar tool windows, not bottom panels.
- Keyboard shortcuts route through registry.

### 4.3 Left rail and tool windows

- Thin icon-first left rail.
- Active/collapsed state.
- Per-tool width memory.
- Project/Search/Problems/Git/Data/Artifacts/Jobs/Diagnostics first-class surfaces.

### 4.4 Center editor polish

- Editor-first empty state.
- Stable tabs and split layout.
- Better save state.
- Better diagnostics/outline/breadcrumb density.
- Large-content scroll behavior.

### 4.5 Right assistant polish

- Assistant header hierarchy.
- Source digest.
- Tool timeline.
- Approval recovery.
- Composer pinned to bottom.
- Sources/Lineage/Inspector secondary surfaces.

### 4.6 Remove bottom tool panel

- Remove the horizontal bottom tool panel entirely.
- Keep the bottom region as a subtle status bar only.
- Move Problems, Search, Git, Tasks, Jobs, Audit, Diagnostics, and Activity to the left sidebar/tool-window registry.
- Make each former bottom tool reachable in one click from the left rail and via keyboard.

### 4.7 Theme and density pass

- Remove hardcoded panel colors.
- Centralize spacing/density tokens.
- Apply restrained accent use.
- Add theme drift checks where practical.

### 4.8 Visual smoke

- Capture screenshots for first launch, workspace, editor, assistant, data, artifacts, settings, diagnostics, jobs, and approvals.
- Add manual or automated visual acceptance workflow.

Exit criteria:

- The app visually resembles the target JetBrains-style reference.
- Resize from desktop to laptop size does not break layout.
- Tool windows are single-click and keyboard reachable.
- The assistant feels integrated and native.
- UI files are maintainable enough for future work.

## 8. Phase 5: Data, Artifact, And Assistant Maturity

Goal: turn broad functionality into polished, reliable workflows.

Milestones:

### 5.1 Data workbench polish

- Data source tree improvements.
- Query editor/result grid polish.
- Result virtualization.
- Connector profile inspector.
- Query history UX.
- Export UX.

### 5.2 Artifact polish

- Artifact gallery improvements.
- Freshness and source coverage clarity.
- Regeneration coverage expansion.
- Artifact compare UX.
- DOCX/PPTX template polish.
- Cross-suite document/deck smoke.

### 5.3 Assistant retrieval quality

- Better context ranking.
- Better source diagnostics.
- Better uncited/cited source UX.
- Better stale-source prompts.
- Better model route suggestions.

### 5.4 Tool roadmap design

- Browser automation design.
- Interactive terminal design.
- PR platform tool design.
- MCP tool design.
- Automation/scheduler design.
- Plugin/trust design.

Exit criteria:

- Data and artifacts feel like real production workflows.
- Assistant answers are source-grounded and visibly trustworthy.
- Planned risky tools have clear designs before implementation.

## 9. Phase 6: Packaging And Release Trust

Goal: make release artifacts installable and trustworthy.

Milestones:

### 6.1 Release build pipeline

- Define repeatable release build process.
- Produce Windows, macOS, and Linux artifacts.
- Generate SHA-256 manifest.
- Embed version and commit metadata.

### 6.2 Signing and notarization

- Sign Windows executable/installer.
- Decide macOS signing/notarization path.
- Implement macOS package smoke.
- Define Linux package format and dependency notes.

### 6.3 SBOM and provenance

- Generate SBOM.
- Generate provenance evidence.
- Store release evidence with artifacts.
- Surface release trust in Diagnostics/About.

### 6.4 CI and platform smoke

- Windows CI validates format, tests, build, release manifest.
- Linux CI validates tests/build/package assumptions.
- macOS CI validates tests/build/package assumptions.
- Protected secret smoke per platform.
- Clean-machine smoke script/checklist.

### 6.5 Update visibility

- Add manual check for updates.
- No silent auto-install.
- Clear release notes.

Exit criteria:

- Clean Windows install launches without developer tools.
- macOS package launches under documented trust path.
- Linux package launches on supported distro target.
- Release metadata and checksums are available.

## 10. Phase 7: Private Beta

Goal: real users complete core workflows without developer guidance.

Milestones:

- First-run onboarding.
- Provider setup wizard.
- Sample workspace smoke.
- Beta feedback template.
- Crash recovery banner polish.
- User-facing safe-agent guide.
- Release notes written for users.
- Known limitations documented.

Exit criteria:

- At least five beta users complete the core v1 flow.
- Bugs are triaged within 48 hours.
- No P0 safety/data-loss bugs remain open.
- Packaging is trusted enough for the beta audience.

## 11. Phase 8: v1 Release Candidate

Goal: freeze scope and validate everything.

Milestones:

- Feature freeze.
- P0/P1 bug burn-down.
- Full cross-platform smoke.
- Security/safety review.
- Performance pass.
- Accessibility pass.
- Docs/readme/release notes final.
- Signed release candidate.

Exit criteria:

- All release-blocker tracker items checked.
- Clean-machine smoke passes on all target platforms.
- No known data-loss path remains.
- No known plaintext secret path remains.
- No hidden workspace-open side effects.
- Release artifacts are signed or clearly documented.

## 12. Validation Standards

Docs-only change:

```powershell
git diff --check
```

Code change:

```powershell
cd nexus-app
gofmt -l .
go test ./...
go build .
git diff --check
```

Windows local build-check without keeping a fresh unsigned executable:

```powershell
cd nexus-app
.\scripts\dev-env.ps1 -BuildCheck
```

Runnable local app build only when explicitly needed:

```powershell
cd nexus-app
.\scripts\dev-env.ps1 -Build
```

Windows native build prerequisite:

```powershell
go tool cgo -V
Test-Path "$(go env GOTOOLDIR)\cgo.exe"
winget install MSYS2.MSYS2
C:\msys64\usr\bin\bash.exe -lc "pacman -Syu --noconfirm"
C:\msys64\usr\bin\bash.exe -lc "pacman -S --needed --noconfirm mingw-w64-ucrt-x86_64-gcc mingw-w64-ucrt-x86_64-binutils mingw-w64-ucrt-x86_64-zlib"
```

If `Test-Path "$(go env GOTOOLDIR)\cgo.exe"` returns `False`, repair or reinstall Go before running the native build. A missing `cgo.exe` means the build will fail with `go: no such tool "cgo"` even when MSYS2 GCC is present.

If MSYS2 is not installed at `C:\msys64`, set `MSYS2_ROOT` before invoking the helper scripts. The expected compiler path is `C:\msys64\ucrt64\bin\gcc.exe` or `C:\msys64\ucrt64\bin\x86_64-w64-mingw32-gcc.exe`, with the same paths under `$env:MSYS2_ROOT` when overridden.

Windows CI/release-style checkpoint:

```powershell
cd nexus-app
.\scripts\ci-windows.ps1
```

`ci-windows.ps1` runs gofmt verification, `go test ./...`, `go vet ./...`, build metadata validation, native executable build, release manifest generation, and `git diff --check`. It removes generated unsigned artifacts in its cleanup block. If MSYS2 UCRT64 GCC or the Windows resource tools are missing, the script fails before tests and compilation; install `mingw-w64-ucrt-x86_64-gcc`, `mingw-w64-ucrt-x86_64-binutils`, and `mingw-w64-ucrt-x86_64-zlib` rather than trying `CGO_ENABLED=0`.

## 13. Risk Management

Top risks:

- UI refactor may expand scope. Mitigation: controller extraction one panel at a time.
- Safety fixes may reveal deeper design gaps. Mitigation: keep Phase 1 before feature work.
- Packaging trust requires external certificates/accounts. Mitigation: document cost and fallback early.
- Local toolchain differences can block builds. Mitigation: CI plus dev-env scripts.
- Planned tools can accidentally become executable too early. Mitigation: catalog drift tests and approval gates.
- Long sessions can degrade UI. Mitigation: streaming/activity throttles and stress tests.

## 14. Repeated Development Prompt

```text
Continue NexusDesk toward a production-ready Fyne-native app. Review docs/README.md and docs/06_TRACKER.md first. Pick the earliest highest-value unchecked item, prioritizing safety/data-loss, performance, audit honesty, JetBrains-style UI, agent tool completeness, durable jobs, and release trust. Implement one logical slice end-to-end with focused tests and docs/tracker updates. Preserve boundaries: services framework-free, Fyne only in app/UI/theme/brand, workspace open cheap, slow work via jobs, risky actions approval/audit/redaction/cancellation/rollback where practical. Validate with gofmt, go test ./..., go build . or the platform build-check script, and git diff --check. Remove generated binaries unless explicitly requested. Report changes, validation, remaining blockers, and progress.
```
