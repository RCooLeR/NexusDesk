param(
    [string[]]$CleanMachineReport = @(),
    [string[]]$BetaInstallReport = @(),
    [switch]$RequireAllCleanMachinePlatforms,
    [switch]$RequireFiveBetaReports
)

$ErrorActionPreference = 'Stop'

function Read-ReportText {
    param([string]$Path)

    if (-not (Test-Path -LiteralPath $Path)) {
        throw "Validation report was not found: $Path"
    }
    return Get-Content -LiteralPath $Path -Raw
}

function Get-BulletValue {
    param(
        [string]$Text,
        [string]$Label
    )

    $pattern = "(?m)^-\s+$([regex]::Escape($Label))\s*:\s*(.*)$"
    $match = [regex]::Match($Text, $pattern)
    if (-not $match.Success) {
        throw "Missing report field: $Label"
    }
    return $match.Groups[1].Value.Trim()
}

function Assert-NonPlaceholderValue {
    param(
        [string]$Text,
        [string]$Label
    )

    $value = Get-BulletValue -Text $Text -Label $Label
    if ([string]::IsNullOrWhiteSpace($value)) {
        throw "Report field is empty: $Label"
    }
    if ($value -match '\b(pass|fail|yes|no|not checked|unsure)\s*/\s*') {
        throw "Report field still contains an unselected template choice: $Label = $value"
    }
    return $value
}

function Assert-ExactChoice {
    param(
        [string]$Text,
        [string]$Label,
        [string]$Expected
    )

    $value = Assert-NonPlaceholderValue -Text $Text -Label $Label
    if ($value.ToLowerInvariant() -ne $Expected.ToLowerInvariant()) {
        throw "Report field '$Label' must be '$Expected'; got '$value'."
    }
    return $value
}

function Assert-Sha256Value {
    param(
        [string]$Text,
        [string]$Label
    )

    $value = Assert-NonPlaceholderValue -Text $Text -Label $Label
    if ($value -notmatch '^[0-9a-fA-F]{64}$') {
        throw "Report field '$Label' must be a 64-character SHA-256 value; got '$value'."
    }
    return $value.ToLowerInvariant()
}

function Get-MarkdownTableRows {
    param(
        [string]$Text,
        [string]$FirstColumnHeader
    )

    $lines = $Text -split "\r?\n"
    for ($i = 0; $i -lt $lines.Count; $i++) {
        $line = $lines[$i].Trim()
        if (-not ($line.StartsWith('|') -and $line.EndsWith('|'))) {
            continue
        }
        $headers = @($line.Trim('|') -split '\|' | ForEach-Object { $_.Trim() })
        if ($headers.Count -eq 0 -or $headers[0] -ne $FirstColumnHeader) {
            continue
        }

        $rows = @()
        for ($j = $i + 2; $j -lt $lines.Count; $j++) {
            $rowLine = $lines[$j].Trim()
            if (-not ($rowLine.StartsWith('|') -and $rowLine.EndsWith('|'))) {
                break
            }
            $cells = @($rowLine.Trim('|') -split '\|' | ForEach-Object { $_.Trim() })
            if ($cells.Count -lt $headers.Count) {
                throw "Malformed table row under '$FirstColumnHeader': $rowLine"
            }
            $row = [ordered]@{}
            for ($k = 0; $k -lt $headers.Count; $k++) {
                $row[$headers[$k]] = $cells[$k]
            }
            $rows += [pscustomobject]$row
        }
        return $rows
    }

    throw "Missing Markdown table with first column '$FirstColumnHeader'."
}

function Assert-TableRowsPass {
    param(
        [string]$Text,
        [string]$FirstColumnHeader,
        [string]$ResultHeader,
        [string[]]$RequiredRows
    )

    $rows = Get-MarkdownTableRows -Text $Text -FirstColumnHeader $FirstColumnHeader
    foreach ($required in $RequiredRows) {
        $row = @($rows | Where-Object { $_.$FirstColumnHeader -eq $required } | Select-Object -First 1)
        if ($row.Count -eq 0) {
            throw "Missing required table row '$required' in '$FirstColumnHeader' table."
        }
        $result = "$($row[0].$ResultHeader)".Trim()
        if ([string]::IsNullOrWhiteSpace($result)) {
            throw "Required table row '$required' has no result."
        }
        if (-not $result.ToLowerInvariant().StartsWith('pass')) {
            throw "Required table row '$required' must pass; got '$result'."
        }
    }
}

function Assert-CleanMachineReport {
    param([string]$Path)

    $text = Read-ReportText -Path $Path
    $requiredIdentity = @(
        'Tester',
        'Date and local time',
        'OS version and build',
        'Architecture',
        'Machine type',
        'Artifact filename',
        'App version',
        'Commit',
        'Build date',
        'Signing/notarization/package trust state',
        'Trust prompts, Gatekeeper prompts, antivirus/reputation prompts, or package-manager warnings'
    )
    foreach ($label in $requiredIdentity) {
        Assert-NonPlaceholderValue -Text $text -Label $label | Out-Null
    }

    $platform = Assert-NonPlaceholderValue -Text $text -Label 'Platform'
    Assert-ExactChoice -Text $text -Label 'Fresh profile or clean VM' -Expected 'yes' | Out-Null
    Assert-NonPlaceholderValue -Text $text -Label 'Upgrade test' | Out-Null
    Assert-Sha256Value -Text $text -Label 'Artifact SHA-256' | Out-Null
    Assert-ExactChoice -Text $text -Label 'Manifest/SBOM/provenance sidecars present' -Expected 'yes' | Out-Null
    Assert-ExactChoice -Text $text -Label 'Artifact hash matches manifest' -Expected 'pass' | Out-Null
    Assert-ExactChoice -Text $text -Label 'About/version output matches manifest' -Expected 'pass' | Out-Null
    Assert-ExactChoice -Text $text -Label 'Packaged smoke command run' -Expected 'pass' | Out-Null
    Assert-NonPlaceholderValue -Text $text -Label 'Packaged smoke command' | Out-Null
    Assert-NonPlaceholderValue -Text $text -Label 'Packaged smoke workspace path' | Out-Null
    Assert-ExactChoice -Text $text -Label 'Packaged smoke JSON/output archived' -Expected 'yes' | Out-Null

    Assert-TableRowsPass -Text $text -FirstColumnHeader 'Area' -ResultHeader 'Result' -RequiredRows @(
        'Normal launch path opens NexusDesk',
        'App icon/window title visible',
        'Help > About metadata matches manifest',
        'Home readiness renders with no workspace',
        'Open trusted sample workspace',
        'Project tree and recent workspace update',
        'Quick open/search basics work',
        'Preview supported text/Markdown file',
        'Edit, save, dirty guard, rollback/revert',
        'Configure local/test provider',
        'Test connection and model suggestion behavior',
        'Protected-secret backend status visible',
        'Ask workflow with pinned context and citations',
        'Low-risk Agent workflow shows approval/audit/jobs',
        'Profile/query small CSV/JSON/XLSX or SQLite sample',
        'Generate artifact and inspect preview/lineage/freshness',
        'Run Diagnostics',
        'Export redacted issue report',
        'Issue report excludes workspace contents by default',
        'Uninstall/remove app files',
        'Expected user data locations retained/documented'
    )

    if ($platform -match 'Windows') {
        foreach ($label in @('Windows version/build', 'Installer path used', 'Start Menu shortcut created', 'Authenticode signature state', 'Antivirus/reputation prompt state', 'Uninstall result')) {
            Assert-NonPlaceholderValue -Text $text -Label $label | Out-Null
        }
        Assert-ExactChoice -Text $text -Label 'DPAPI protected-secret smoke' -Expected 'pass' | Out-Null
        $platformName = 'Windows'
    } elseif ($platform -match 'macOS') {
        foreach ($label in @('macOS version/build', 'Package path used', 'Quarantine attribute present before first launch', 'Gatekeeper prompt state', 'Codesign verification result', 'Notarization/stapling state', 'App cleanup result')) {
            Assert-NonPlaceholderValue -Text $text -Label $label | Out-Null
        }
        Assert-ExactChoice -Text $text -Label 'Keychain protected-secret smoke' -Expected 'pass' | Out-Null
        $platformName = 'macOS'
    } elseif ($platform -match 'Linux') {
        foreach ($label in @('Distribution/version', 'Package path used', 'Runtime dependency issues', 'Desktop entry/icon behavior', 'Secret Service/libsecret behavior', 'Unsupported-secret refusal behavior if no keyring exists', 'App cleanup result')) {
            Assert-NonPlaceholderValue -Text $text -Label $label | Out-Null
        }
        $session = Assert-NonPlaceholderValue -Text $text -Label 'Desktop/session'
        if ($session -notmatch 'Wayland|X11|headless') {
            throw "Linux Desktop/session must include Wayland, X11, or headless; got '$session'."
        }
        $platformName = 'Linux'
    } else {
        throw "Clean-machine report platform must be Windows, macOS, or Linux; got '$platform'."
    }

    Assert-ExactChoice -Text $text -Label 'Overall result' -Expected 'pass' | Out-Null
    Assert-ExactChoice -Text $text -Label 'Blocks release' -Expected 'no' | Out-Null
    Assert-NonPlaceholderValue -Text $text -Label 'Logs, screenshots, diagnostics bundle paths' | Out-Null
    Assert-NonPlaceholderValue -Text $text -Label 'Reviewer' | Out-Null
    Assert-NonPlaceholderValue -Text $text -Label 'Release owner sign-off' | Out-Null

    return [pscustomobject]@{
        Path = (Resolve-Path -LiteralPath $Path).Path
        Platform = $platformName
        ArtifactSha256 = Get-BulletValue -Text $text -Label 'Artifact SHA-256'
    }
}

function Assert-BetaInstallReport {
    param([string]$Path)

    $text = Read-ReportText -Path $Path
    foreach ($label in @(
        'Tester',
        'Date and local time',
        'Platform and OS version',
        'Architecture',
        'Machine type',
        'Fresh install or upgrade',
        'Artifact filename',
        'App version',
        'Commit',
        'Build date',
        'Signing/notarization/package trust state',
        'Install/unpack path',
        'First launch path',
        'Trust prompts or warnings',
        'Antivirus/reputation warning',
        'Start Menu/Desktop/App launcher integration',
        'Time to first usable window',
        'Crash or hang observed',
        'Top positive signal',
        'Top confusing moment',
        'Top blocker',
        'Data-loss concern',
        'Security/trust concern',
        'Accessibility concern',
        'Performance concern',
        'Documentation gap',
        'Report received at',
        'Triage due by',
        'Triage completed at',
        'Owner',
        'Issues filed',
        'Fix/defer/not-planned decision',
        'User-facing release-note update needed'
    )) {
        Assert-NonPlaceholderValue -Text $text -Label $label | Out-Null
    }
    Assert-Sha256Value -Text $text -Label 'Artifact SHA-256' | Out-Null
    Assert-ExactChoice -Text $text -Label 'Would use again for a trusted local workspace' -Expected 'yes' | Out-Null
    Assert-ExactChoice -Text $text -Label 'Closed within 48 hours' -Expected 'yes' | Out-Null

    Assert-TableRowsPass -Text $text -FirstColumnHeader 'Flow' -ResultHeader 'Result' -RequiredRows @(
        'Launch and About metadata',
        'Open workspace',
        'Preview/edit/save/revert file',
        'Search workspace',
        'Provider setup and Test connection',
        'Ask with source context and citations',
        'Low-risk Agent with approval/audit/jobs',
        'Dataset/SQLite profile or query',
        'Artifact generation and preview',
        'Diagnostics report',
        'Redacted issue-report export',
        'Uninstall/remove app files',
        'User data retention understood'
    )

    $received = [datetime]::Parse((Get-BulletValue -Text $text -Label 'Report received at'))
    $due = [datetime]::Parse((Get-BulletValue -Text $text -Label 'Triage due by'))
    $completed = [datetime]::Parse((Get-BulletValue -Text $text -Label 'Triage completed at'))
    if ($due -gt $received.AddHours(48)) {
        throw "Beta report triage due date is later than 48 hours after receipt: $Path"
    }
    if ($completed -gt $received.AddHours(48)) {
        throw "Beta report triage completed later than 48 hours after receipt: $Path"
    }
    if ($completed -gt $due) {
        throw "Beta report triage completed after its due date: $Path"
    }

    return [pscustomobject]@{
        Path = (Resolve-Path -LiteralPath $Path).Path
        Tester = Get-BulletValue -Text $text -Label 'Tester'
        ArtifactSha256 = Get-BulletValue -Text $text -Label 'Artifact SHA-256'
    }
}

if ($CleanMachineReport.Count -eq 0 -and $BetaInstallReport.Count -eq 0) {
    throw 'Provide at least one -CleanMachineReport or -BetaInstallReport path.'
}

$cleanResults = @()
foreach ($path in $CleanMachineReport) {
    $cleanResults += Assert-CleanMachineReport -Path $path
}
if ($RequireAllCleanMachinePlatforms) {
    foreach ($platform in @('Windows', 'macOS', 'Linux')) {
        if (@($cleanResults | Where-Object { $_.Platform -eq $platform }).Count -eq 0) {
            throw "Missing required passing clean-machine report for platform: $platform"
        }
    }
}

$betaResults = @()
foreach ($path in $BetaInstallReport) {
    $betaResults += Assert-BetaInstallReport -Path $path
}
if ($RequireFiveBetaReports -and $betaResults.Count -lt 5) {
    throw "Expected at least five passing beta install reports; got $($betaResults.Count)."
}

Write-Host "Release validation reports verified."
Write-Host "Clean-machine reports: $($cleanResults.Count)"
foreach ($result in $cleanResults) {
    Write-Host "  $($result.Platform): $($result.Path)"
}
Write-Host "Beta install reports: $($betaResults.Count)"
foreach ($result in $betaResults) {
    Write-Host "  $($result.Tester): $($result.Path)"
}
