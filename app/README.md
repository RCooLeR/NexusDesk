# NexusDesk App

This directory contains the first runnable NexusDesk desktop application.

Stack:

- Wails v2 desktop shell
- Go backend bindings
- React + TypeScript frontend
- Vite frontend build

## Local Commands

On this Windows workstation, set Node to use the system CA store before npm or Wails frontend commands:

```powershell
$env:NODE_OPTIONS='--use-system-ca --dns-result-order=ipv4first'
```

Then run:

```powershell
npm.cmd install --prefix frontend
python scripts/generate_windows_icon.py
npm.cmd run build --prefix frontend
go test ./...
wails build
```

The production binary is written to `build/bin/app.exe`.

## Development

Run live development from this directory:

```powershell
$env:NODE_OPTIONS='--use-system-ca --dns-result-order=ipv4first'
wails dev
```

The current UI is a static MVP workbench shell backed by `GetStartupState` in `app.go`. Real workspace browsing, LLM settings, and tool execution should be added through backend modules as those features land.
