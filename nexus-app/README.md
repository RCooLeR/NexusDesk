# Nexus Augentic Studio Fyne App

This is the new Fyne-native desktop application. The previous Wails/React implementation is preserved in `../app-wails` as the reference implementation and migration source.

## Shape

- `main.go`: desktop entrypoint only.
- `internal/app`: application lifecycle and window setup.
- `internal/domain`: framework-free domain models.
- `internal/services`: use-case services that do not depend on UI widgets.
- `internal/ui`: Fyne views, layouts, widgets, and theme code.

## Local Toolchain

Fyne desktop builds need CGO and a C compiler on Windows. The current Codex shell has `CGO_ENABLED=0`, so framework-independent packages can be tested now, while full app builds need a local compiler such as MSYS2 MinGW-w64 and `CGO_ENABLED=1`.

```powershell
go test ./internal/domain ./internal/services/workspace ./internal/ui/shell ./internal/ui/theme
$env:CGO_ENABLED='1'
go run .
```
