# NexusDesk Tracker

This tracker reflects the repository as it exists today and keeps planned work separate from created directories.

## Current Repository State

- [x] Product and engineering docs live under `docs/`.
- [x] Brand package lives under `docs/brand/`.
- [x] Wails application scaffold exists at `app/`.
- [x] React + TypeScript frontend exists at `app/frontend/`.
- [x] Go backend binding exposes startup state to the frontend.
- [x] Initial NexusDesk workbench shell replaced the Wails starter screen.
- [x] Brand SVG assets are copied into `app/frontend/src/assets/brand/`.
- [x] App shell uses NexusDesk symbol, horizontal logo, and domain icons.
- [x] App styles use NexusDesk color/type tokens from `app/frontend/src/brand-tokens.css`.
- [x] Wails app icon and Windows icon are sourced from the brand package.
- [x] Windows taskbar icon is generated as a multi-size ICO from high-resolution brand PNGs.
- [x] Icon generation script lives at `app/scripts/generate_windows_icon.py`.
- [x] Frontend startup state types live in `app/frontend/src/types.ts`.
- [x] Runtime brand asset mapping lives in `app/frontend/src/brand/assets.ts`.
- [x] Browser-safe fallback startup state lives in `app/frontend/src/data/startupState.ts`.
- [x] First reusable button, icon button, and status badge components live in `app/frontend/src/components/ui.tsx`.
- [x] First reusable card component lives in `app/frontend/src/components/ui.tsx`.
- [x] Reusable branded empty, loading, and inline alert states live in `app/frontend/src/components/ui.tsx`.
- [x] Shell layout is split into `app/frontend/src/features/shell/NexusDeskShell.tsx`.
- [x] Agent chat card is split into `app/frontend/src/features/shell/AgentChatCard.tsx`.
- [x] LLM settings card is split into `app/frontend/src/features/shell/LLMSettingsCard.tsx`.
- [x] Tool timeline is split into `app/frontend/src/features/shell/ToolTimeline.tsx`.
- [x] Workspace navigator is split into `app/frontend/src/features/shell/WorkspaceNavigator.tsx`.
- [x] Workbench panel is split into `app/frontend/src/features/shell/WorkbenchPanel.tsx`.
- [x] Workspace rail is split into `app/frontend/src/features/shell/WorkspaceRail.tsx`.
- [x] Agent panel is split into `app/frontend/src/features/shell/AgentPanel.tsx`.
- [x] Workspace scanner package exists at `app/internal/workspace/`.
- [x] Artifact writer package exists at `app/internal/artifact/`.
- [x] Scanner skips noisy folders, symlinks, and oversized/deep listings.
- [x] Workspace file preview is implemented in `app/internal/workspace/preview.go`.
- [x] File previews are rooted, traversal-checked, symlink-aware, size-limited, and UTF-8/text-only.
- [x] Common workspace images render as bounded inline previews.
- [x] Workspace PDFs render as bounded inline previews.
- [x] UTF-8 BOM and UTF-16 text previews are decoded.
- [x] BOM-less UTF-16 and Windows-1251 Cyrillic text previews are decoded.
- [x] Source preview headers show file type, decoded encoding, size, and truncation status when available.
- [x] CSV previews parse bounded rows/columns into a table while keeping raw text content available for chat context.
- [x] CSV previews show bounded column profiles with inferred type, missing count, distinct count, and numeric ranges.
- [x] CSV profile stats read a larger bounded file sample than the visible text preview.
- [x] Lightweight syntax highlighting exists at `app/frontend/src/features/shell/HighlightedCode.tsx`.
- [x] Desktop workspace picker is bound through `SelectWorkspace`.
- [x] Frontend switches from scaffold preview to indexed workspace nodes after folder selection.
- [x] Center workbench pane previews selected workspace text files.
- [x] Workspace refresh preserves the selected file when it still exists.
- [x] Workspace open/refresh auto-loads a preview for the selected or first file node.
- [x] Preview button reloads the selected workspace preview from disk.
- [x] Workspace navigator renders indexed nodes as an expandable tree.
- [x] Workspace navigator uses filesystem tree ordering instead of depth-grouped ordering.
- [x] Workspace navigator width can be resized with a drag handle.
- [x] Workspace navigator keeps fallback and file-tree rows aligned inside the resizable sidebar.
- [x] App shell stays fixed to the window while long navigator, preview, chat, settings, and timeline content scroll inside their panels.
- [x] Expanded workspace directories are reconciled and preserved across refreshes.
- [x] Backend remembers the selected workspace root for the session.
- [x] Refresh action rescans the active workspace through `RefreshWorkspace`.
- [x] Recent workspace store exists at `app/internal/storage/`.
- [x] Opened workspaces are persisted to local JSON config.
- [x] Frontend loads and displays recent workspaces.
- [x] Recent workspaces can be reopened through `OpenWorkspace`.
- [x] Recent workspaces can be removed individually or cleared.
- [x] LLM settings store exists at `app/internal/storage/llm_settings.go`.
- [x] LLM provider settings are persisted to local JSON config.
- [x] Saved LLM API keys are redacted before settings are returned to the UI.
- [x] Redacted LLM API keys are resolved only inside backend test/save flows that need the stored secret.
- [x] Agent panel includes a branded LLM provider settings form.
- [x] LLM connection probe exists at `app/internal/llm/`.
- [x] Agent panel can test an OpenAI-compatible `/models` endpoint.
- [x] LLM probe infers model-list, chat, embedding, vision, and reranking capability hints from provider model IDs.
- [x] LLM probe warns when the configured model is not returned by the provider.
- [x] Non-streaming OpenAI-compatible chat is implemented in `app/internal/llm/chat.go`.
- [x] Streaming OpenAI-compatible chat is implemented in `app/internal/llm/chat.go`.
- [x] Agent panel can send prompts through `AskLLM`.
- [x] Explain button sends a grounded explanation prompt for selected text/code previews.
- [x] Agent panel streams partial assistant responses through `nexusdesk:chat-stream` Wails events.
- [x] Selected workspace text previews can be attached as bounded chat context without sending image/PDF data URLs.
- [x] Selected CSV previews send a structured column profile and bounded sample as chat context.
- [x] Workspace chat history is persisted through `app/internal/storage/chat_history.go`.
- [x] Report button creates timestamped Markdown artifacts under `.nexusdesk/artifacts/`.
- [x] Markdown report artifacts are created without overwriting existing files.
- [x] Workbench artifact browser lists generated Markdown artifacts.
- [x] Artifact rows can select the generated report preview when visible in the workspace tree.
- [x] Helper services placeholder exists at `services/docker-compose.yml`.
- [x] Repository ignore rules exist in `.gitignore`.
- [x] Current and target directory structures are documented separately.
- [x] Production desktop build succeeds at `app/build/bin/app.exe`.

## Brand Integration

- [x] Use orbital NexusDesk symbol for compact app/navigation surfaces.
- [x] Use horizontal NexusDesk logo in the workspace header.
- [x] Use branded AI, code, data, document, and ops icons in navigation and cards.
- [x] Keep app runtime brand assets under `app/frontend/src/assets/brand/`.
- [x] Keep source brand package under `docs/brand/` as the design source of truth.
- [x] Keep brand asset imports centralized in `app/frontend/src/brand/assets.ts`.
- [x] Replace generated template font with an Inter-first system font strategy.
- [x] Add branded empty, loading, and inline alert states.
- [ ] Add branded approval states when approval flows are implemented.
- [ ] Add visual regression screenshots once the first interactive flows exist.
- [ ] Confirm final app icon pipeline for Windows/macOS/Linux packaging.
- [ ] Add macOS/Linux icon generation checks when packaging those targets.

## Next Work

- [x] Add a safe workspace folder picker.
- [x] Build a real file tree from approved workspace roots.
- [x] Add safe text file preview for selected workspace files.
- [x] Add safe image preview for selected workspace files.
- [x] Add basic PDF preview for selected workspace files.
- [x] Add lightweight syntax highlighting for text/code previews.
- [x] Add first bounded CSV table preview.
- [x] Add first bounded CSV column profiles.
- [x] Expand CSV profiling beyond the visible preview window with a larger capped sample.
- [x] Send structured CSV summaries as selected chat context.
- [x] Persist recent workspaces locally.
- [x] Add refresh behavior for the currently opened workspace.
- [x] Preserve selected file across refreshes.
- [x] Add expandable tree state once nested tree rendering replaces the flat indexed list.
- [x] Add recent workspace remove/clear actions.
- [x] Add local settings storage for LLM provider configuration.
- [x] Add LLM connection test.
- [x] Add LLM capability detection beyond model listing.
- [x] Add first non-streaming chat call with selected text context.
- [x] Persist chat history per workspace.
- [x] Add streaming chat responses.
- [x] Wire topbar Preview and Explain actions to real workspace/chat behavior.
- [x] Mask API keys before they leave the backend settings store.
- [ ] Migrate API keys into OS credential storage before production release.
- [x] Add first Markdown report artifact creation flow.
- [x] Add first artifact browser for generated Markdown reports.
- [x] Split brand-aware shell sections into smaller rail, navigator, workbench pane, agent panel, and timeline components when they need behavior.
- [x] Add first reusable button, icon button, and status badge components.
- [x] Add first reusable card component when panel extraction starts.
- [ ] Add backend module layout only when implementation files are created.
- [ ] Split the workbench UI into feature components once behavior lands.
- [ ] Replace the services placeholder with real development/test services when needed.
- [ ] Add automated frontend tests after interactive behavior exists.

## Directory Notes

`app/` contains the Wails desktop app. The current backend is intentionally small; create `internal/` packages incrementally as real workspace, settings, storage, indexing, and agent code lands.

`app/internal/workspace/` owns safe workspace scanning. It should keep ignore rules, depth limits, entry limits, and path safety close to the backend instead of trusting frontend filtering.

`app/internal/storage/` owns local app persistence. Recent workspaces and LLM settings currently use small JSON files in the user's config directory; settings storage can build on the same boundary until SQLite or OS credential storage is introduced.

`services/` is reserved for Docker Compose or supporting development services. It should not contain runtime app state; local service data belongs in ignored folders such as `services/data/`.

`docs/` remains the source of truth for product direction, architecture, delivery phases, developer experience, and brand assets.

`app/frontend/src/assets/brand/` contains copied runtime assets from `docs/brand/`. Update the docs source first when changing brand assets, then refresh the app copies deliberately.

## Verified Commands

Run these from `app/` on this Windows workstation:

```powershell
$env:NODE_OPTIONS='--use-system-ca --dns-result-order=ipv4first'
python scripts/generate_windows_icon.py
npm.cmd run build
go test ./...
wails build
```
