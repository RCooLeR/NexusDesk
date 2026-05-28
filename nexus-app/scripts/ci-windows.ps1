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

$ucrtBin = Join-Path $MsysRoot 'ucrt64\bin'
$usrBin = Join-Path $MsysRoot 'usr\bin'

if (-not (Test-Path (Join-Path $ucrtBin 'gcc.exe'))) {
    throw "MSYS2 UCRT64 gcc.exe was not found at $ucrtBin. Install MSYS2 and mingw-w64-ucrt-x86_64-gcc."
}

$env:PATH = "$ucrtBin;$usrBin;$env:PATH"
$env:CGO_ENABLED = '1'
$env:GOFLAGS = '-mod=readonly'

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
    go test ./...

    Write-Host 'Running static analysis...'
    go vet ./...

    if (-not $SkipBuild) {
        Write-Host 'Building native Windows executable...'
        New-Item -ItemType Directory -Force -Path build | Out-Null
        go build -o build\nexusdesk.exe .
    }

    Write-Host 'Checking diff whitespace...'
    git -C $repoRoot diff --check
} finally {
    Remove-Item -LiteralPath (Join-Path $appRoot 'nexusdesk.exe') -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath (Join-Path $appRoot 'build\nexusdesk.exe') -Force -ErrorAction SilentlyContinue
    Pop-Location
}
