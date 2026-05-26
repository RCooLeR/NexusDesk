# Nexus Augentic Studio Fyne App

This is the new Fyne-native desktop application. The previous Wails/React implementation is preserved in `../app-wails` as the reference implementation and migration source.

## Shape

- `main.go`: desktop entrypoint only.
- `internal/app`: application lifecycle and window setup.
- `internal/domain`: framework-free domain models.
- `internal/services`: use-case services that do not depend on UI widgets.
- `internal/ui`: Fyne views, layouts, widgets, and theme code.

## Local Toolchain

Fyne desktop builds need CGO and a C compiler on Windows. Framework-independent packages can be tested now, but full app builds need a local compiler such as MSYS2 MinGW-w64 and `CGO_ENABLED=1`.

```powershell
$env:GOFLAGS='-mod=readonly'
go test ./internal/domain ./internal/services/... ./internal/ui/shell ./internal/ui/theme ./internal/brand
$env:CGO_ENABLED='1'
go build -o build\nexusdesk.exe .
```

Current known blocker: `CGO_ENABLED=1 go build .` fails on machines where no C compiler is on `PATH`; `CGO_ENABLED=0 go build .` fails because the Fyne OpenGL driver requires CGO-backed bindings.
