<#
.SYNOPSIS
    Setup script for building minecraft-tunnel on Windows
.DESCRIPTION
    Installs Go, Node.js, WebView2, and Wails CLI, then builds the application.
    Safe to run multiple times - skips already-installed tools.
.EXAMPLE
    .\setup-windows.ps1
#>

param(
    [switch]$SkipBuild,
    [switch]$Force
)

$ErrorActionPreference = "Stop"

$GoVersion = "1.23"
$NodeVersion = "20"

function Write-Step {
    param([string]$Message)
    Write-Host "`n==> $Message" -ForegroundColor Cyan
}

function Write-Success {
    param([string]$Message)
    Write-Host "    OK: $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "    WARNING: $Message" -ForegroundColor Yellow
}

function Test-Command {
    param([string]$Command)
    $null -ne (Get-Command $Command -ErrorAction SilentlyContinue)
}

function Get-GoVersion {
    if (Test-Command "go") {
        $version = (go version) -replace ".*go(\d+\.\d+).*", '$1'
        return [version]$version
    }
    return $null
}

function Get-NodeVersion {
    if (Test-Command "node") {
        $version = (node --version) -replace "v(\d+)\..*", '$1'
        return [int]$version
    }
    return $null
}

function Install-ViaWinget {
    param([string]$PackageId)
    if (Test-Command "winget") {
        winget install $PackageId --accept-package-agreements --accept-source-agreements --silent
        return $LASTEXITCODE -eq 0
    }
    return $false
}

function Install-Go {
    Write-Step "Checking Go $GoVersion+"
    $currentVersion = Get-GoVersion
    $requiredVersion = [version]$GoVersion

    if ($currentVersion -and $currentVersion -ge $requiredVersion) {
        Write-Success "Go $currentVersion already installed"
        return $true
    }

    Write-Host "    Installing Go $GoVersion..."
    
    if (Install-ViaWinget "GoLang.Go") {
        Write-Success "Go installed via winget"
        return $true
    }

    Write-Host "    Winget not available, downloading from go.dev..."
    $goUrl = "https://go.dev/dl/go1.23.4.windows-amd64.msi"
    $goInstaller = "$env:TEMP\go-installer.msi"
    
    try {
        Invoke-WebRequest -Uri $goUrl -OutFile $goInstaller -UseBasicParsing
        Start-Process msiexec.exe -ArgumentList "/i `"$goInstaller`" /quiet /norestart" -Wait
        Remove-Item $goInstaller -Force -ErrorAction SilentlyContinue
        Write-Success "Go installed from go.dev"
        return $true
    }
    catch {
        Write-Warning "Failed to install Go: $_"
        return $false
    }
}

function Install-Node {
    Write-Step "Checking Node.js $NodeVersion+"
    $currentVersion = Get-NodeVersion

    if ($currentVersion -and $currentVersion -ge [int]$NodeVersion) {
        Write-Success "Node.js $currentVersion already installed"
        return $true
    }

    Write-Host "    Installing Node.js $NodeVersion LTS..."
    
    if (Install-ViaWinget "OpenJS.NodeJS.LTS") {
        Write-Success "Node.js installed via winget"
        return $true
    }

    Write-Host "    Winget not available, downloading from nodejs.org..."
    $nodeUrl = "https://nodejs.org/dist/v20.18.1/node-v20.18.1-x64.msi"
    $nodeInstaller = "$env:TEMP\node-installer.msi"
    
    try {
        Invoke-WebRequest -Uri $nodeUrl -OutFile $nodeInstaller -UseBasicParsing
        Start-Process msiexec.exe -ArgumentList "/i `"$nodeInstaller`" /quiet /norestart" -Wait
        Remove-Item $nodeInstaller -Force -ErrorAction SilentlyContinue
        Write-Success "Node.js installed from nodejs.org"
        return $true
    }
    catch {
        Write-Warning "Failed to install Node.js: $_"
        return $false
    }
}

function Install-WebView2 {
    Write-Step "Checking WebView2 Runtime"
    
    $webview2Path = "${env:ProgramFiles(x86)}\Microsoft\EdgeWebView\Application"
    if (Test-Path $webview2Path) {
        Write-Success "WebView2 already installed"
        return $true
    }

    Write-Host "    Installing WebView2 Runtime..."
    
    if (Install-ViaWinget "Microsoft.EdgeWebView2Runtime") {
        Write-Success "WebView2 installed via winget"
        return $true
    }

    Write-Host "    Winget not available, downloading from Microsoft..."
    $webview2Url = "https://go.microsoft.com/fwlink/p/?LinkId=2124703"
    $webview2Installer = "$env:TEMP\webview2-installer.exe"
    
    try {
        Invoke-WebRequest -Uri $webview2Url -OutFile $webview2Installer -UseBasicParsing
        Start-Process $webview2Installer -ArgumentList "/silent", "/install" -Wait
        Remove-Item $webview2Installer -Force -ErrorAction SilentlyContinue
        Write-Success "WebView2 installed from Microsoft"
        return $true
    }
    catch {
        Write-Warning "Failed to install WebView2: $_"
        return $false
    }
}

function Refresh-Path {
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")
    
    $goBin = "$env:USERPROFILE\go\bin"
    if (Test-Path $goBin -and $env:Path -notlike "*$goBin*") {
        $env:Path += ";$goBin"
    }
}

function Install-Wails {
    Write-Step "Checking Wails CLI"
    
    Refresh-Path
    
    if ((Test-Command "wails") -and -not $Force) {
        Write-Success "Wails already installed"
        return $true
    }

    Write-Host "    Installing Wails CLI..."
    
    if (-not (Test-Command "go")) {
        Write-Warning "Go not found in PATH. You may need to restart your terminal."
        return $false
    }

    & go install github.com/wailsapp/wails/v2/cmd/wails@latest
    
    if ($LASTEXITCODE -eq 0) {
        Refresh-Path
        Write-Success "Wails installed"
        return $true
    }
    else {
        Write-Warning "Failed to install Wails"
        return $false
    }
}

function Invoke-Build {
    Write-Step "Running Wails doctor"
    
    if (Test-Command "wails") {
        & wails doctor
    }
    else {
        Write-Warning "Wails not found, skipping doctor"
    }

    Write-Step "Building application"
    
    if (-not (Test-Command "wails")) {
        Write-Host "ERROR: Wails not found. Please restart your terminal and try again." -ForegroundColor Red
        Write-Host "       Or see docs/WINDOWS_SETUP.md for manual instructions." -ForegroundColor Red
        return $false
    }

    & wails build -platform windows/amd64 -clean

    if ($LASTEXITCODE -eq 0) {
        $exePath = "build\bin\minecraft-tunnel.exe"
        if (Test-Path $exePath) {
            Write-Host "`n========================================" -ForegroundColor Green
            Write-Host "BUILD SUCCESSFUL!" -ForegroundColor Green
            Write-Host "Executable: $exePath" -ForegroundColor Green
            Write-Host "========================================" -ForegroundColor Green
            
            $openFolder = Read-Host "Open output folder? (Y/n)"
            if ($openFolder -ne "n" -and $openFolder -ne "N") {
                explorer.exe "build\bin"
            }
            return $true
        }
    }
    
    Write-Host "ERROR: Build failed. See docs/WINDOWS_SETUP.md for troubleshooting." -ForegroundColor Red
    return $false
}

# Main execution
Write-Host @"

  __  __ _       _ _   _ _____ _____               _             _ 
 |  \/  (_)     (_) | | /  __ \_   _|             | |           | |
 | .  . |_ _ __  _| | |_| /  \/ | |    _ __   __ _| | _____  ___| |
 | |\/| | | '_ \| | | __| |     | |   | '_ \ / _` | |/ / _ \/ __| |
 | |  | | | | | | | | |_| \__/\_| |_  | | | | (_| |   <  __/\__ \_|
 |_|  |_|_|_| |_|_|_|\__|\____/_____| |_| |_|\__,_|_|\_\___||___(_) Setup Script

"@ -ForegroundColor Cyan

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $scriptDir

$success = $true

if (-not (Install-Go)) { $success = $false }
if (-not (Install-Node)) { $success = $false }
if (-not (Install-WebView2)) { $success = $false }
if (-not (Install-Wails)) { $success = $false }

if (-not $SkipBuild -and $success) {
    if (-not (Invoke-Build)) { $success = $false }
}
elseif ($SkipBuild) {
    Write-Host "`nSkipped build (--SkipBuild specified)" -ForegroundColor Yellow
    Write-Host "Run 'wails build -platform windows/amd64 -clean' to build manually"
}

if (-not $success) {
    Write-Host "`nSetup encountered issues. See docs/WINDOWS_SETUP.md for manual instructions." -ForegroundColor Red
    exit 1
}
