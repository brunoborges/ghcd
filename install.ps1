# ghx installer for Windows — downloads the latest release from GitHub
# Usage: irm https://raw.githubusercontent.com/brunoborges/ghx/main/install.ps1 | iex

$ErrorActionPreference = 'Stop'

$Repo = 'brunoborges/ghx'
$InstallDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'ghx\bin' }

# Detect architecture
$Arch = switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture) {
    'X64'   { 'amd64' }
    'Arm64' { 'arm64' }
    default { Write-Error "Unsupported architecture: $_"; exit 1 }
}

# Get latest version (skip plugin-only releases)
Write-Host 'Detecting latest version...'
$Releases = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases" -UseBasicParsing
$Version = ($Releases | Where-Object { $_.tag_name -notmatch '^plugin-' } | Select-Object -First 1).tag_name

if (-not $Version) {
    Write-Error 'Could not determine latest version'
    exit 1
}
Write-Host "Latest version: $Version"

# Download
$ZipName = "ghx-windows-$Arch.zip"
$Url = "https://github.com/$Repo/releases/download/$Version/$ZipName"
$TmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName())
New-Item -ItemType Directory -Path $TmpDir -Force | Out-Null

try {
    Write-Host "Downloading $ZipName..."
    Invoke-WebRequest -Uri $Url -OutFile (Join-Path $TmpDir $ZipName) -UseBasicParsing

    Write-Host 'Extracting...'
    Expand-Archive -Path (Join-Path $TmpDir $ZipName) -DestinationPath $TmpDir -Force

    # Install
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Copy-Item (Join-Path $TmpDir 'ghx.exe') $InstallDir -Force
    Copy-Item (Join-Path $TmpDir 'ghxd.exe') $InstallDir -Force

    # Install gh.cmd shim only if no real gh.exe is available on the system
    $ShimDst = Join-Path $InstallDir 'gh.cmd'
    $GhCmd = Get-Command gh.exe -ErrorAction SilentlyContinue

    if ($GhCmd) {
        Write-Host "Real gh.exe found on the system - skipping gh shim installation."
        Write-Host "  Use 'ghx' instead of 'gh' to benefit from caching."
    } else {
        $ShimSrc = Join-Path $TmpDir 'gh.cmd'
        if (Test-Path $ShimSrc) {
            Copy-Item $ShimSrc $InstallDir -Force
        } elseif (-not (Test-Path $ShimDst) -or (Select-String -Path $ShimDst -Pattern 'ghx-shim' -Quiet)) {
            @"
@echo off
rem ghx-shim: this script redirects gh commands through ghx for caching
ghx %*
"@ | Set-Content -Path $ShimDst -Encoding ASCII
        } else {
            Write-Host "Existing gh.cmd found at $ShimDst - not overwriting"
        }
    }

    Write-Host ''
    Write-Host "ghx  installed to $InstallDir\ghx.exe"
    Write-Host "ghxd installed to $InstallDir\ghxd.exe"
    if (-not $GhCmd) {
        Write-Host "gh   shim installed to $ShimDst (redirects to ghx)"
    }

    # Add to PATH if not already present
    $UserPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if ($UserPath -notlike "*$InstallDir*") {
        Write-Host ''
        Write-Host "Adding $InstallDir to your user PATH..."
        [Environment]::SetEnvironmentVariable('Path', "$UserPath;$InstallDir", 'User')
        $env:Path = "$env:Path;$InstallDir"
        Write-Host 'PATH updated. Restart your terminal for changes to take effect.'
    }

    Write-Host ''
    Write-Host "Run 'ghx --help' to get started, or just use 'ghx' instead of 'gh'."
}
finally {
    Remove-Item -Recurse -Force $TmpDir -ErrorAction SilentlyContinue
}
