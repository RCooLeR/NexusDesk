# UI Workbench

Status: canonical UI and UX target for NexusDesk.

## 1. Design Contract

NexusDesk must feel like a professional JetBrains-style native workbench:

- three-column layout;
- compact top menu and toolbar;
- thin left tool stripe;
- project/database/tool window on the left;
- large central editor canvas;
- right AI chat and agent panel;
- subtle status bar;
- low-noise dark theme;
- strong resize behavior;
- keyboard-first navigation;
- dense, calm, serious visual language.

The workbench is the product. The assistant belongs inside the workbench. The app must not feel like a dashboard, a debug console, a chat page, or a file browser with extra panels.

## 2. Target Screen Layout

```text
+------------------------------------------------------------------------------------+
| File Edit View Navigate Code Refactor Run Tools Help        workspace branch model |
+------------------------------------------------------------------------------------+
| Open Workspace | Refresh | Git | Run Task | Search workspace                 gear |
+---+---------------------------+--------------------------------+-------------------+
|   | Project                   | Editor tabs                    | AI Chat           |
| L | ------------------------- | ------------------------------ | ----------------- |
| e | workspace root            | file.go  README.md  data.csv   | route/model       |
| f |  src/                     |                                | sources/context   |
| t |  docs/                    | Large editor / preview canvas  | messages          |
|   |  data/                    |                                | tool timeline     |
| r |                           | Empty state when no file:      | approvals         |
| a | Search                    | Project View Alt+1             | composer          |
| i | Problems                  | Go to File Ctrl+P              | send/stop         |
| l | Git                       | Search Everywhere Ctrl+Shift+P |                   |
|   | Data Sources              | Drop files to open             |                   |
|   | Artifacts                 |                                |                   |
|   | Operations                |                                |                   |
|   | Tasks                     |                                |                   |
|   | Jobs                      |                                |                   |
|   | Audit                     |                                |                   |
|   | Diagnostics               |                                |                   |
|   | Activity                  |                                |                   |
+---+---------------------------+--------------------------------+-------------------+
| workspace | branch | model route | jobs | warnings | file | encoding | version        |
+------------------------------------------------------------------------------------+
```

The exact pixel sizes can change, but the hierarchy cannot:

1. The center editor canvas is primary.
2. The left side is navigation and tools.
3. The right side is assistant and agent work.
4. The top is command and context.
5. The bottom is status only. Problems, Search, Git, Tasks, Jobs, Audit, Diagnostics, and Activity live in the left sidebar.

## 3. Window Regions

### 3.1 Top menu

Required menu groups:

- File: open workspace, open file, save, save all, close tab, recent workspaces, settings, exit.
- Edit: undo/redo where available, cut/copy/paste, find, replace, select all.
- View: tool windows, appearance, zoom, split editor, focus editor, toggle left tool windows.
- Navigate: quick open, command palette, go to symbol, go to definition, find references, next/previous problem.
- Code: format, diagnostics, explain selection, generate tests, refactor actions when available.
- Refactor: rename/move/extract actions as they land.
- Run: discovered tasks, run current task, stop job, rerun last job.
- Tools: data tools, artifact tools, issue report, release diagnostics.
- Help: user guide, diagnostics, about, check for updates.

Rules:

- One compact row.
- Text labels are short.
- Shortcut hints should be visible in menus.
- No oversized logo in the menu row.

### 3.2 Toolbar

The toolbar is a dense command strip, not a hero banner.

Required controls:

- workspace picker/name;
- refresh;
- Git branch/status summary;
- run task selector;
- global search field;
- model/provider readiness indicator;
- settings button;
- diagnostics warning indicator.

Rules:

- The toolbar must stay one row.
- Controls collapse gracefully on small widths.
- The toolbar must not push the editor downward with multi-line status text.
- Readiness details belong in Diagnostics, not across the toolbar.

### 3.3 Left tool stripe

The left stripe is icon-first and narrow. It should feel like an IDE rail.

Default icons, top to bottom:

- Project
- Search
- Problems
- Git
- Data Sources
- Artifacts
- Operations
- Tasks
- Jobs
- History
- Approvals
- Diagnostics
- Activity

Rules:

- Active icon has clear accent state.
- Hover has a small tooltip and shortcut.
- `Alt+1` through `Alt+9` and `Alt+0` toggle primary tools.
- Clicking the active icon collapses the left tool window.
- Width target: about 32 to 44 px, depending on DPI.
- No text labels in the stripe itself.

### 3.4 Left tool window

The left tool window changes based on the selected stripe icon.

General behavior:

- Resizable with a visible handle.
- Minimum width around 220 px.
- Comfortable width around 280 to 360 px.
- Width remembered per tool window.
- Collapsible.
- Keyboard focus visible.
- Header includes title, tiny actions, and optional filter.

Project tool window:

- Tree with disclosure arrows.
- File/folder icons.
- Git badges.
- Active file highlight.
- Context menu for create, rename, copy, move, delete, reveal, add to context, open externally when safe.
- Lazy child loading and caps.
- Ignored files hidden by default with toggle.
- Missing/unreadable items surfaced without crashing.

Search tool window:

- Query field.
- Path/content toggle.
- Regex/case options.
- Result list grouped by file.
- Snippets with line numbers.
- Cancel in-progress search.
- Search performance status.

Problems tool window:

- Problems grouped by file and severity.
- Sources include markers, syntax diagnostics, conflicts, generated diagnostics.
- Click jumps to file/line where possible.
- Filter by severity/source.

Git tool window:

- Branch summary.
- Changed files grouped by directory.
- Stage/unstage actions.
- Diff/hunk preview.
- Commit message composer.
- History and blame entry points.
- Conflict resolution actions.

Data Sources tool window:

- Data source tree like a database IDE.
- Workspace datasets.
- Workspace SQLite files.
- External read-only profiles.
- Schema/table nodes.
- Query history.
- Add/test/edit profile actions.

Artifacts tool window:

- Artifact list by type/status/freshness.
- Search/filter.
- Preview action.
- Lineage action.
- Regenerate where supported.
- Archive/restore/delete with audit.
- Pin to assistant context.

Operations tool window:

- Read-only evidence tree: Dockerfiles, Compose, env/config, scripts, logs.
- Inspection summaries.
- Generate runbook action.
- Mutating system actions stay unavailable until a separate approved design lands.

Tasks tool window:

- Discovered tasks grouped by source.
- Run button with approval when needed.
- Last run status.
- Output/job link.

Jobs tool window:

- Job list with status and progress.
- Tail logs.
- Cancel/retry/open-output actions.
- Filters by status and kind.

Diagnostics tool window:

- Health cards.
- Provider/model status.
- Protected secret status.
- Metadata status.
- Tool registry status.
- Release trust status.
- Export issue report action.

### 3.5 Center editor canvas

The editor canvas is the largest and most stable region.

Required behavior:

- File tabs with dirty, pinned, close, and active states.
- Split editor support.
- Empty state with quiet keyboard hints.
- Text editor and preview modes.
- Markdown source/render mode.
- Data/document/image/binary preview surfaces.
- Breadcrumbs.
- Outline/document map.
- Find/replace.
- Formatting actions.
- Diagnostics and problem navigation.
- Definition/reference navigation.
- Save/revert state.
- Encoding and line-ending indicators.
- Truncation/read-only warnings.

Empty state should be calm:

```text
Project View        Alt+1
Go to File          Ctrl+P
Search Everywhere   Ctrl+Shift+P
Recent Files        Ctrl+E
Command Palette     Ctrl+Shift+P
Drop files here to open them
```

Rules:

- The center pane gets resize priority.
- Side pane changes must not reflow editor content wildly.
- Cursor and scroll should survive save, refresh, and preview updates where possible.
- Wide content uses horizontal scrolling inside the editor region, not window overflow.

### 3.6 Right assistant panel

The right panel is always assistant-first. It may contain internal tabs or secondary overlays, but the default is chat and agent work.

Required sections:

- Header: current mode, route/model, new chat, stop/cancel, settings.
- Context strip: pinned files/folders/artifacts, token budget, source quality.
- Message list: streamed answers, citations, weak evidence warnings, tool outputs.
- Tool timeline: plan, requested tools, approvals, execution result, errors.
- Approval affordances: approve/deny/details for pending risky actions.
- Composer: multiline input, Ask/Agent toggle, model route, attach context, send.

Secondary right surfaces:

- Sources: cited and uncited source coverage, snippets, stale sources.
- Lineage: artifact dependency graph and regeneration status.
- Inspector: current file/artifact/job/selection details.
- Job Monitor: active long-running work and logs.

Rules:

- Composer pins to bottom.
- Stop/cancel is always visible during streaming or agent work.
- Source/citation quality is visible without expanding debug panels.
- Chat should look native and restrained, not like a web embed.

### 3.7 Status bar

The status bar is one quiet line.

Required fields:

- workspace name/path hint;
- Git branch;
- model route/provider readiness;
- active jobs count;
- warning count;
- active file path or selection;
- encoding and line endings;
- app version.

Rules:

- Muted colors.
- No wrapping.
- Clickable only where it opens a specific panel.
- Errors are summarized; details belong in Diagnostics.

## 4. Visual System

### 4.1 Theme direction

Default theme: dark, low-noise, layered.

Target palette direction:

```text
App background:     deep blue-black / neutral charcoal
Panel background:   slightly lighter than app background
Editor background:  distinct but calm central surface
Raised surfaces:    dialogs, popovers, menus
Borders:            low-contrast but visible
Text primary:       high-contrast neutral
Text secondary:     muted neutral
Text tertiary:      very muted neutral
Accent:             restrained blue
Selection:          accent-tinted overlay
Success:            muted green
Warning:            amber
Error:              restrained red
```

Rules:

- Accent is used for active tab, focus ring, selected route, primary action.
- Error color is reserved for actual errors.
- Warning color is reserved for recoverable or risky states.
- No bright brand blocks in the everyday workbench.
- No decorative gradients in the workbench chrome.
- No emoji in chrome. Icons should be consistent and native-feeling.

### 4.2 Density

The app is for power users. Density should be high but not cramped.

Rules:

- Compact row height in trees and lists.
- Small but readable typography.
- Padding is consistent and tokenized.
- Buttons in chrome are compact.
- Dialogs can be spacious when configuration is complex.
- First-launch UI should not show every diagnostic detail at once.

### 4.3 Motion

Motion should be functional and rare.

Allowed:

- small focus transitions;
- spinner/progress for active jobs;
- subtle saved pill fade;
- panel collapse/expand if performant.

Not allowed:

- decorative animation;
- bouncing panels;
- motion that hides state changes;
- streaming updates faster than the UI can render smoothly.

## 5. Resize Behavior

Resize must be excellent. Broken resizing makes the app feel unfinished no matter how many features exist.

Targets:

- Default window: around 1280 x 820.
- Minimum supported working size: 1024 x 640.
- Comfortable desktop: 1600 x 900 and above.
- High-DPI behavior follows OS scaling.

Rules:

- Center editor keeps priority.
- Left and right panes have minimum widths and collapse when needed.
- The left and right panes collapse before the editor becomes unusable.
- Long text wraps in panels.
- Logs and tables scroll inside bounded containers.
- Wide data grids use horizontal scrolling inside the grid.
- No surface may force the whole window wider than common laptop screens.
- Pane widths are remembered per workspace and tool where practical.

Critical resize states to test:

- first launch no workspace;
- workspace open with project tree;
- editor open with right assistant visible;
- database tool window open;
- Jobs or Diagnostics selected in the left sidebar;
- settings dialog open;
- approval dialog open;
- long chat answer streaming;
- large data grid visible;
- 1024 x 640 laptop mode.

## 6. Keyboard Model

Minimum shortcuts:

- `Ctrl+O`: open workspace.
- `Ctrl+R`: refresh.
- `Ctrl+S`: save.
- `Ctrl+W`: close tab.
- `Ctrl+F`: find in file.
- `Ctrl+Shift+F`: search workspace.
- `Ctrl+P`: quick open.
- `Ctrl+Shift+P`: command palette.
- `Ctrl+,`: settings.
- `Alt+1..9`: primary left tool windows.
- `Alt+0`: diagnostics or activity.
- `Ctrl+Tab`: next editor tab.
- `Ctrl+Shift+Tab`: previous editor tab.
- `Esc`: close popover/dialog/find where safe.

Keyboard rules:

- Every visible tool window must have a focus path.
- Dialogs must trap focus correctly.
- Escape should back out one layer, not destroy work.
- Command palette should expose every meaningful action.
- Quick open should be fast enough to become habit.

## 7. Accessibility Baseline

- Text contrast must be readable on the default dark theme.
- Focus state must be visible.
- Buttons and inputs need clear labels or tooltips.
- Warnings and errors must not rely on color only.
- Font size should be adjustable with app zoom.
- Disabled controls must explain why they are disabled.
- Long-running work must announce progress visibly.

## 8. Screen Acceptance Checklist

### 8.1 First launch

- [ ] Window opens at target size.
- [ ] No huge readiness text stretches the layout.
- [ ] Center empty state is calm and editor-like.
- [ ] Settings/provider setup call-to-action is visible but not noisy.
- [ ] Status bar reports no workspace and model state truthfully.

### 8.2 Workspace open

- [ ] Project tree appears quickly.
- [ ] Workspace open does not run hidden expensive work.
- [ ] Left rail active state is clear.
- [ ] Center editor remains primary.
- [ ] Right assistant is present and ready.

### 8.3 Editor

- [ ] Tabs, dirty state, save, revert, close guard work.
- [ ] Find/replace is usable.
- [ ] Breadcrumbs and outline do not crowd the editor.
- [ ] Large/truncated files are clearly read-only or guarded.
- [ ] Resize keeps editor usable.

### 8.4 Assistant

- [ ] Ask and Agent modes are clear.
- [ ] Model route and source context are visible.
- [ ] Streaming is smooth.
- [ ] Tool calls are visible and understandable.
- [ ] Approvals are clear and recoverable.
- [ ] Composer stays pinned and usable.

### 8.5 Data

- [ ] Data Sources tree feels like a database workbench.
- [ ] Query editor and result grid are distinct.
- [ ] Cancellation is visible.
- [ ] Read-only posture is explicit.
- [ ] Large results do not break layout.

### 8.6 Artifacts

- [ ] Artifact freshness and source count are visible.
- [ ] Preview, lineage, compare, archive, restore, delete, regenerate are discoverable.
- [ ] Unsupported regeneration explains why.

### 8.7 Diagnostics

- [ ] Health cards summarize state.
- [ ] Details are expandable.
- [ ] Issue report export is clear and redacted by default.
- [ ] Release trust and local toolchain problems are understandable.

## 9. UI Non-Negotiables

- Editor-first, not dashboard-first.
- Right assistant is integrated, not an afterthought.
- Left tool stripe is thin and icon-first.
- No bottom tool panel; all non-status tools live in the left sidebar or right assistant surfaces.
- Dark theme is calm and professional.
- Resize behavior is a release blocker.
- Every risky action must be visible and explainable.
