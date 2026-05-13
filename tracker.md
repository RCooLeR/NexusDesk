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
- [x] Shell layout is split into `app/frontend/src/features/shell/NexusDeskShell.tsx`.
- [x] Workspace scanner package exists at `app/internal/workspace/`.
- [x] Scanner skips noisy folders, symlinks, and oversized/deep listings.
- [x] Workspace file preview is implemented in `app/internal/workspace/preview.go`.
- [x] File previews are rooted, traversal-checked, symlink-aware, size-limited, and UTF-8/text-only.
- [x] Desktop workspace picker is bound through `SelectWorkspace`.
- [x] Frontend switches from scaffold preview to indexed workspace nodes after folder selection.
- [x] Center workbench pane previews selected workspace text files.
- [x] Workspace refresh preserves the selected file when it still exists.
- [x] Workspace open/refresh auto-loads a preview for the selected or first file node.
- [x] Backend remembers the selected workspace root for the session.
- [x] Refresh action rescans the active workspace through `RefreshWorkspace`.
- [x] Recent workspace store exists at `app/internal/storage/`.
- [x] Opened workspaces are persisted to local JSON config.
- [x] Frontend loads and displays recent workspaces.
- [x] Recent workspaces can be reopened through `OpenWorkspace`.
- [x] Recent workspaces can be removed individually or cleared.
- [x] LLM settings store exists at `app/internal/storage/llm_settings.go`.
- [x] LLM provider settings are persisted to local JSON config.
- [x] Agent panel includes a branded LLM provider settings form.
- [x] LLM connection probe exists at `app/internal/llm/`.
- [x] Agent panel can test an OpenAI-compatible `/models` endpoint.
- [x] LLM probe infers model-list, chat, embedding, vision, and reranking capability hints from provider model IDs.
- [x] LLM probe warns when the configured model is not returned by the provider.
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
- [ ] Replace generated template font with a local Inter asset or approved system font strategy.
- [ ] Add branded empty, loading, approval, and error states.
- [ ] Add visual regression screenshots once the first interactive flows exist.
- [ ] Confirm final app icon pipeline for Windows/macOS/Linux packaging.
- [ ] Add macOS/Linux icon generation checks when packaging those targets.

## Next Work

- [x] Add a safe workspace folder picker.
- [x] Build a real file tree from approved workspace roots.
- [x] Add safe text file preview for selected workspace files.
- [x] Persist recent workspaces locally.
- [x] Add refresh behavior for the currently opened workspace.
- [x] Preserve selected file across refreshes.
- [ ] Add expandable tree state once nested tree rendering replaces the flat indexed list.
- [x] Add recent workspace remove/clear actions.
- [x] Add local settings storage for LLM provider configuration.
- [x] Add LLM connection test.
- [x] Add LLM capability detection beyond model listing.
- [ ] Mask or migrate API keys into OS credential storage before production release.
- [ ] Split brand-aware shell sections into smaller rail, navigator, workbench pane, agent panel, and timeline components when they need behavior.
- [ ] Add first reusable button, icon button, card, and status badge components.
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
