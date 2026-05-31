param(
    [Parameter(Mandatory = $true)]
    [string[]]$FilePath,
    [string]$CertificateThumbprint = $env:NEXUSDESK_WINDOWS_CERT_THUMBPRINT,
    [string]$PfxPath = $env:NEXUSDESK_WINDOWS_PFX_PATH,
    [string]$PfxPassword = $env:NEXUSDESK_WINDOWS_PFX_PASSWORD,
    [string]$TimestampUrl = $(if ([string]::IsNullOrWhiteSpace($env:NEXUSDESK_WINDOWS_TIMESTAMP_URL)) { 'http://timestamp.digicert.com' } else { $env:NEXUSDESK_WINDOWS_TIMESTAMP_URL }),
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

$resolvedPaths = @()
foreach ($path in $FilePath) {
    if (-not (Test-Path -LiteralPath $path)) {
        throw "Artifact to sign was not found: $path"
    }
    $resolvedPaths += (Resolve-Path -LiteralPath $path).Path
}

if ($VerifyOnly) {
    foreach ($path in $resolvedPaths) {
        $signature = Assert-AuthenticodeValid -Path $path
        Write-Host "Verified Authenticode signature: $path"
        Write-Host "Signer: $($signature.SignerCertificate.Subject)"
    }
    return
}

$certificate = Get-WindowsSigningCertificate
foreach ($path in $resolvedPaths) {
    $result = Set-AuthenticodeSignature -LiteralPath $path -Certificate $certificate -TimestampServer $TimestampUrl -HashAlgorithm SHA256
    if ($result.Status -ne 'Valid') {
        throw "Signing failed for $path with status $($result.Status): $($result.StatusMessage)"
    }
    $signature = Assert-AuthenticodeValid -Path $path
    Write-Host "Signed Windows artifact: $path"
    Write-Host "Signer: $($signature.SignerCertificate.Subject)"
    Write-Host "Timestamp server: $TimestampUrl"
}
