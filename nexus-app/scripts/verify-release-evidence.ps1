param(
    [Parameter(Mandatory = $true)]
    [string]$Artifact,
    [Parameter(Mandatory = $true)]
    [string]$Manifest,
    [Parameter(Mandatory = $true)]
    [string]$SBOM,
    [Parameter(Mandatory = $true)]
    [string]$Provenance
)

$ErrorActionPreference = 'Stop'

function Read-Json {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) {
        throw "Required evidence file is missing: $Path"
    }
    return Get-Content -LiteralPath $Path -Raw | ConvertFrom-Json
}

function Get-FileSHA256 {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) {
        throw "Required artifact is missing: $Path"
    }
    return (Get-FileHash -Algorithm SHA256 -LiteralPath $Path).Hash.ToLowerInvariant()
}

function Assert-Equal {
    param(
        [string]$Name,
        [object]$Actual,
        [object]$Expected
    )
    if ("$Actual" -ne "$Expected") {
        throw "$Name mismatch: got '$Actual', expected '$Expected'"
    }
}

$artifactItem = Get-Item -LiteralPath $Artifact
$manifestJson = Read-Json -Path $Manifest
$sbomJson = Read-Json -Path $SBOM
$provenanceJson = Read-Json -Path $Provenance

$artifactSHA = Get-FileSHA256 -Path $Artifact
Assert-Equal -Name 'manifest artifactName' -Actual $manifestJson.artifactName -Expected $artifactItem.Name
Assert-Equal -Name 'manifest artifactSize' -Actual $manifestJson.artifactSize -Expected $artifactItem.Length
Assert-Equal -Name 'manifest artifactSha256' -Actual $manifestJson.artifactSha256 -Expected $artifactSHA

if ($sbomJson.bomFormat -ne 'CycloneDX') {
    throw "SBOM bomFormat mismatch: $($sbomJson.bomFormat)"
}
$componentHashes = @($sbomJson.metadata.component.hashes | ForEach-Object { $_.content })
if ($componentHashes -notcontains $artifactSHA) {
    throw "SBOM component hashes do not include artifact SHA256 $artifactSHA"
}

Assert-Equal -Name 'provenance subject artifactName' -Actual $provenanceJson.subject.artifactName -Expected $manifestJson.artifactName
Assert-Equal -Name 'provenance subject artifactSize' -Actual $provenanceJson.subject.artifactSize -Expected $manifestJson.artifactSize
Assert-Equal -Name 'provenance subject artifactSha256' -Actual $provenanceJson.subject.artifactSha256 -Expected $manifestJson.artifactSha256
Assert-Equal -Name 'provenance subject version' -Actual $provenanceJson.subject.version -Expected $manifestJson.version
Assert-Equal -Name 'provenance subject commit' -Actual $provenanceJson.subject.commit -Expected $manifestJson.commit
Assert-Equal -Name 'provenance subject buildDate' -Actual $provenanceJson.subject.buildDate -Expected $manifestJson.buildDate

$manifestEvidence = @($provenanceJson.evidence | Where-Object { $_.kind -eq 'release-manifest' } | Select-Object -First 1)
$sbomEvidence = @($provenanceJson.evidence | Where-Object { $_.kind -eq 'sbom' } | Select-Object -First 1)
if ($manifestEvidence.Count -eq 0) {
    throw 'Provenance is missing release-manifest evidence.'
}
if ($sbomEvidence.Count -eq 0) {
    throw 'Provenance is missing SBOM evidence.'
}
Assert-Equal -Name 'manifest evidence path' -Actual $manifestEvidence[0].path -Expected (Split-Path -Leaf $Manifest)
Assert-Equal -Name 'manifest evidence sha' -Actual $manifestEvidence[0].sha256 -Expected (Get-FileSHA256 -Path $Manifest)
Assert-Equal -Name 'sbom evidence path' -Actual $sbomEvidence[0].path -Expected (Split-Path -Leaf $SBOM)
Assert-Equal -Name 'sbom evidence sha' -Actual $sbomEvidence[0].sha256 -Expected (Get-FileSHA256 -Path $SBOM)

Write-Host "Release evidence verified."
Write-Host "Artifact: $($artifactItem.FullName)"
Write-Host "SHA256: $artifactSHA"
Write-Host "Manifest: $Manifest"
Write-Host "SBOM: $SBOM"
Write-Host "Provenance: $Provenance"
