$ErrorActionPreference = "Stop"

$Repo = "kuraudo-lab/teleskope"
$InstallDir = if ($env:INSTALL_DIR) {
    $env:INSTALL_DIR
} else {
    Join-Path $env:LOCALAPPDATA "Microsoft\WindowsApps"
}

$ProcessorArchitecture = if ($env:PROCESSOR_ARCHITEW6432) {
    $env:PROCESSOR_ARCHITEW6432
} else {
    $env:PROCESSOR_ARCHITECTURE
}

$Architecture = switch ($ProcessorArchitecture) {
    "AMD64" { "amd64" }
    "ARM64" { "arm64" }
    default { throw "Unsupported architecture: $ProcessorArchitecture" }
}

function Invoke-WithRetry {
    param(
        [scriptblock] $Command,
        [int] $Attempts = 3
    )

    for ($Attempt = 1; $Attempt -le $Attempts; $Attempt++) {
        try {
            return & $Command
        } catch {
            if ($Attempt -eq $Attempts) {
                throw
            }
            Start-Sleep -Seconds 2
        }
    }
}

$Release = Invoke-WithRetry { Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" }
$Tag = $Release.tag_name
$Version = $Tag.TrimStart("v")
$Archive = "teleskope_${Version}_windows_${Architecture}.zip"
$BaseUrl = "https://github.com/$Repo/releases/download/$Tag"
$TempDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName())

New-Item -ItemType Directory -Path $TempDir | Out-Null
try {
    $ArchivePath = Join-Path $TempDir $Archive
    $ChecksumsPath = Join-Path $TempDir "checksums.txt"

    Invoke-WithRetry { Invoke-WebRequest -Uri "$BaseUrl/$Archive" -OutFile $ArchivePath }
    Invoke-WithRetry { Invoke-WebRequest -Uri "$BaseUrl/checksums.txt" -OutFile $ChecksumsPath }

    $ChecksumLine = Get-Content $ChecksumsPath | Where-Object { ($_ -split "\s+")[1] -eq $Archive }
    if (-not $ChecksumLine) {
        throw "Checksum entry not found for $Archive"
    }

    $Expected = ($ChecksumLine -split "\s+")[0].ToLowerInvariant()
    $Actual = (Get-FileHash -Algorithm SHA256 $ArchivePath).Hash.ToLowerInvariant()
    if ($Actual -ne $Expected) {
        throw "Checksum mismatch for $Archive"
    }

    Expand-Archive -Path $ArchivePath -DestinationPath $TempDir -Force
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Copy-Item -Path (Join-Path $TempDir "teleskope.exe") -Destination (Join-Path $InstallDir "teleskope.exe") -Force

    Write-Host "Installed teleskope $Version to $(Join-Path $InstallDir "teleskope.exe")"
    Write-Host "Run: teleskope --version"
} finally {
    Remove-Item -Recurse -Force $TempDir -ErrorAction SilentlyContinue
}
