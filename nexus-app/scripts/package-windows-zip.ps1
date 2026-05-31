param(
    [string]$MsysRoot = $env:MSYS2_ROOT,
    [string]$OutputDir = "",
    [string]$Version = $env:NEXUSDESK_VERSION,
    [string]$Commit = "",
    [string]$BuildDate = $env:NEXUSDESK_BUILD_DATE,
    [switch]$Sign,
    [string]$CertificateThumbprint = $env:NEXUSDESK_WINDOWS_CERT_THUMBPRINT,
    [string]$PfxPath = $env:NEXUSDESK_WINDOWS_PFX_PATH,
    [string]$PfxPassword = $env:NEXUSDESK_WINDOWS_PFX_PASSWORD,
    [string]$TimestampUrl = $(if ([string]::IsNullOrWhiteSpace($env:NEXUSDESK_WINDOWS_TIMESTAMP_URL)) { 'http://timestamp.digicert.com' } else { $env:NEXUSDESK_WINDOWS_TIMESTAMP_URL })
)

$ErrorActionPreference = 'Stop'

$appRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$repoRoot = (Resolve-Path (Join-Path $appRoot '..')).Path

if ([string]::IsNullOrWhiteSpace($MsysRoot)) {
    $MsysRoot = 'C:\msys64'
}
if ([string]::IsNullOrWhiteSpace($OutputDir)) {
    $OutputDir = Join-Path $appRoot 'dist'
}
$OutputDir = [System.IO.Path]::GetFullPath($OutputDir)
if ([string]::IsNullOrWhiteSpace($Version)) {
    $Version = '0.0.0-ci'
}
if ([string]::IsNullOrWhiteSpace($Commit)) {
    if (-not [string]::IsNullOrWhiteSpace($env:GITHUB_SHA)) {
        $Commit = $env:GITHUB_SHA.Substring(0, [Math]::Min(12, $env:GITHUB_SHA.Length))
    } else {
        $Commit = (git -C $repoRoot rev-parse --short HEAD)
    }
}
if ([string]::IsNullOrWhiteSpace($BuildDate)) {
    $BuildDate = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
}

$candidateRoots = @($MsysRoot)
if (-not [string]::IsNullOrWhiteSpace($env:MSYS2_LOCATION)) {
    $candidateRoots += $env:MSYS2_LOCATION
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
    throw "MSYS2 UCRT64 gcc.exe was not found. Install MSYS2 and mingw-w64-ucrt-x86_64-gcc, or pass -MsysRoot."
}

$env:PATH = "$ucrtBin;$usrBin;$env:PATH"
$env:CC = $gcc
$env:CGO_ENABLED = '1'
$env:GOFLAGS = '-mod=readonly'

$safeVersion = ($Version -replace '[^0-9A-Za-z._+-]', '-')
$staging = Join-Path $OutputDir ("nexusdesk-windows-" + $safeVersion)
$zipPath = Join-Path $OutputDir ("nexusdesk-windows-" + $safeVersion + ".zip")
$artifactPath = Join-Path $staging 'nexusdesk.exe'
$manifestPath = Join-Path $staging 'nexusdesk-windows-manifest.json'
$sbomPath = Join-Path $staging 'nexusdesk-windows-sbom.json'
$provenancePath = Join-Path $staging 'nexusdesk-windows-provenance.json'
$signingEvidencePath = Join-Path $staging 'nexusdesk-windows-signing.json'
$ldflags = "-linkmode=internal -X nexusdesk/internal/buildinfo.Version=$Version -X nexusdesk/internal/buildinfo.Commit=$Commit -X nexusdesk/internal/buildinfo.BuildDate=$BuildDate"

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

function Assert-UnderDirectory {
    param(
        [string]$Path,
        [string]$Root
    )

    $rootFull = [System.IO.Path]::GetFullPath($Root)
    $pathFull = [System.IO.Path]::GetFullPath($Path)
    if (-not $pathFull.StartsWith($rootFull, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing to remove path outside output directory: $pathFull"
    }
}

New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
$outputRoot = (Resolve-Path -LiteralPath $OutputDir).Path
if (Test-Path $staging) {
    Assert-UnderDirectory -Path (Resolve-Path -LiteralPath $staging).Path -Root $outputRoot
    Remove-Item -LiteralPath $staging -Recurse -Force
}
if (Test-Path $zipPath) {
    Assert-UnderDirectory -Path (Resolve-Path -LiteralPath $zipPath).Path -Root $outputRoot
    Remove-Item -LiteralPath $zipPath -Force
}
New-Item -ItemType Directory -Force -Path $staging | Out-Null

Push-Location $appRoot
try {
    Invoke-Checked 'go' @('build', '-ldflags', $ldflags, '-o', $artifactPath, '.')
    if ($Sign) {
        & (Join-Path $PSScriptRoot 'sign-windows-artifacts.ps1') -FilePath $artifactPath -CertificateThumbprint $CertificateThumbprint -PfxPath $PfxPath -PfxPassword $PfxPassword -TimestampUrl $TimestampUrl -EvidencePath $signingEvidencePath
        if ($LASTEXITCODE -ne 0) {
            throw "sign-windows-artifacts.ps1 failed with exit code $LASTEXITCODE."
        }
    }
    Invoke-Checked 'go' @('run', './cmd/release-manifest', '-artifact', $artifactPath, '-output', $manifestPath, '-platform', 'windows', '-version', $Version, '-commit', $Commit, '-build-date', $BuildDate, '-repository', 'RCooLeR/NexusDesk', '-workflow', 'scripts/package-windows-zip.ps1', '-source-commit-full', $Commit)
} finally {
    Pop-Location
}

foreach ($path in @($artifactPath, $manifestPath, $sbomPath, $provenancePath)) {
    if (-not (Test-Path $path)) {
        throw "Expected package input was not generated: $path"
    }
}
if ($Sign -and -not (Test-Path -LiteralPath $signingEvidencePath)) {
    throw "Expected Windows signing evidence was not generated: $signingEvidencePath"
}

Compress-Archive -Path (Join-Path $staging '*') -DestinationPath $zipPath -CompressionLevel Optimal
if (-not (Test-Path $zipPath)) {
    throw "Windows zip package was not generated: $zipPath"
}

$zipInfo = Get-Item $zipPath
Write-Host "Wrote Windows zip package: $zipPath"
Write-Host "Package bytes: $($zipInfo.Length)"
Write-Host "Included: nexusdesk.exe, nexusdesk-windows-manifest.json, nexusdesk-windows-sbom.json, nexusdesk-windows-provenance.json$(if ($Sign) { ', nexusdesk-windows-signing.json' } else { '' })"
Write-Host "Executable signing: $(if ($Sign) { 'signed before release evidence generation' } else { 'not signed' })"
