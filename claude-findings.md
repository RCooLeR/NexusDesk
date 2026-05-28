# Nexus Augentic Studio (NexusDesk) — Findings

**Reviewer:** Claude (Cowork)
**Date:** 2026-05-28
**Scope:** `nexus-app/` (active Fyne-native build). `app-wails/` is preserved as reference and intentionally not audited.
**Method:** Static reading of Go source, services, UI shell, docs, and `tracker.md`. The Fyne app could not be built in this sandbox (CGO/GLFW), so dynamic UI behavior is inferred from code.

Items are tagged with **Severity** (Critical / High / Medium / Low) and **Category** (Bug / Gap / Security / UX / Architecture / Performance / Docs).

---

## 1. Critical issues

### 1.1 No API key configurable through the UI — OpenAI-compatible auth is unreachable
**Files:** `nexus-app/internal/services/settings/settings.go`, `nexus-app/internal/services/llm/types.go`, `nexus-app/internal/ui/shell/settings_panel.go`
**Category:** Gap / UX
The `llm.Config` struct exposes `APIKey` and the HTTP client adds `Authorization: Bearer ...` when set, but `settings.Settings` has no `APIKey` field and `ConfigFromSettings` never populates it. The Settings panel (`settings_panel.go`) only renders Provider / Base URL / Model / Context tokens / Response reserve. Result: when `Provider = openai-compatible` (one of the two built-in choices), the user has no way to provide a key and every chat request will fail against any real OpenAI-compatible endpoint. The README and `docs/06_AI_AGENT_AND_LLM_STRATEGY` explicitly promise "configure an LLM base URL, model, **API key**, and capabilities," so this is also a documentation/contract mismatch.
**Suggested fix:** add `APIKey` to `Settings`, expose as a password-style `widget.NewPasswordEntry` in the settings form, persist via the existing settings store (which is already 0o600 in `store.go`), and store in OS keychain once `tracker.md`'s "Native protected secret storage" gate ships.

### 1.2 Agent can never run shell tools
**File:** `nexus-app/internal/ui/shell/assistant_panel.go` (line 296)
**Category:** Bug / Gap
`runAgentRequest` builds `agentSvc.Request{ApproveShell: false}` unconditionally. The `run_task` tool (`tools/defaults.go:194`) hard-rejects with "approval is required before running workspace tasks" whenever `ApproveShell` is false, and there is no UI affordance anywhere to flip it. As a consequence the `run_task` tool — registered, exposed in `toolDescriptors()`, and described in the agent prompt — is effectively dead. The model will keep trying it, the agent will keep returning approval errors, and the user has no path to grant approval.
**Suggested fix:** add a "Allow agent to run discovered tasks (this run)" checkbox to the assistant header or a per-tool approval prompt that fires when the model requests a high-risk tool.

### 1.3 Path-traversal protection is bypassed by symlinks on the read path
**Files:** `nexus-app/internal/services/workspace/path.go` (`resolveFile`, `isInside`)
**Category:** Security
`cleanRel` rejects literal `..` segments, but `resolveFile` uses `os.Stat` (not `os.Lstat`) and `isInside` compares the *requested* path, not its symlink-evaluated target. A file the user opens via `PreviewFile` whose path is a symlink pointing outside the workspace root will be read despite the "workspace path must stay inside the root" check. Write/append paths do call `os.Lstat` and `filepath.EvalSymlinks` (`write_path.go:32-58`) so writes are safe, but reads, search, context-pack, and preview are not. For a "local-first, safe" workbench this is the most concrete attack surface — a malicious project repo could exfiltrate `~/.ssh/` through a symlink simply by being browsed.
**Suggested fix:** mirror `ensureWriteParentInsideRoot` on the read path: reject symlinked components on every file read, or `filepath.EvalSymlinks(absTarget)` and re-check `isInside` against the evaluated path.

### 1.4 Metadata SQLite store reopens the DB on every call and re-runs schema bootstrap
**File:** `nexus-app/internal/services/metadata/store.go` (`open`, `Ensure`)
**Category:** Performance / Architecture
Every `SaveChatMessage`, `ListChatMessages`, `SearchChatMessages`, `SaveJob`, `SaveApprovalRecord`, etc. calls `s.open()`, which calls `s.Ensure()`, which runs: `os.MkdirAll`, `os.WriteFile(schema.sql)`, `db.Exec(schemaSQL)`, `ensureColumn`, an `INSERT OR UPDATE workspaces`, `listTables`, and rewrites the manifest JSON. Then it opens a fresh `sql.DB`. On a streaming chat or an agent loop with dozens of tool calls this is hundreds of redundant filesystem and SQL operations per user turn.
**Suggested fix:** open the `sql.DB` once per `Store` (lazily, then cached); call `Ensure` exactly once at workspace open. `sql.DB` is already a pool — it does not need to be reopened. Move manifest/schema writes out of the hot path.

---

## 2. High-severity issues

### 2.1 `activityText` grows unboundedly — long sessions will get slow then crash
**File:** `nexus-app/internal/ui/shell/activity.go`
**Category:** Bug / Performance
`addActivity` appends `"\n\n" + message` to `v.activityText` on every call and then `v.activityLog.ParseMarkdown(v.activityText)` re-parses the whole markdown string. There is a 400-line cap on `v.activityLines`, but `activityText` is never reset. A long Agent run (each tool start / done / error is a line), several editor operations, or a busy data session will quickly push the activity buffer into the multi-megabyte range. Each subsequent `addActivity` then re-parses that whole buffer in `ParseMarkdown` and replaces all rich-text segments, causing visible UI lag.
**Suggested fix:** rebuild `activityText` from `v.activityLines` each refresh, or store a `strings.Builder` that's truncated when the line cap kicks in.

### 2.2 Workspace search returns at most one match per file
**File:** `nexus-app/internal/services/workspace/search.go` (`searchFileContent`)
**Category:** Bug / UX
`searchFileContent` iterates lines but `return`s the first match it finds. For any file with multiple hits, the user only sees the first. That makes "Search workspace" feel broken for real code searches, especially in larger source files. Standard developer expectation is "all matches with line numbers."
**Suggested fix:** accumulate matches per file with a per-file cap (e.g. 5–10), then apply the global `maxResults` cap at the call site.

### 2.3 Model is a fixed `Select` — users can't enter the model they actually run
**File:** `nexus-app/internal/services/settings/settings.go` (`ModelOptions`), `settings_panel.go`
**Category:** Gap / UX
The Settings panel uses `widget.NewSelect(settingsSvc.ModelOptions(), ...)` with a hardcoded list of 7 model tags. Anyone running, for example, `llama3.2:3b`, `qwen2.5:32b-instruct`, `phi-3.5-mini`, a fine-tuned local model, or any OpenAI-style server with a custom model name cannot select their model. There is also no "test connection" button despite `llm.Probe` existing, so users cannot validate what the provider actually supports before they save.
**Suggested fix:** make the model an editable combobox (Fyne's `widget.NewEntry` with autocomplete, or pair the `Select` with a free-text override). Surface `llm.Probe` results behind a "Test connection" button that lists the models the endpoint actually returns.

### 2.4 ApproveWrites is a global, ambient flag — there is no per-call approval prompt
**Files:** `assistant_panel.go:295`, `tools/default_mutations.go`, `approvals/service.go`
**Category:** Security / UX
`runAgentRequest` sets `ApproveWrites = v.approvalService.HasFullProjectAccess(workspace.Root)`. If the workspace has full project access (granted via the Approvals UI), every subsequent mutating tool (`write_file`, `append_file`, `copy_file`, `move_file`, `delete_file`, `apply_patch`, `rollback_file_mutation`) is auto-approved silently for the entire access window. There is no live "do you want to allow this write?" dialog, no per-tool ledger preview before the write happens. The approval log only shows what already executed. For a "permissioned, auditable" product this is a coarse all-or-nothing gate.
**Suggested fix:** implement a per-call approval modal for high-risk tools, with sticky "Allow for this run" semantics. Keep the full-project-access policy as the "always-on" escape hatch.

### 2.5 Preview is hard-capped at 256 KB with no graceful truncation
**File:** `nexus-app/internal/services/workspace/preview.go` (`PreviewFile`, `defaultPreviewByteSize`)
**Category:** Bug / UX
Any file larger than `256 * 1024` bytes errors with `"file is too large for inline preview"` instead of returning a truncated preview. Real workspaces routinely contain `package-lock.json`, generated SQL dumps, large JSON, source files in the 300–500 KB range. The user simply gets an error dialog and can't open them at all.
**Suggested fix:** for `PreviewText`, fall back to reading the first `previewByteLimit` bytes and tag the preview as `Truncated`; only error for binary previews above the cap.

### 2.6 `textExtensions` and `isContextCandidate` are inconsistent and incomplete
**Files:** `nexus-app/internal/services/workspace/preview.go` (`textExtensions`), `nexus-app/internal/services/workspace/context.go` (`isContextCandidate`)
**Category:** Bug / Gap
`textExtensions` accepts ~19 extensions; `isContextCandidate` accepts ~42. So a `.sh`, `.java`, `.c`, `.cpp`, `.rb`, `.ini`, `.env`, `.conf` file: the agent will include it in a context pack (`isContextCandidate` says yes) but if the user clicks it in the navigator it goes through `previewKind()` → falls through `textExtensions` lookup → "if `utf8.Valid` then text else binary." That mostly works by luck but it's inconsistent, and explicitly fails for valid-UTF8 files that happen to contain a NUL byte (e.g. some `.log` rotations).

Also missing from both lists despite being mainstream: `.scss`, `.less`, `.vue`, `.svelte`, `.kt`, `.swift`, `.scala`, `.clj`, `.lua`, `.r`, `.dart`, `.zig`, `.makefile`/`Makefile` (no-extension), `.gradle`, `.cmake`, `.editorconfig`, `.gitignore`, `.gitattributes`, `.dockerignore`, `Dockerfile` (no extension), `.proto`, `.graphql`, `.lock`. `Makefile`, `Dockerfile`, `Jenkinsfile`, etc. are no-extension files entirely — neither matcher will handle them by name today.
**Suggested fix:** consolidate into one table keyed by ext **and** filename (case-insensitive), share between preview and context.

### 2.7 Windows-1251 fallback corrupts non-Russian non-UTF-8 files
**File:** `nexus-app/internal/services/workspace/encoding.go` (`decodeText`)
**Category:** Bug
If a file is neither UTF-8 nor UTF-16 (no BOM, no null pattern), `decodeText` unconditionally decodes via `charmap.Windows1251`. This is a hardcoded fallback tailored to the author's locale; for a Latin-1 / Windows-1252 / ISO-8859-15 / Shift-JIS / GB18030 file the result will be silently garbled. Worse, the function then *returns success* with `encoding = "windows-1251"`, so the user has no clue the bytes were misinterpreted, and a subsequent **save** will re-encode the corrupted string and persist the garbage.
**Suggested fix:** use a proper detector (golang.org/x/text/encoding/htmlindex with charset detection, or chardet), or refuse to decode and surface a "binary or unknown encoding" preview with a hex/byte view.

### 2.8 Approval log re-saves every record to the SQLite repo on every append
**File:** `nexus-app/internal/services/approvals/service.go` (`Append`)
**Category:** Bug / Performance
```go
if s.repository != nil {
    for _, record := range records {
        _ = s.repository.SaveApprovalRecord(record)
    }
}
```
After saving N records, the next append re-saves all N+1 (the JSON list is rebuilt each time and then every record is replayed into the SQLite repo). With 200-record cap, that's up to 200 SQLite upserts (each of which calls `Ensure`+`Open`+`Exec`) per approval action. Errors are silently swallowed with `_ =`. The JSON sidecar at `.nexusdesk/approvals/log.json` and the SQLite store also have no locking — two concurrent flows (e.g. user clicks "grant access" while an agent run is mid-flight) can corrupt the JSON.
**Suggested fix:** save only the new record to the repository; protect the JSON file with a `sync.Mutex` keyed by workspace root (or rely on SQLite alone and remove the JSON sidecar).

### 2.9 Folder-only "browse" — no way to open a single file
**File:** `nexus-app/internal/ui/shell/workspace_actions.go` (`openWorkspaceDialog`)
**Category:** Gap / UX
The only entry point is `dialog.ShowFolderOpen`. Many real flows want "open this one file" (script, log, PDF, CSV) without indexing a parent project. There's no recent-workspaces list and no command-line argument honored (`main.go` ignores `os.Args`).
**Suggested fix:** add a "Recent workspaces" submenu populated from `~/.NexusDesk/recent.json`, accept `nexusdesk path/to/folder` from CLI in `main.go`, and add a "Quick Open File" action that wraps a `dialog.ShowFileOpen` then routes through `PreviewFile`.

### 2.10 Bottom panel has 14 tabs — overload and discoverability problem
**File:** `nexus-app/internal/ui/shell/panels.go` (`newBottomPanel`)
**Category:** UX / Architecture
Tabs in the bottom panel today: Activity, Data, Operations, Search, Problems, Git, Tasks, Jobs, History, Chat, Agent Audit, Diagnostics, Artifacts, Rollbacks, Approvals = 14 (+1 Activity). With `TabLocationTop` on a 1440 px window many of these will overflow into the `...` menu, so the user can never see them all simultaneously. This is also what `docs/12_PROJECT_REVIEW.md` calls out as "internal/ui/shell carries a lot of orchestration state … reduce crowded action strips."
**Suggested fix:** group into a smaller set ("Workbench: Activity/Search/Problems/Tasks/Jobs", "Data: Data/Operations", "Source Control: Git/Rollbacks", "Audit: Chat/History/Agent Audit/Approvals/Diagnostics", "Artifacts") via a leading mode segmented control, or move Artifacts/Rollbacks/Approvals into the side rail.

### 2.11 `data_panel.go` is 2,722 lines — single-file UI controller
**File:** `nexus-app/internal/ui/shell/data_panel.go`
**Category:** Architecture
A single file with this many responsibilities (profile, query, SQL, notebook, dashboard, chart, SQLite, connector profiles, export, history, copy actions, keyboard nav) is a maintenance hazard. `docs/12_PROJECT_REVIEW.md` already flags `internal/ui/shell` as the biggest architectural risk; this file is its center.
**Suggested fix:** split into `data_panel/`(`profile.go`, `query.go`, `sql.go`, `notebook.go`, `chart.go`, `dashboard.go`, `sqlite.go`, `connectors.go`, `history.go`), keep `data_panel.go` as the composition root.

---

## 3. Medium-severity issues

### 3.1 Append flow's `defer file.Close()` error is dropped, and write failure leaves dangling rollback
**File:** `nexus-app/internal/services/workspace/write.go` (`ApplyFileAppend`)
**Category:** Bug
`defer file.Close()` ignores the error. More importantly, when `file.Write(encoded)` fails, the rollback record has already been **prepared** but not **committed** (`commitRollback` runs after the write), so the snapshot exists on disk under `.nexusdesk/rollbacks/<id>/...` but is never recorded in `log.json` — orphaned bytes that the rollback browser will never expose for cleanup.
**Suggested fix:** on write error, delete the prepared snapshot directory; check `file.Close()`'s return; or wrap both in a single transactional helper that always commits-or-discards.

### 3.2 `entryLimit = 600` silently truncates large directory listings without UI signal
**File:** `nexus-app/internal/services/workspace/service.go`
**Category:** Bug / UX
`ListChildrenWithOptions` stops after 600 entries per directory and bumps `EntryCap`, but the navigator widget never surfaces that. For a folder with 700 children the user sees 600 silently and may assume the rest are absent. `navigatorVisibilitySummary` shows `<X> shown` and `<Y> ignored hidden` but not `<Z> truncated by entry cap`.
**Suggested fix:** render `EntryCap` in the navigator footer ("3 items truncated by safety cap") and offer a "show all in this folder" override.

### 3.3 No editor "discard on close" guard
**Files:** `editor_chrome.go`, `tabs.go` (`CloseIntercept`)
**Category:** Bug / UX
`configureEditorTabs` wires `CloseIntercept` to `v.requestCloseTab(item)`. Whether `requestCloseTab` blocks closing of dirty tabs to prompt for save/discard is what `tracker.md` claims is implemented ("explicit discard confirmation for modified tabs"), but the implementation should be verified — given that `MarkDraftSaved` exists but no inline confirmation is visible in `editor_chrome.go`, a regression here would silently drop user work. Worth a focused integration test.

### 3.4 `refreshWorkspace` re-opens the whole workspace after every file action
**Files:** `workspace_actions.go` (`refreshWorkspace`), `workspace_file_actions.go` (`finishFileOperation`)
**Category:** Performance
Every successful create/copy/rename/delete calls `v.refreshWorkspace()`, which calls `openWorkspace(root)` again, which re-runs `workspace.Open()`, re-creates the metadata store, re-imports compatibility data, re-loads all chat history, refreshes jobs, approvals, audit, navigator, status. For a single-file rename this is enormous overkill — it also resets editor selection, closes the tree's open branches, and discards lazy-loaded directories.
**Suggested fix:** add `workspaceService.RefreshNode(parent)` and call only that.

### 3.5 Diff builder has no hunk windowing
**File:** `nexus-app/internal/services/workspace/write_diff.go` (`buildUnifiedDiff`, `lcsDiffLines`)
**Category:** Bug / Performance
The "unified diff" produced before every safe write includes every unchanged line as a context line. For a 5,000-line file with a 10-line change, the diff is ~5,000 lines and the LCS table is O(N²) memory. Real unified diffs window to `@@ -L,n +L,m @@` hunks with a few lines of context.
**Suggested fix:** generate proper hunks (3-line context window is conventional). For files exceeding a threshold, fall back to "elided diff" with line counts only.

### 3.6 `chatTurnPreview` can panic on empty role
**File:** `nexus-app/internal/ui/shell/assistant_panel.go` (`chatTurnPreview`)
**Category:** Bug (latent)
The function defaults `role` to `"turn"` if empty, then runs `strings.ToUpper(role[:1])`. Safe today, but the check uses `strings.TrimSpace(turn.Role) == ""` after lowercasing — fine. Still, the same pattern in `titleAction` (workspace/write.go:191) uses `action[:1]` with no empty-string fallback for arbitrary callers. A future caller passing an empty action will panic.
**Suggested fix:** guard `[:1]` with `len(value) == 0` checks or use `strings.Title` via `golang.org/x/text/cases`.

### 3.7 No "test LLM connection" UI even though `Probe` exists
**Files:** `nexus-app/internal/services/llm/probe.go`, `settings_panel.go`
**Category:** Gap / UX
`Client.Probe` is implemented end-to-end (model list, capabilities, Ollama runtime status, warnings). The Settings panel doesn't expose a button for it, only "Save." Diagnostics panel calls it (`diagnostics_panel.go:60+`) but that's behind a different tab and only after a workspace is open. First-time users have no model-list confirmation at the moment they pick a provider/model.
**Suggested fix:** add "Test connection" next to "Save" in the settings form; surface model count, warnings, and Ollama runtime status inline.

### 3.8 Approval/job persistence silently drops repository errors
**Files:** `approvals/service.go` (`Append`), `jobs/service.go` (`persistLocked`)
**Category:** Bug
Both use `_ = s.repository.SaveX(record)`. If the underlying SQLite write fails (disk full, schema drift, locked DB), the user has no clue — only the JSON sidecar gets updated and the audit trail diverges silently. Diagnostics panel will not pick this up because it only inspects metadata state, not persistence errors.
**Suggested fix:** capture and route errors into the Activity log (or a dedicated "Persistence" surface in Diagnostics).

### 3.9 Streaming chat does not stop on `ctx.Done()` in `readChatCompletionStream`
**File:** `nexus-app/internal/services/llm/chat.go` (`readChatCompletionStream`)
**Category:** Bug
The scanner loop reads from `response.Body` but doesn't check for the request context being canceled. If the user cancels (or closes the workspace) mid-stream, the goroutine will keep draining the response until the upstream sends `[DONE]` or closes the socket. The HTTP client honors the context for the connection, so it will eventually unblock, but interim delta writes can race against UI teardown.
**Suggested fix:** select on `ctx.Done()` between scans, or run `response.Body.Close()` when `ctx.Err() != nil`.

### 3.10 Provider list is closed (`ollama`, `openai-compatible`) and there is no "custom"
**File:** `nexus-app/internal/services/settings/settings.go` (`ProviderOptions`)
**Category:** Gap
`shouldSendOllamaOptions` already does provider-by-string-match for Ollama-specific options, but the user is locked into two labels. Locally hosted vLLM, LM Studio, llama.cpp server, Text Generation WebUI, etc. all behave a bit differently around `options`/`num_ctx`/`num_predict`. There's no way to mark a provider that *is* OpenAI-compatible but doesn't accept the Ollama `options` field, which today gets sent any time the URL contains `localhost:11434` or the provider name contains `local`.
**Suggested fix:** make providers extensible (read from settings file too) and add explicit toggles for "send Ollama options" / "supports streaming" / "supports system role" instead of string heuristics.

### 3.11 `agentContextBudgetBytes` returns 4 when budget is non-positive
**File:** `nexus-app/internal/ui/shell/assistant_panel.go` (`agentContextBudgetBytes`)
**Category:** Bug
`return 4` (literal four bytes) when `budgetTokens <= 0`. Probably a typo for `4 * charsPerTokenEstimate` or `defaultAgentContextMaxBytes`. With a 4-byte budget the context pack is effectively empty, the user gets a "context pack was capped" warning every time, and the agent runs blind. Same shape in `assistant.contextBudgetBytes` returns `charsPerTokenEstimate` (= 4) — also nearly empty.
**Suggested fix:** clamp to `defaultAgentContextMaxBytes` when the user's settings produce a non-positive remaining budget, and emit a warning that response reserve is misconfigured.

### 3.12 `Ctrl+C` shortcut is captured globally
**File:** `nexus-app/internal/ui/shell/shortcuts.go` (`shortcutCopy`)
**Category:** Bug / UX
`shortcutCopy` binds `Ctrl+C` at the canvas level and dispatches to `v.copySelection`. Fyne's `widget.Entry` (used for the editor and many forms) handles Ctrl+C itself for text selection copying; binding it at the canvas level can hijack copy-from-editor depending on `copySelection`'s implementation. Risk of frustrating "copy doesn't work in the editor" complaints.
**Suggested fix:** only bind `Ctrl+C` when no widget has focus, or remove the global shortcut and rely on per-widget defaults.

### 3.13 Search-result snippet drops the whole line including the match position
**File:** `nexus-app/internal/services/workspace/search.go` (`trimSearchSnippet`)
**Category:** UX
The snippet is built by joining all whitespace into single spaces, then trimming to 160 runes from the start. If the match is at byte 200 of a long line, the user sees the *beginning* of the line, never the match. Standard search snippets show a window centered on the match.
**Suggested fix:** find the first match position in the line, then take ±80 runes around it.

### 3.14 `cleanRel` does not call `filepath.Clean`
**File:** `nexus-app/internal/services/workspace/path.go` (`cleanRel`)
**Category:** Bug (latent)
`cleanRel` checks for `..` segments by string operations only. Edge inputs like `foo/./..` are *not* the same as `..` and *do* contain `/../` so they're correctly rejected today. But inputs like `foo//bar` collapse to `foo/bar` only after `filepath.Join`, which is fine. Subtle case: `cleanRel("foo/.")` returns `"foo/."`. Downstream `filepath.Join` collapses it but the *clean* relpath kept in `WorkspaceNode.ID`/preview cache is `"foo/."`, which won't deduplicate with `"foo"`.
**Suggested fix:** apply `filepath.Clean` after `cleanRel` checks, then re-check the result for traversal.

### 3.15 Append safety check reads only 4 KB, false-positive on UTF-16 with BOM later in file
**File:** `nexus-app/internal/services/workspace/write.go` (`ensureAppendTargetSafe`)
**Category:** Bug (edge case)
Reads first 4096 bytes and decides via `looksBinary` + UTF-16 heuristics. For an UTF-16 file with leading binary header bytes (uncommon but legal), or a sparse file where the first 4 KB are zeros, the check rejects the append even though the file would be safe.
**Suggested fix:** read more (32–64 KB) and combine the binary check with the encoding detection from `decodeText`.

---

## 4. Lower-severity / polish

### 4.1 `tracker.md` and `docs/13_PRODUCTION_READINESS.md` list many unchecked items as "required for Gate 1"
**Files:** `tracker.md`, `docs/13_PRODUCTION_READINESS.md`
**Category:** Docs / Planning
Outstanding items explicitly marked required before Native Parity Beta: IDE-grade editor (syntax highlighting, find/replace, split groups, breadcrumbs, outline), external DB profiles, native secret storage, assistant quality parity, UI cleanup, Wails inventory. None of these have started landing yet, so the gate definition and the actual code are 6–12 months apart in scope. The tracker is honest about this; just calling it out so the gap is visible in this report.

### 4.2 `welcomePanel()` says only "Open a workspace to begin" with no link
**File:** `nexus-app/internal/ui/shell/tabs.go`
**Category:** UX
The welcome tab is a single centered markdown block with no Open Workspace button. New users have to discover the toolbar action or the File menu. Onboarding (also a tracker gap, `tracker.md` Phase 8) starts here cheaply.

### 4.3 About dialog says "Fyne-native migration build" — production users shouldn't see this
**File:** `nexus-app/internal/ui/shell/menu.go` (`showAbout`)
**Category:** UX / Docs
Tagline is acceptable now, but when the app ships, "migration build" should not be in the About text. Should pull from a versioned build constant.

### 4.4 `tracker.md` cross-platform support undefined
**File:** `tracker.md`, `docs/13_PRODUCTION_READINESS.md`
**Category:** Gap
Build/install lifecycle is explicitly "Windows first." The Fyne stack and the `_windows.go` / `_other.go` conditional files (e.g. `services/tasks/command_*.go`, `services/git/process_*.go`) already mean macOS and Linux *should* compile, but no CI proves it. For a local-first privacy-focused tool, Linux users are a core audience.

### 4.5 Hardcoded `qwen2.5-coder:14b` default with a 32 K context
**File:** `nexus-app/internal/services/settings/settings.go` (`Defaults`)
**Category:** UX
Many users won't have that exact model pulled, so first run will surface a "Configured model was not returned by the provider" warning until they change settings. Defaulting to "no model selected" with a clear "go pick a model" call-to-action is friendlier.

### 4.6 `agent/state.go` `claimsMutation` regex is English-only and very fuzzy
**File:** `nexus-app/internal/services/agent/state.go` (`claimsMutation`, `guardMutationClaim`)
**Category:** Bug (UX)
The safety wrapper that appends "this run did not receive a successful mutating tool observation" is keyed to English phrases (`created`, `wrote`, `written`, `saved` etc.). A model replying in another language, or wording like "stored", "persisted", "produced" — slips past the guard. Negative phrases like `"did not modify"` are correctly suppressed but only in English.
**Suggested fix:** ground the check on whether any tool result had `Mutated=true`; if not, always prepend a non-conditional verification note rather than try to outguess the model's prose.

### 4.7 `service.go` for `tasks` only allows three task kinds — extending requires editing whitelist
**File:** `nexus-app/internal/services/tasks/run.go` (`validateRunnableTask`)
**Category:** Gap
Only `npm-script`, `go-test`, `compose config` are runnable. Discovery surfaces them as "tasks" but executing anything else returns "not runnable by the safe task runner." With approval logic in place, more kinds (pytest, cargo, just, make, bun, pnpm/yarn) should be runnable. This is also called out in `tracker.md`.

### 4.8 `runDiscovered` uses `sh -c` / `cmd /C` with the discovered string — quoting via `quotePath`
**Files:** `tasks/run.go`, `tasks/paths.go` (`quotePath`)
**Category:** Security (low risk, narrow attack surface)
Discovered commands today are constructed only from npm script names, Go test paths, and Compose filenames in the workspace. `quotePath` only handles spaces/tabs/double-quotes. A maliciously crafted `package.json` script name containing backticks or `$()` could inject shell when passed through `sh -c`. Risk is bounded because the user must explicitly approve via shell approval (which, per 1.2, isn't reachable yet from the UI), but it's worth fixing before that gate opens.
**Suggested fix:** when running discovered tasks, use `exec.Command(argv0, argv1, ...)` directly rather than shelling out; for npm, exec `npm run <script>` as a `["npm","run",name]` argv rather than a `sh -c` string.

### 4.9 `decodeText` Windows-1251 fallback re-checks no decoder error, but `charmap.Windows1251.NewDecoder().Bytes()` never returns an error in practice
**File:** `workspace/encoding.go`
**Category:** Minor
Dead error path; not buggy but misleading.

### 4.10 Schema written next to the DB on every `Ensure` call
**File:** `metadata/store.go` (`Ensure`)
**Category:** Performance / Polish
`os.WriteFile(schemaPath, []byte(schemaSQL), 0o644)` runs every time the store is opened. The SQL string is constant; this only needs to happen on first init or on schema-version change.

### 4.11 `chat.go` `systemPrompt()` is a single sentence with no localization, no role discipline
**File:** `services/llm/chat.go`
**Category:** Polish
"You are Nexus, the assistant inside Nexus Augentic Studio." is fine; but the agent prompt assembly in `services/agent/prompt.go` and `runtimePrompt` carries the heavier instructions. Make sure the *Ask* mode has a separate system prompt from the *Agent* mode; today both paths share the same `systemPrompt`. That makes Ask responses lean noticeably "agent-shaped."

### 4.12 No keyboard shortcut for "Open file directly" / "Quick Open" (file picker by typed name)
**Category:** UX
Ctrl+P quick-file-open is table-stakes for IDE-style products. Not wired today.

### 4.13 Welcome → opening a workspace doesn't close the welcome tab
**Files:** `editor_chrome.go`, `tabs.go`
**Category:** UX
The welcome tab persists after `openWorkspace`. Minor noise; either close it on first file open or convert it to a workspace-aware home tab with recent files.

---

## 5. What's strong (so the review isn't only negative)

- Clear separation of `internal/services/*` (framework-free) from `internal/ui/shell/*` (Fyne). The Wails-to-Fyne migration is genuinely cleaner.
- `Workspace.ApplyFileWrite` honors symlink protection on writes (`write_path.go`'s `ensureWriteParentInsideRoot` with `EvalSymlinks`), refuses metadata writes, and creates a rollback snapshot with checksum verification before each mutation.
- The rollback model is well-thought-out: SHA-256 verified backups, "applied once" enforcement, restored/removed split, recovery path for corrupt metadata.
- Agent state machine is conservative: emergency iteration cap (64), bounded observation bytes, "claimed mutation" guard, finalization fallback.
- Datasets/SQLite layer has notable depth (notebooks, charts, dashboards, profiles, history, connector profiles).
- Metadata layer can detect corruption and self-archive (`archiveCorruptMetadataStore`).
- Approvals model has explicit time-boxed full-project-access with auto-expiry, and full audit trail per action.

---

## 6. Suggested triage order

1. **1.3** symlink read protection — security floor.
2. **1.1** API key in settings — blocking real OpenAI-compatible usage.
3. **1.2** agent shell-approval path — currently dead code.
4. **2.1** activity log unbounded growth — visible degradation in long sessions.
5. **2.2** search shows one match per file — feels broken to first-time users.
6. **2.5 / 2.6 / 2.7** preview/encoding correctness — the workbench has to read files reliably.
7. **1.4** metadata DB lifecycle — perf cliff under load.
8. **2.10 / 2.11** bottom-panel overload + `data_panel.go` split — the next refactor pass to keep velocity sane.
9. **2.4** per-call agent approval modal — needed before "approved shell" or "approved Docker" gates open.

---

## 7. Notes on confidence

This review is static. The Fyne build was not exercised; UI sizing, focus traps, drag behavior, scrolling responsiveness, theming under high-DPI, and platform-specific dialog behavior would need a runtime pass. Items flagged "latent" or "edge case" are inferred from code shape, not reproduced.

Tests exist under each service package (`*_test.go`) but several flagged behaviors here (e.g. search-one-match-per-file, activity unbounded growth, Windows-1251 false positives, approval re-save N×N) appear not to have regression coverage. Adding tests for each finding before fixing it would be the lowest-risk way to retire the list.
