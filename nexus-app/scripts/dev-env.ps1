param(
    [switch]$Build,
    [switch]$BuildCheck,
    [switch]$Test,
    [switch]$Run
)

$ErrorActionPreference = 'Stop'

$msysRoot = $env:MSYS2_ROOT
if ([string]::IsNullOrWhiteSpace($msysRoot)) {
    $msysRoot = 'C:\msys64'
}

$ucrtBin = Join-Path $msysRoot 'ucrt64\bin'
$usrBin = Join-Path $msysRoot 'usr\bin'
$gcc = Join-Path $ucrtBin 'gcc.exe'
$prefixedGcc = Join-Path $ucrtBin 'x86_64-w64-mingw32-gcc.exe'

if (Test-Path $gcc) {
    $env:CC = $gcc
} elseif (Test-Path $prefixedGcc) {
    $env:CC = $prefixedGcc
} else {
    throw "MSYS2 UCRT64 GCC was not found at $ucrtBin. Install MSYS2 and the mingw-w64-ucrt-x86_64-gcc package."
}

$env:PATH = "$ucrtBin;$usrBin;$env:PATH"
$env:CGO_ENABLED = '1'
if ([string]::IsNullOrWhiteSpace($env:GOFLAGS)) {
    $env:GOFLAGS = '-mod=readonly'
}
$version = if ([string]::IsNullOrWhiteSpace($env:NEXUSDESK_VERSION)) { '0.0.0-dev' } else { $env:NEXUSDESK_VERSION }
$commit = if ([string]::IsNullOrWhiteSpace($env:NEXUSDESK_COMMIT)) { (git rev-parse --short HEAD) } else { $env:NEXUSDESK_COMMIT }
$buildDate = if ([string]::IsNullOrWhiteSpace($env:NEXUSDESK_BUILD_DATE)) { (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ') } else { $env:NEXUSDESK_BUILD_DATE }
$ldflags = "-X nexusdesk/internal/buildinfo.Version=$version -X nexusdesk/internal/buildinfo.Commit=$commit -X nexusdesk/internal/buildinfo.BuildDate=$buildDate"
$goToolDir = (& go env GOTOOLDIR).Trim()
$cgoTool = Join-Path $goToolDir 'cgo.exe'
if (-not (Test-Path $cgoTool)) {
    throw "Go CGO tool was not found at $cgoTool. Repair or reinstall Go for Windows, then open a fresh PowerShell."
}

function Invoke-Checked {
    param(
        [string]$Command,
        [string[]]$Arguments
    )

    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$Command failed with exit code $LASTEXITCODE."
    }
}

Write-Host "Nexus native dev environment ready."
Write-Host "CC=$env:CC"
Write-Host "CGO_ENABLED=$env:CGO_ENABLED"
Write-Host "GOFLAGS=$env:GOFLAGS"
Write-Host "Version=$version Commit=$commit BuildDate=$buildDate"

if ($Test) {
    Invoke-Checked 'go' @('test', './internal/domain', './internal/services/...', './internal/ui/shell', './internal/ui/theme', './internal/brand')
}

if ($Build) {
    New-Item -ItemType Directory -Force -Path build | Out-Null
    if ($IsWindows -or $env:OS -eq 'Windows_NT') {
        & (Join-Path $PSScriptRoot 'build-windows-icon.ps1')
    }
    Invoke-Checked 'go' @('build', '-ldflags', $ldflags, '-o', 'build\nexusdesk.exe', '.')
}

if ($BuildCheck) {
    $checkRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("nexusdesk-build-check-" + [guid]::NewGuid().ToString("N"))
    New-Item -ItemType Directory -Force -Path $checkRoot | Out-Null
    try {
        if ($IsWindows -or $env:OS -eq 'Windows_NT') {
            & (Join-Path $PSScriptRoot 'build-windows-icon.ps1')
            $checkArtifact = Join-Path $checkRoot 'nexusdesk-build-check.exe'
        } else {
            $checkArtifact = Join-Path $checkRoot 'nexusdesk-build-check'
        }
        Invoke-Checked 'go' @('build', '-ldflags', $ldflags, '-o', $checkArtifact, '.')
        Write-Host "Build check passed; removed temporary unsigned artifact at $checkArtifact."
    } finally {
        Remove-Item -Recurse -Force -LiteralPath $checkRoot -ErrorAction SilentlyContinue
    }
}

if ($Run) {
    Invoke-Checked 'go' @('run', '-ldflags', $ldflags, '.')
}
