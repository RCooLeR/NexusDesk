param(
    [string]$MsysRoot = $env:MSYS2_ROOT,
    [switch]$SkipBuild
)

$ErrorActionPreference = 'Stop'

$appRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$repoRoot = (Resolve-Path (Join-Path $appRoot '..')).Path

if ([string]::IsNullOrWhiteSpace($MsysRoot)) {
    $MsysRoot = 'C:\msys64'
}

$candidateRoots = @()
if (-not [string]::IsNullOrWhiteSpace($MsysRoot)) {
    $candidateRoots += $MsysRoot
}
if (-not [string]::IsNullOrWhiteSpace($env:MSYS2_LOCATION)) {
    $candidateRoots += $env:MSYS2_LOCATION
}
if (-not [string]::IsNullOrWhiteSpace($env:RUNNER_TEMP)) {
    $candidateRoots += (Join-Path $env:RUNNER_TEMP 'msys64')
}
$candidateRoots += 'C:\msys64'

$gcc = $null
$ucrtBin = $null
$usrBin = $null
foreach ($root in ($candidateRoots | Where-Object { -not [string]::IsNullOrWhiteSpace($_) } | Select-Object -Unique)) {
    $candidateUcrtBin = Join-Path $root 'ucrt64\bin'
    $candidateGcc = Join-Path $candidateUcrtBin 'gcc.exe'
    $candidatePrefixedGcc = Join-Path $candidateUcrtBin 'x86_64-w64-mingw32-gcc.exe'
    if (Test-Path $candidateGcc) {
        $gcc = $candidateGcc
        $ucrtBin = $candidateUcrtBin
        $usrBin = Join-Path $root 'usr\bin'
        break
    }
    if (Test-Path $candidatePrefixedGcc) {
        $gcc = $candidatePrefixedGcc
        $ucrtBin = $candidateUcrtBin
        $usrBin = Join-Path $root 'usr\bin'
        break
    }
}
if ($null -eq $gcc) {
    $pathGcc = Get-Command gcc.exe -ErrorAction SilentlyContinue
    if ($pathGcc) {
        $gcc = $pathGcc.Source
        $ucrtBin = Split-Path -Parent $gcc
        $root = Split-Path -Parent (Split-Path -Parent $ucrtBin)
        $usrBin = Join-Path $root 'usr\bin'
    }
}

if ($null -eq $gcc) {
    $searched = ($candidateRoots | Where-Object { -not [string]::IsNullOrWhiteSpace($_) } | Select-Object -Unique) -join ', '
    throw "MSYS2 UCRT64 gcc.exe was not found. Searched roots: $searched. In GitHub Actions, pass MSYS2_ROOT from msys2/setup-msys2's msys2-location output; locally, install MSYS2 plus mingw-w64-ucrt-x86_64-gcc, mingw-w64-ucrt-x86_64-binutils, and mingw-w64-ucrt-x86_64-zlib."
}

$env:PATH = "$ucrtBin;$usrBin;$env:PATH"
$env:CC = $gcc
$env:CGO_ENABLED = '1'
$env:GOFLAGS = '-mod=readonly'
$version = if ([string]::IsNullOrWhiteSpace($env:NEXUSDESK_VERSION)) { '0.0.0-ci' } else { $env:NEXUSDESK_VERSION }
$commit = if ([string]::IsNullOrWhiteSpace($env:GITHUB_SHA)) { (git -C $repoRoot rev-parse --short HEAD) } else { $env:GITHUB_SHA.Substring(0, [Math]::Min(12, $env:GITHUB_SHA.Length)) }
$buildDate = if ([string]::IsNullOrWhiteSpace($env:NEXUSDESK_BUILD_DATE)) { (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ') } else { $env:NEXUSDESK_BUILD_DATE }
$ldflags = "-X nexusdesk/internal/buildinfo.Version=$version -X nexusdesk/internal/buildinfo.Commit=$commit -X nexusdesk/internal/buildinfo.BuildDate=$buildDate"
$buildLdflags = "-linkmode=internal $ldflags"
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

Push-Location $appRoot
try {
    Write-Host 'Checking gofmt...'
    $goFiles = @(git ls-files '*.go')
    if ($goFiles.Count -gt 0) {
        $unformatted = @(gofmt -l @goFiles)
        if ($unformatted.Count -gt 0) {
            $unformatted | ForEach-Object { Write-Error "gofmt required: $_" }
            throw 'gofmt check failed.'
        }
    }

    Write-Host 'Running tests...'
    Invoke-Checked 'go' @('test', './...')

    Write-Host 'Running static analysis...'
    Invoke-Checked 'go' @('vet', './...')

    Write-Host 'Validating build metadata...'
    Invoke-Checked 'go' @('test', '-ldflags', $ldflags, './internal/buildinfo')

    if (-not $SkipBuild) {
        Write-Host 'Building native Windows executable...'
        New-Item -ItemType Directory -Force -Path build | Out-Null
        $artifactPath = Join-Path $appRoot 'build\nexusdesk.exe'
        $manifestPath = Join-Path $appRoot 'build\nexusdesk-windows-manifest.json'
        $sbomPath = Join-Path $appRoot 'build\nexusdesk-windows-sbom.json'
        $provenancePath = Join-Path $appRoot 'build\nexusdesk-windows-provenance.json'
        Invoke-Checked 'go' @('build', '-ldflags', $buildLdflags, '-o', $artifactPath, '.')

        Write-Host 'Generating release evidence...'
        Invoke-Checked 'go' @('run', './cmd/release-manifest', '-artifact', $artifactPath, '-output', $manifestPath, '-platform', 'windows', '-version', $version, '-commit', $commit, '-build-date', $buildDate, '-repository', 'RCooLeR/NexusDesk', '-workflow', 'scripts/ci-windows.ps1', '-source-commit-full', $commit)
        if (-not (Test-Path $manifestPath)) {
            throw 'Release manifest was not generated.'
        }
        if (-not (Test-Path $sbomPath)) {
            throw 'Release SBOM was not generated.'
        }
        if (-not (Test-Path $provenancePath)) {
            throw 'Release provenance was not generated.'
        }

        Write-Host 'Running Windows installer smoke...'
        $smokeOutputDir = Join-Path $appRoot 'build\installer-smoke'
        & (Join-Path $PSScriptRoot 'smoke-windows-installer.ps1') -MsysRoot $MsysRoot -OutputDir $smokeOutputDir -Version $version -Commit $commit -BuildDate $buildDate
        if ($LASTEXITCODE -ne 0) {
            throw "smoke-windows-installer.ps1 failed with exit code $LASTEXITCODE."
        }
    }

    Write-Host 'Checking diff whitespace...'
    Invoke-Checked 'git' @('-C', $repoRoot, 'diff', '--check')
} finally {
    Remove-Item -LiteralPath (Join-Path $appRoot 'nexusdesk.exe') -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath (Join-Path $appRoot 'build\nexusdesk.exe') -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath (Join-Path $appRoot 'build\nexusdesk-windows-manifest.json') -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath (Join-Path $appRoot 'build\nexusdesk-windows-sbom.json') -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath (Join-Path $appRoot 'build\nexusdesk-windows-provenance.json') -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath (Join-Path $appRoot 'build\installer-smoke') -Recurse -Force -ErrorAction SilentlyContinue
    Pop-Location
}
