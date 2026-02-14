# Windows Setup Script Design

**Date:** 2026-02-14
**Target audience:** End users who want to build from source on Windows

## Files Created

- `scripts/setup-windows.ps1` - PowerShell automation script
- `docs/WINDOWS_SETUP.md` - Manual fallback instructions

## What the Script Does

1. Checks if running as Administrator (needed for some installs)
2. Detects existing Go 1.23+, Node 20+, and WebView2
3. Installs missing prerequisites using winget or direct download
4. Installs Wails CLI via `go install`
5. Runs `wails build` to create the executable
6. Reports success with location of built binary

## Design Principles

- **Idempotent:** Safe to run multiple times. Skip already-installed tools.
- **Graceful fallbacks:** If winget fails, use direct downloads.
- **Clear errors:** Tell users what failed and link to manual instructions.

## Prerequisites

| Tool | Minimum Version | Install Method |
|------|-----------------|----------------|
| Go | 1.23 | winget → fallback to go.dev/dl |
| Node.js | 20.x | winget → fallback to nodejs.org |
| WebView2 | Latest | winget → fallback to Microsoft download |

## Build Process

1. Refresh environment variables from registry
2. Install Wails: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
3. Run `wails doctor` for diagnostics
4. Build: `wails build -platform windows/amd64 -clean`
5. Output: `build/bin/wails-base-fresh.exe`

## Manual Instructions Structure

1. Prerequisites with download links
2. Step-by-step installation for each tool
3. Build commands
4. Troubleshooting section (PATH issues, WebView2, wails doctor)
