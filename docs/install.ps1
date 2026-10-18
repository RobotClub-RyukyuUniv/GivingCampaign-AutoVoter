# Install the latest gcvote (Windows). In PowerShell:
#   irm https://robotclub-ryukyuuniv.github.io/GivingCampaign-AutoVoter/install.ps1 | iex
# (irm | iex may not decode UTF-8 on Windows PowerShell 5.1, so messages are ASCII only)
$ErrorActionPreference = 'Stop'

$repo = 'RobotClub-RyukyuUniv/GivingCampaign-AutoVoter'
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'arm64' } else { 'amd64' }
$name = "gcvote_windows_$arch.zip"
$base = "https://github.com/$repo/releases/latest/download"
$dir = Join-Path $env:LOCALAPPDATA 'gcvote'
$tmp = Join-Path ([IO.Path]::GetTempPath()) ([Guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null

try {
    Write-Host "Downloading $name"
    Invoke-WebRequest -UseBasicParsing -Uri "$base/$name" -OutFile "$tmp\$name"
    Invoke-WebRequest -UseBasicParsing -Uri "$base/checksums.txt" -OutFile "$tmp\checksums.txt"

    $want = (Get-Content "$tmp\checksums.txt" | Where-Object { $_ -match " $([regex]::Escape($name))$" }) -split ' ' | Select-Object -First 1
    $got = (Get-FileHash -Algorithm SHA256 "$tmp\$name").Hash.ToLower()
    if (-not $want -or $want -ne $got) { throw 'Checksum mismatch. Please try again.' }

    Expand-Archive -Path "$tmp\$name" -DestinationPath "$tmp\x" -Force
    New-Item -ItemType Directory -Path $dir -Force | Out-Null
    Copy-Item "$tmp\x\gcvote.exe" "$dir\gcvote.exe" -Force
} finally {
    Remove-Item -Recurse -Force $tmp
}

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if (($userPath -split ';') -notcontains $dir) {
    [Environment]::SetEnvironmentVariable('Path', "$dir;$userPath", 'User')
    Write-Host "Added $dir to your PATH. Open a new terminal to use gcvote."
}
Write-Host "Installed: $dir\gcvote.exe ($(& "$dir\gcvote.exe" version))"
