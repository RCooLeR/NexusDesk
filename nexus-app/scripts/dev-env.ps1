param(
    [switch]$Build,
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

if (-not (Test-Path (Join-Path $ucrtBin 'gcc.exe'))) {
    throw "MSYS2 UCRT64 gcc.exe was not found at $ucrtBin. Install MSYS2 and the mingw-w64-ucrt-x86_64-gcc package."
}

$env:PATH = "$ucrtBin;$usrBin;$env:PATH"
$env:CGO_ENABLED = '1'
if ([string]::IsNullOrWhiteSpace($env:GOFLAGS)) {
    $env:GOFLAGS = '-mod=readonly'
}

Write-Host "Nexus native dev environment ready."
Write-Host "gcc: $((Get-Command gcc).Source)"
Write-Host "CGO_ENABLED=$env:CGO_ENABLED"
Write-Host "GOFLAGS=$env:GOFLAGS"

if ($Test) {
    go test ./internal/domain ./internal/services/... ./internal/ui/shell ./internal/ui/theme ./internal/brand
}

if ($Build) {
    New-Item -ItemType Directory -Force -Path build | Out-Null
    go build -o build\nexusdesk.exe .
}

if ($Run) {
    go run .
}
