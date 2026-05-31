# NexusDesk Fyne App

This is the active Fyne-native desktop application.

## Shape

- `main.go`: desktop entrypoint only.
- `internal/app`: application lifecycle and window setup.
- `internal/domain`: framework-free domain models.
- `internal/services`: use-case services that do not depend on UI widgets.
- `internal/ui`: Fyne views, layouts, widgets, and theme code.

## Local Toolchain

Fyne desktop builds need CGO and a native C compiler. On Windows, use MSYS2 UCRT64 GCC; the default expected install root is `C:\msys64`, with the compiler at `C:\msys64\ucrt64\bin\gcc.exe` or `C:\msys64\ucrt64\bin\x86_64-w64-mingw32-gcc.exe`.

The Go installation must include the standard CGO tool. Verify it before chasing MSYS2 issues:

```powershell
go tool cgo -V
Test-Path "$(go env GOTOOLDIR)\cgo.exe"
```

If the second command returns `False` or builds fail with `go: no such tool "cgo"`, repair or reinstall Go from the official Windows installer, then open a fresh PowerShell.

Install or repair the Windows compiler prerequisite before running native builds:

```powershell
winget install MSYS2.MSYS2
C:\msys64\usr\bin\bash.exe -lc "pacman -Syu --noconfirm"
C:\msys64\usr\bin\bash.exe -lc "pacman -S --needed --noconfirm mingw-w64-ucrt-x86_64-gcc mingw-w64-ucrt-x86_64-binutils mingw-w64-ucrt-x86_64-zlib"
```

Restart PowerShell after installing MSYS2. If MSYS2 is installed somewhere other than `C:\msys64`, set `MSYS2_ROOT` for the current session:

```powershell
$env:MSYS2_ROOT = 'D:\tools\msys64'
```

From this directory, use the project helper for local validation and builds:

```powershell
.\scripts\dev-env.ps1 -Test
.\scripts\dev-env.ps1 -BuildCheck
.\scripts\dev-env.ps1 -Build
.\scripts\dev-env.ps1 -Run
```

Use `-BuildCheck` for normal validation. It configures `PATH`, sets `CGO_ENABLED=1`, generates `resource_windows.syso` from the approved brand PNG, builds to a temporary folder, and removes the unsigned executable immediately. Use `-Build` only when a local runnable artifact is intentionally needed; it writes `build\nexusdesk.exe`.

For the full Windows CI/release-style checkpoint, run:

```powershell
.\scripts\ci-windows.ps1
```

That script checks formatting, runs `go test ./...`, runs `go vet ./...`, validates build metadata, builds `build\nexusdesk.exe`, generates `build\nexusdesk-windows-manifest.json`, checks whitespace, then removes generated unsigned artifacts.

If both compiler names are missing, the build stops before tests or compilation with an MSYS2 UCRT64 GCC error. `CGO_ENABLED=0 go build .` is not a valid workaround because the Fyne OpenGL driver requires CGO-backed bindings.
