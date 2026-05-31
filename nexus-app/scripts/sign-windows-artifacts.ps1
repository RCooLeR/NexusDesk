param(
    [Parameter(Mandatory = $true)]
    [string[]]$FilePath,
    [string]$CertificateThumbprint = $env:NEXUSDESK_WINDOWS_CERT_THUMBPRINT,
    [string]$PfxPath = $env:NEXUSDESK_WINDOWS_PFX_PATH,
    [string]$PfxPassword = $env:NEXUSDESK_WINDOWS_PFX_PASSWORD,
    [string]$TimestampUrl = $(if ([string]::IsNullOrWhiteSpace($env:NEXUSDESK_WINDOWS_TIMESTAMP_URL)) { 'http://timestamp.digicert.com' } else { $env:NEXUSDESK_WINDOWS_TIMESTAMP_URL }),
    [string]$EvidencePath = "",
    [switch]$VerifyOnly
)

$ErrorActionPreference = 'Stop'

function Get-WindowsSigningCertificate {
    if (-not [string]::IsNullOrWhiteSpace($CertificateThumbprint)) {
        $thumbprint = ($CertificateThumbprint -replace '\s', '').ToUpperInvariant()
        $cert = Get-Item -LiteralPath "Cert:\CurrentUser\My\$thumbprint" -ErrorAction SilentlyContinue
        if ($null -ne $cert) {
            return $cert
        }
        $cert = Get-Item -LiteralPath "Cert:\LocalMachine\My\$thumbprint" -ErrorAction SilentlyContinue
        if ($null -ne $cert) {
            return $cert
        }
        throw "Windows signing certificate thumbprint was not found in CurrentUser or LocalMachine MY store: $CertificateThumbprint"
    }

    if (-not [string]::IsNullOrWhiteSpace($PfxPath)) {
        if (-not (Test-Path -LiteralPath $PfxPath)) {
            throw "Windows signing PFX was not found: $PfxPath"
        }
        if ([string]::IsNullOrWhiteSpace($PfxPassword)) {
            throw 'NEXUSDESK_WINDOWS_PFX_PASSWORD or -PfxPassword is required when -PfxPath is used.'
        }
        $securePassword = ConvertTo-SecureString -String $PfxPassword -AsPlainText -Force
        return [System.Security.Cryptography.X509Certificates.X509Certificate2]::new(
            $PfxPath,
            $securePassword,
            [System.Security.Cryptography.X509Certificates.X509KeyStorageFlags]::EphemeralKeySet
        )
    }

    throw 'Provide -CertificateThumbprint / NEXUSDESK_WINDOWS_CERT_THUMBPRINT or -PfxPath / NEXUSDESK_WINDOWS_PFX_PATH before signing Windows artifacts.'
}

function Assert-AuthenticodeValid {
    param([string]$Path)

    $signature = Get-AuthenticodeSignature -LiteralPath $Path
    if ($signature.Status -ne 'Valid') {
        throw "Authenticode verification failed for $Path with status $($signature.Status): $($signature.StatusMessage)"
    }
    return $signature
}

function ConvertTo-CertificateEvidence {
    param($Certificate)

    if ($null -eq $Certificate) {
        return $null
    }

    return [ordered]@{
        subject      = $Certificate.Subject
        issuer       = $Certificate.Issuer
        thumbprint   = $Certificate.Thumbprint
        notBeforeUtc = $Certificate.NotBefore.ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
        notAfterUtc  = $Certificate.NotAfter.ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
    }
}

function New-SigningEvidenceRecord {
    param(
        [string]$Path,
        $Signature,
        [string]$Mode,
        [string]$TimestampServer,
        [datetime]$GeneratedAt
    )

    $hash = Get-FileHash -LiteralPath $Path -Algorithm SHA256
    return [ordered]@{
        path            = $Path
        fileName        = [System.IO.Path]::GetFileName($Path)
        sha256          = $hash.Hash.ToLowerInvariant()
        mode            = $Mode
        signatureStatus = $Signature.Status.ToString()
        statusMessage   = $Signature.StatusMessage
        signatureType   = $Signature.SignatureType.ToString()
        signer          = ConvertTo-CertificateEvidence -Certificate $Signature.SignerCertificate
        timestamper     = ConvertTo-CertificateEvidence -Certificate $Signature.TimeStamperCertificate
        timestampServer = $TimestampServer
        recordedAtUtc   = $GeneratedAt.ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
    }
}

function Write-SigningEvidence {
    param(
        [string]$Path,
        [object[]]$Records,
        [string]$Mode,
        [datetime]$GeneratedAt
    )

    if ([string]::IsNullOrWhiteSpace($Path)) {
        return
    }

    $resolvedEvidencePath = [System.IO.Path]::GetFullPath($Path)
    $evidenceDir = Split-Path -Parent $resolvedEvidencePath
    if (-not [string]::IsNullOrWhiteSpace($evidenceDir)) {
        New-Item -ItemType Directory -Force -Path $evidenceDir | Out-Null
    }

    $payload = [ordered]@{
        schemaVersion = 1
        generatedAtUtc = $GeneratedAt.ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
        mode = $Mode
        artifactCount = $Records.Count
        artifacts = $Records
    }
    $payload | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $resolvedEvidencePath -Encoding UTF8
    Write-Host "Wrote Windows signing evidence: $resolvedEvidencePath"
}

$resolvedPaths = @()
foreach ($path in $FilePath) {
    if (-not (Test-Path -LiteralPath $path)) {
        throw "Artifact to sign was not found: $path"
    }
    $resolvedPaths += (Resolve-Path -LiteralPath $path).Path
}

if ($VerifyOnly) {
    $generatedAt = (Get-Date).ToUniversalTime()
    $records = @()
    foreach ($path in $resolvedPaths) {
        $signature = Assert-AuthenticodeValid -Path $path
        $records += New-SigningEvidenceRecord -Path $path -Signature $signature -Mode 'verify' -TimestampServer $null -GeneratedAt $generatedAt
        Write-Host "Verified Authenticode signature: $path"
        Write-Host "Signer: $($signature.SignerCertificate.Subject)"
    }
    Write-SigningEvidence -Path $EvidencePath -Records $records -Mode 'verify' -GeneratedAt $generatedAt
    return
}

$certificate = Get-WindowsSigningCertificate
$generatedAt = (Get-Date).ToUniversalTime()
$records = @()
foreach ($path in $resolvedPaths) {
    $result = Set-AuthenticodeSignature -LiteralPath $path -Certificate $certificate -TimestampServer $TimestampUrl -HashAlgorithm SHA256
    if ($result.Status -ne 'Valid') {
        throw "Signing failed for $path with status $($result.Status): $($result.StatusMessage)"
    }
    $signature = Assert-AuthenticodeValid -Path $path
    $records += New-SigningEvidenceRecord -Path $path -Signature $signature -Mode 'sign' -TimestampServer $TimestampUrl -GeneratedAt $generatedAt
    Write-Host "Signed Windows artifact: $path"
    Write-Host "Signer: $($signature.SignerCertificate.Subject)"
    Write-Host "Timestamp server: $TimestampUrl"
}
Write-SigningEvidence -Path $EvidencePath -Records $records -Mode 'sign' -GeneratedAt $generatedAt
