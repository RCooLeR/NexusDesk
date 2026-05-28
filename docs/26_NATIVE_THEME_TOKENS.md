# NexusDesk Native Theme Tokens

This document defines the current Fyne-native visual token baseline for the JetBrains-like NexusDesk workbench. The active implementation lives in `nexus-app/internal/ui/theme` and must remain UI-only; services and domain packages must not import Fyne or theme code.

## Direction

NexusDesk should feel like a calm professional IDE/workbench, not a browser dashboard. The default visual direction is:

- dark, layered workspace surfaces;
- compact density suitable for code, data grids, artifacts, and diagnostics;
- restrained blue accent for active states and primary actions;
- clear semantic status colors for success, warning, and error states;
- visible but not noisy focus, selection, border, and shadow treatment.

## Core Palette

| Token | Intended Use |
| --- | --- |
| `Background` | Main window and lowest-level shell background. |
| `Panel` | Tool windows, menus, grouped bottom panels, and secondary surfaces. |
| `PanelRaised` | Dialog-like overlays, elevated tool surfaces, and popover bodies. |
| `Editor` | Code/editor surfaces and long-form reading panes. |
| `Border` | Separators, tool-window boundaries, and subtle panel outlines. |
| `Shadow` | Elevated surface shadow where Fyne exposes shadow color. |
| `TextPrimary` | Main readable text. |
| `TextSecondary` | Metadata, secondary labels, row details, and helper text. |
| `TextMuted` | Placeholders, disabled text, and low-emphasis metadata. |
| `Accent` | Primary actions, active tab indicators, selected route/model emphasis. |
| `AccentHover` | Hovered active/primary affordances. |
| `AccentPressed` | Pressed active/primary affordances. |
| `AccentForeground` | Text/icons on accent fills. |
| `Selection` | Selected rows, selected tabs, selected project tree nodes. |
| `Focus` | Keyboard focus ring and active input affordance. |
| `Input` | Text fields, search fields, query editors, and compact forms. |
| `InputBorder` | Text-field borders and form focus boundaries. |
| `Button` | Neutral button surfaces. |
| `ButtonDisabled` | Disabled button surface. |
| `Success` | Completed jobs, healthy diagnostics, safe confirmations. |
| `Warning` | Recoverable risk, over-budget timings, missing optional runtime state. |
| `Error` | Failed jobs, blocked actions, invalid settings, unsafe operation refusal. |

## Fyne Mapping

`NexusTheme` maps the Fyne theme names to the token palette so existing widgets inherit the baseline without ad hoc styling:

- `ColorNameBackground` -> `Background`
- `ColorNameForeground` -> `TextPrimary`
- `ColorNamePrimary` -> `Accent`
- `ColorNameForegroundOnPrimary` -> `AccentForeground`
- `ColorNameSelection` -> `Selection`
- `ColorNameFocus` -> `Focus`
- `ColorNameInputBackground` -> `Input`
- `ColorNameInputBorder` -> `InputBorder`
- `ColorNameButton` -> `Button`
- `ColorNameDisabledButton` -> `ButtonDisabled`
- `ColorNameDisabled` and `ColorNamePlaceHolder` -> `TextMuted`
- `ColorNameHeaderBackground` and `ColorNameMenuBackground` -> `Panel`
- `ColorNameOverlayBackground` -> `PanelRaised`
- `ColorNameSeparator` -> `Border`
- `ColorNameShadow` -> `Shadow`
- `ColorNameSuccess`, `ColorNameWarning`, and `ColorNameError` -> semantic status tokens

## Density

The current compact baseline uses:

- outer padding: `8`;
- inner padding: `6`.

Future comfortable-density work should add an explicit mode instead of changing these defaults silently.

## Rules For Future UI Work

- Prefer Fyne theme names or exported palette tokens instead of hardcoded local colors.
- Keep semantic status colors consistent across Jobs, Diagnostics, Problems, Settings, Agent Audit, and Artifacts.
- Use `Panel`/`PanelRaised`/`Editor` to create hierarchy before adding brighter accents.
- Use accent color sparingly for active or primary affordances, not decoration.
- Do not import `internal/ui/theme` from services, domain, or persistence packages.
