param(
    [string]$MsysRoot = $env:MSYS2_ROOT,
    [string]$OutputDir = "",
    [string]$Version = $env:NEXUSDESK_VERSION,
    [string]$Commit = "",
    [string]$BuildDate = $env:NEXUSDESK_BUILD_DATE,
    [string]$PayloadZip = "",
    [switch]$Sign,
    [string]$CertificateThumbprint = $env:NEXUSDESK_WINDOWS_CERT_THUMBPRINT,
    [string]$PfxPath = $env:NEXUSDESK_WINDOWS_PFX_PATH,
    [string]$PfxPassword = $env:NEXUSDESK_WINDOWS_PFX_PASSWORD,
    [string]$TimestampUrl = $(if ([string]::IsNullOrWhiteSpace($env:NEXUSDESK_WINDOWS_TIMESTAMP_URL)) { 'http://timestamp.digicert.com' } else { $env:NEXUSDESK_WINDOWS_TIMESTAMP_URL })
)

$ErrorActionPreference = 'Stop'

$appRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$repoRoot = (Resolve-Path (Join-Path $appRoot '..')).Path

if ([string]::IsNullOrWhiteSpace($OutputDir)) {
    $OutputDir = Join-Path $appRoot 'dist'
}
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

$safeVersion = ($Version -replace '[^0-9A-Za-z._+-]', '-')
$payloadName = "nexusdesk-windows-$safeVersion.zip"
$staging = Join-Path $OutputDir ("nexusdesk-windows-installer-" + $safeVersion)
$installerZipPath = Join-Path $OutputDir ("nexusdesk-windows-installer-" + $safeVersion + ".zip")
$manifestPath = Join-Path $OutputDir 'nexusdesk-windows-installer-manifest.json'
$sbomPath = Join-Path $OutputDir 'nexusdesk-windows-installer-sbom.json'
$provenancePath = Join-Path $OutputDir 'nexusdesk-windows-installer-provenance.json'

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

if ([string]::IsNullOrWhiteSpace($PayloadZip)) {
    $packageArgs = @{
        MsysRoot              = $MsysRoot
        OutputDir             = $OutputDir
        Version               = $Version
        Commit                = $Commit
        BuildDate             = $BuildDate
        CertificateThumbprint = $CertificateThumbprint
        PfxPath               = $PfxPath
        PfxPassword           = $PfxPassword
        TimestampUrl          = $TimestampUrl
    }
    if ($Sign) {
        $packageArgs.Sign = $true
    }
    & (Join-Path $PSScriptRoot 'package-windows-zip.ps1') @packageArgs
    if ($LASTEXITCODE -ne 0) {
        throw "package-windows-zip.ps1 failed with exit code $LASTEXITCODE."
    }
    $PayloadZip = Join-Path $OutputDir $payloadName
}

if (-not (Test-Path -LiteralPath $PayloadZip)) {
    throw "Windows payload zip was not found: $PayloadZip"
}

foreach ($path in @($staging, $installerZipPath, $manifestPath, $sbomPath, $provenancePath)) {
    if (Test-Path -LiteralPath $path) {
        Assert-UnderDirectory -Path (Resolve-Path -LiteralPath $path).Path -Root $outputRoot
        Remove-Item -LiteralPath $path -Recurse -Force
    }
}
New-Item -ItemType Directory -Force -Path $staging | Out-Null

Copy-Item -LiteralPath $PayloadZip -Destination (Join-Path $staging $payloadName) -Force

$installerScript = @'
param(
    [string]$InstallDir = (Join-Path $env:LOCALAPPDATA 'NexusDesk'),
    [switch]$NoShortcut
)

$ErrorActionPreference = 'Stop'

$payload = Join-Path $PSScriptRoot '__PAYLOAD_NAME__'
if (-not (Test-Path -LiteralPath $payload)) {
    throw "Payload archive not found next to installer: $payload"
}

$temp = Join-Path ([System.IO.Path]::GetTempPath()) ('nexusdesk-install-' + [System.Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Force -Path $temp | Out-Null
try {
    Expand-Archive -LiteralPath $payload -DestinationPath $temp -Force
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    Copy-Item -Path (Join-Path $temp '*') -Destination $InstallDir -Recurse -Force
    Copy-Item -LiteralPath (Join-Path $PSScriptRoot 'uninstall-nexusdesk.ps1') -Destination (Join-Path $InstallDir 'uninstall-nexusdesk.ps1') -Force

    if (-not $NoShortcut) {
        $programs = [Environment]::GetFolderPath('Programs')
        $shortcutPath = Join-Path $programs 'NexusDesk.lnk'
        $shell = New-Object -ComObject WScript.Shell
        $shortcut = $shell.CreateShortcut($shortcutPath)
        $shortcut.TargetPath = Join-Path $InstallDir 'nexusdesk.exe'
        $shortcut.WorkingDirectory = $InstallDir
        $shortcut.Description = 'NexusDesk'
        $shortcut.Save()
    }

    Write-Host "NexusDesk installed to $InstallDir"
    Write-Host "Run uninstall-nexusdesk.ps1 from the install directory to remove application files."
} finally {
    Remove-Item -LiteralPath $temp -Recurse -Force -ErrorAction SilentlyContinue
}
'@ -replace '__PAYLOAD_NAME__', $payloadName

$uninstallerScript = @'
param(
    [string]$InstallDir = (Join-Path $env:LOCALAPPDATA 'NexusDesk'),
    [switch]$KeepUserData
)

$ErrorActionPreference = 'Stop'

$shortcutPath = Join-Path ([Environment]::GetFolderPath('Programs')) 'NexusDesk.lnk'
Remove-Item -LiteralPath $shortcutPath -Force -ErrorAction SilentlyContinue
Remove-Item -LiteralPath $InstallDir -Recurse -Force -ErrorAction SilentlyContinue

if (-not $KeepUserData) {
    Write-Host 'Application files removed. Workspace .nexusdesk folders and OS credential-store entries are user data and are not deleted automatically.'
} else {
    Write-Host 'Application files removed. User data was left untouched.'
}
'@

$readme = @"
NexusDesk Windows installer bundle
Version: $Version
Commit: $Commit
Build date: $BuildDate

Install:
  powershell -ExecutionPolicy Bypass -File .\install-nexusdesk.ps1

Install without Start Menu shortcut:
  powershell -ExecutionPolicy Bypass -File .\install-nexusdesk.ps1 -NoShortcut

Uninstall application files:
  powershell -ExecutionPolicy Bypass -File .\uninstall-nexusdesk.ps1

The payload archive includes nexusdesk.exe plus manifest, SBOM, and provenance sidecars.
This installer bundle has its own manifest, SBOM, and provenance files next to the artifact.
"@

Set-Content -LiteralPath (Join-Path $staging 'install-nexusdesk.ps1') -Value $installerScript -Encoding UTF8
Set-Content -LiteralPath (Join-Path $staging 'uninstall-nexusdesk.ps1') -Value $uninstallerScript -Encoding UTF8
Set-Content -LiteralPath (Join-Path $staging 'README.txt') -Value $readme -Encoding UTF8

if ($Sign) {
    & (Join-Path $PSScriptRoot 'sign-windows-artifacts.ps1') -FilePath @((Join-Path $staging 'install-nexusdesk.ps1'), (Join-Path $staging 'uninstall-nexusdesk.ps1')) -CertificateThumbprint $CertificateThumbprint -PfxPath $PfxPath -PfxPassword $PfxPassword -TimestampUrl $TimestampUrl
    if ($LASTEXITCODE -ne 0) {
        throw "sign-windows-artifacts.ps1 failed with exit code $LASTEXITCODE."
    }
}

Compress-Archive -Path (Join-Path $staging '*') -DestinationPath $installerZipPath -CompressionLevel Optimal
if (-not (Test-Path -LiteralPath $installerZipPath)) {
    throw "Windows installer package was not generated: $installerZipPath"
}

Push-Location $appRoot
try {
    Invoke-Checked 'go' @('run', './cmd/release-manifest', '-artifact', $installerZipPath, '-output', $manifestPath, '-platform', 'windows-installer', '-version', $Version, '-commit', $Commit, '-build-date', $BuildDate, '-repository', 'RCooLeR/NexusDesk', '-workflow', 'scripts/package-windows-installer.ps1', '-source-commit-full', $Commit)
} finally {
    Pop-Location
}

foreach ($path in @($installerZipPath, $manifestPath, $sbomPath, $provenancePath)) {
    if (-not (Test-Path -LiteralPath $path)) {
        throw "Expected installer evidence was not generated: $path"
    }
}

$installerInfo = Get-Item -LiteralPath $installerZipPath
Write-Host "Wrote Windows installer package: $installerZipPath"
Write-Host "Installer bytes: $($installerInfo.Length)"
Write-Host "Included: install-nexusdesk.ps1, uninstall-nexusdesk.ps1, README.txt, $payloadName"
Write-Host "Installer script signing: $(if ($Sign) { 'signed before installer evidence generation' } else { 'not signed' })"
