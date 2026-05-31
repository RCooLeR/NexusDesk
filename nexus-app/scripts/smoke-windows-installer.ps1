param(
    [string]$MsysRoot = $env:MSYS2_ROOT,
    [string]$OutputDir = "",
    [string]$Version = $env:NEXUSDESK_VERSION,
    [string]$Commit = "",
    [string]$BuildDate = $env:NEXUSDESK_BUILD_DATE
)

$ErrorActionPreference = 'Stop'

$appRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$repoRoot = (Resolve-Path (Join-Path $appRoot '..')).Path

if ([string]::IsNullOrWhiteSpace($OutputDir)) {
    $OutputDir = Join-Path $appRoot 'dist-smoke'
}
$OutputDir = [System.IO.Path]::GetFullPath($OutputDir)
if ([string]::IsNullOrWhiteSpace($Version)) {
    $Version = '0.0.0-smoke'
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
$installerZipPath = Join-Path $OutputDir ("nexusdesk-windows-installer-" + $safeVersion + ".zip")
$smokeRoot = Join-Path $OutputDir ("smoke-windows-installer-" + $safeVersion)
$extractDir = Join-Path $smokeRoot 'installer'
$installDir = Join-Path $smokeRoot 'install'
$workspaceDir = Join-Path $smokeRoot 'workspace'
$workspaceData = Join-Path $workspaceDir '.nexusdesk\keep.txt'

function Assert-UnderDirectory {
    param(
        [string]$Path,
        [string]$Root
    )

    $rootFull = [System.IO.Path]::GetFullPath($Root)
    $pathFull = [System.IO.Path]::GetFullPath($Path)
    if (-not $pathFull.StartsWith($rootFull, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing to remove path outside smoke directory: $pathFull"
    }
}

New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
$outputRoot = (Resolve-Path -LiteralPath $OutputDir).Path
if (Test-Path -LiteralPath $smokeRoot) {
    Assert-UnderDirectory -Path (Resolve-Path -LiteralPath $smokeRoot).Path -Root $outputRoot
    Remove-Item -LiteralPath $smokeRoot -Recurse -Force
}

& (Join-Path $PSScriptRoot 'package-windows-installer.ps1') -MsysRoot $MsysRoot -OutputDir $OutputDir -Version $Version -Commit $Commit -BuildDate $BuildDate
if ($LASTEXITCODE -ne 0) {
    throw "package-windows-installer.ps1 failed with exit code $LASTEXITCODE."
}
if (-not (Test-Path -LiteralPath $installerZipPath)) {
    throw "Installer artifact was not generated: $installerZipPath"
}

New-Item -ItemType Directory -Force -Path $extractDir | Out-Null
Expand-Archive -LiteralPath $installerZipPath -DestinationPath $extractDir -Force

New-Item -ItemType Directory -Force -Path (Split-Path -Parent $workspaceData) | Out-Null
Set-Content -LiteralPath $workspaceData -Value 'workspace data must survive uninstall' -Encoding UTF8

& powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $extractDir 'install-nexusdesk.ps1') -InstallDir $installDir -NoShortcut
if ($LASTEXITCODE -ne 0) {
    throw "install-nexusdesk.ps1 failed with exit code $LASTEXITCODE."
}

foreach ($name in @('nexusdesk.exe', 'nexusdesk-windows-manifest.json', 'nexusdesk-windows-sbom.json', 'nexusdesk-windows-provenance.json', 'uninstall-nexusdesk.ps1')) {
    $path = Join-Path $installDir $name
    if (-not (Test-Path -LiteralPath $path)) {
        throw "Installed file missing: $path"
    }
}

$installedExe = Join-Path $installDir 'nexusdesk.exe'
$versionOutput = & $installedExe --version
if ($LASTEXITCODE -ne 0) {
    throw "Installed nexusdesk.exe --version failed with exit code $LASTEXITCODE."
}
if (($versionOutput | Out-String) -notmatch [regex]::Escape($Version)) {
    throw "Installed nexusdesk.exe --version did not include expected version $Version. Output: $($versionOutput | Out-String)"
}

$smokeOutput = & $installedExe --smoke-check $workspaceDir
if ($LASTEXITCODE -ne 0) {
    throw "Installed nexusdesk.exe --smoke-check failed with exit code $LASTEXITCODE."
}
$smokeReport = $smokeOutput | Out-String | ConvertFrom-Json
$requiredSmokeChecks = @(
    'workspace-open',
    'file-preview',
    'workspace-search',
    'edit-save-revert',
    'assistant-settings',
    'dataset-profile',
    'artifact-write-read',
    'diagnostics-export'
)
foreach ($checkName in $requiredSmokeChecks) {
    $match = @($smokeReport.checks | Where-Object { $_.name -eq $checkName -and $_.status -eq 'ok' })
    if ($match.Count -ne 1) {
        throw "Installed app smoke check missing or failed: $checkName"
    }
}

& powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $extractDir 'uninstall-nexusdesk.ps1') -InstallDir $installDir
if ($LASTEXITCODE -ne 0) {
    throw "uninstall-nexusdesk.ps1 failed with exit code $LASTEXITCODE."
}
if (Test-Path -LiteralPath $installDir) {
    throw "Install directory still exists after uninstall: $installDir"
}
if (-not (Test-Path -LiteralPath $workspaceData)) {
    throw "Workspace .nexusdesk data was removed during uninstall smoke."
}

Write-Host "Windows installer smoke passed."
Write-Host "Installer artifact: $installerZipPath"
Write-Host "Installed app smoke checks: $($requiredSmokeChecks -join ', ')"
Write-Host "Install directory removed: $installDir"
Write-Host "Workspace data preserved: $workspaceData"
