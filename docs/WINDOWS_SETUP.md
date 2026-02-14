# Windows Setup Guide

This guide walks you through building minecraft-tunnel from source on Windows.

## Prerequisites

| Tool | Minimum Version | Download |
|------|-----------------|----------|
| Go | 1.23 | [go.dev/dl](https://go.dev/dl/) |
| Node.js | 20.x LTS | [nodejs.org](https://nodejs.org/) |
| WebView2 | Latest | [Microsoft](https://developer.microsoft.com/en-us/microsoft-edge/webview2/) |

## Automated Setup

Run the PowerShell script from the project root:

```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
.\scripts\setup-windows.ps1
```

Options:
- `-SkipBuild` - Install tools only, don't build
- `-Force` - Reinstall Wails even if already present

## Manual Setup

### 1. Install Go 1.23+

1. Download from https://go.dev/dl/
2. Run the MSI installer (e.g., `go1.23.4.windows-amd64.msi`)
3. Verify installation:
   ```powershell
   go version
   ```
   Should show `go1.23.x` or higher.

### 2. Install Node.js 20.x LTS

1. Download from https://nodejs.org/
2. Run the MSI installer
3. Verify installation:
   ```powershell
   node --version
   npm --version
   ```

### 3. Install WebView2 Runtime

Windows 11 includes WebView2 by default. For Windows 10:

1. Download from https://developer.microsoft.com/en-us/microsoft-edge/webview2/
2. Run the installer (Evergreen Bootstrapper)

### 4. Add Go bin to PATH

The Wails CLI installs to `%USERPROFILE%\go\bin`. Add it to your PATH:

1. Open System Properties → Advanced → Environment Variables
2. Under "User variables", find `Path` and click Edit
3. Add: `%USERPROFILE%\go\bin`
4. Click OK and restart your terminal

Or run this in PowerShell:
```powershell
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";$env:USERPROFILE\go\bin", "User")
```

### 5. Install Wails CLI

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

Verify:
```powershell
wails version
```

### 6. Check Dependencies

```powershell
wails doctor
```

This checks for missing dependencies and provides guidance.

### 7. Build the Application

From the project root:

```powershell
wails build -platform windows/amd64 -clean
```

The executable will be at: `build\bin\minecraft-tunnel.exe`

## Troubleshooting

### `go: command not found`

Go isn't in your PATH. Either:
- Restart your terminal/PowerShell
- Re-run the Go installer
- Manually add `C:\Program Files\Go\bin` to PATH

### `wails: command not found`

The Go bin directory isn't in PATH:
```powershell
$env:Path += ";$env:USERPROFILE\go\bin"
```

Or add it permanently (see step 4 above).

### Build fails with WebView2 errors

Install WebView2 manually:
- Download: https://go.microsoft.com/fwlink/p/?LinkId=2124703
- Run the installer

### `npm install` fails in frontend

Clear npm cache and try again:
```powershell
cd frontend
npm cache clean --force
Remove-Item node_modules -Recurse -Force -ErrorAction SilentlyContinue
npm install
```

### Other issues

Run diagnostics:
```powershell
wails doctor
```

Check the [Wails documentation](https://wails.io/docs/guides/windows/) for Windows-specific guidance.

## Development Mode

For hot-reload during development:

```powershell
wails dev
```

The frontend runs at http://localhost:5173 with the Go backend.
