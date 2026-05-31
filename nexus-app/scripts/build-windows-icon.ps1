param(
    [string]$IconPng = 'internal\brand\assets\nexus-app-icon-transparent.png',
    [string]$OutDir = 'build\windows-resource',
    [string]$SysoPath = 'resource_windows.syso'
)

$ErrorActionPreference = 'Stop'

$appRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path

function Resolve-AppPath {
    param([string]$Path)

    if ([System.IO.Path]::IsPathRooted($Path)) {
        return $Path
    }

    return (Join-Path $appRoot $Path)
}

$iconPngPath = (Resolve-Path (Resolve-AppPath $IconPng)).Path
$outDirPath = Resolve-AppPath $OutDir
$sysoPathValue = Resolve-AppPath $SysoPath
New-Item -ItemType Directory -Force -Path $outDirPath | Out-Null

$icoPath = Join-Path $outDirPath 'nexus-app-icon.ico'
$rcPath = Join-Path $outDirPath 'nexus-app-icon.rc'

Add-Type -AssemblyName System.Drawing

$source = [System.Drawing.Image]::FromFile($iconPngPath)
try {
    $size = 256
    $bitmap = New-Object System.Drawing.Bitmap $size, $size, ([System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
    try {
        $graphics = [System.Drawing.Graphics]::FromImage($bitmap)
        try {
            $graphics.Clear([System.Drawing.Color]::Transparent)
            $graphics.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
            $graphics.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
            $graphics.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality

            $scale = [Math]::Min($size / $source.Width, $size / $source.Height)
            $width = [int][Math]::Round($source.Width * $scale)
            $height = [int][Math]::Round($source.Height * $scale)
            $x = [int][Math]::Floor(($size - $width) / 2)
            $y = [int][Math]::Floor(($size - $height) / 2)
            $graphics.DrawImage($source, $x, $y, $width, $height)
        } finally {
            $graphics.Dispose()
        }

        $pngStream = New-Object System.IO.MemoryStream
        try {
            $bitmap.Save($pngStream, [System.Drawing.Imaging.ImageFormat]::Png)
            $pngBytes = $pngStream.ToArray()
        } finally {
            $pngStream.Dispose()
        }
    } finally {
        $bitmap.Dispose()
    }
} finally {
    $source.Dispose()
}

$icoStream = [System.IO.File]::Create($icoPath)
try {
    $writer = New-Object System.IO.BinaryWriter $icoStream
    try {
        $writer.Write([UInt16]0)
        $writer.Write([UInt16]1)
        $writer.Write([UInt16]1)
        $writer.Write([Byte]0)
        $writer.Write([Byte]0)
        $writer.Write([Byte]0)
        $writer.Write([Byte]0)
        $writer.Write([UInt16]1)
        $writer.Write([UInt16]32)
        $writer.Write([UInt32]$pngBytes.Length)
        $writer.Write([UInt32]22)
        $writer.Write($pngBytes)
    } finally {
        $writer.Dispose()
    }
} finally {
    $icoStream.Dispose()
}

$escapedIconPath = (Resolve-Path $icoPath).Path.Replace('\', '\\')
Set-Content -Path $rcPath -Value "1 ICON `"$escapedIconPath`"" -Encoding ASCII

$windres = Get-Command windres -ErrorAction SilentlyContinue
if ($null -eq $windres) {
    throw 'windres.exe was not found on PATH. Run scripts\dev-env.ps1 so MSYS2 UCRT64 bin is available.'
}

$windresVersionOutput = & $windres.Source --version 2>&1
if ($LASTEXITCODE -ne 0) {
    $windresMessage = ($windresVersionOutput | Out-String).Trim()
    if ([string]::IsNullOrWhiteSpace($windresMessage)) {
        $windresMessage = "windres.exe exited with code $LASTEXITCODE before printing diagnostics."
    }
    throw "windres.exe is installed but cannot start. Repair MSYS2 UCRT64 packages mingw-w64-ucrt-x86_64-binutils and mingw-w64-ucrt-x86_64-zlib, then open a fresh PowerShell. Details: $windresMessage"
}

$windresArgs = @('-O', 'coff', '-F', 'pe-x86-64', '-i', $rcPath, '-o', $sysoPathValue)
if (-not [string]::IsNullOrWhiteSpace($env:CC)) {
    $windresArgs = @("--preprocessor=$env:CC -E -xc-header -DRC_INVOKED") + $windresArgs
}

& $windres.Source @windresArgs
if ($LASTEXITCODE -ne 0) {
    throw "windres.exe failed with exit code $LASTEXITCODE."
}
Write-Host "Windows icon resource generated at $sysoPathValue"
