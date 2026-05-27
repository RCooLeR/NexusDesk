# Nexus Augentic Studio - Production Brand Kit

Built from the approved generated logo files and used as the source of truth for the desktop runtime brand.

## Includes

- `logos/png/` - production PNG logo, lockup, and app icon variants
- `design-tokens/` - CSS and JSON tokens

## Brand

- Name: Nexus Augentic Studio
- Short name: Nexus
- Tagline: Agentic work. Augmented by context.
- Colors: blue `#0D6FFE`, cyan `#13B7F1`, green `#34C759`, navy `#0D0F1A`, slate `#1E2433`, light `#F2F3F7`, white `#FFFFFF`

## Runtime Usage

The preserved Wails runtime imports approved logo and app icon assets from `app-wails/frontend/src/assets/brand/`, which mirrors this folder. The new Fyne runtime embeds selected approved assets under `nexus-app/internal/brand/assets/` for the native window icon and rail logo. Windows builds also generate `nexus-app/resource_windows.syso` from the approved app icon PNG so the compiled `.exe` carries the same brand icon in Explorer and the taskbar. Interface glyphs such as file types, route icons, chevrons, refresh, and command/search symbols are UI icons, not product logos.
