# NexusDesk Fyne App

This is the active Fyne-native desktop application.

## Shape

- `main.go`: desktop entrypoint only.
- `internal/app`: application lifecycle and window setup.
- `internal/domain`: framework-free domain models.
- `internal/services`: use-case services that do not depend on UI widgets.
- `internal/ui`: Fyne views, layouts, widgets, and theme code.

## Local Toolchain

Fyne desktop builds need CGO and a C compiler on Windows. This workstation uses MSYS2 UCRT64 GCC from `C:\msys64\ucrt64\bin`. The helper below configures the current PowerShell session and can run tests, builds, or the app.

```powershell
.\scripts\dev-env.ps1 -Test
.\scripts\dev-env.ps1 -BuildCheck
.\scripts\dev-env.ps1 -Build
.\scripts\dev-env.ps1 -Run
```

Use `-BuildCheck` for normal validation. It builds to a temporary folder and removes the unsigned executable immediately, which avoids leaving fresh low-reputation dev binaries in the source tree. Use `-Build` only when you intentionally need a local runnable artifact. On Windows, `-Build` and `-BuildCheck` generate `resource_windows.syso` from the approved brand PNG before `go build`, so built executables carry the app icon in Explorer and the taskbar.

If your machine uses another MSYS2 location, set `MSYS2_ROOT` before invoking the helper. `CGO_ENABLED=0 go build .` still fails because the Fyne OpenGL driver requires CGO-backed bindings.
