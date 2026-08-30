# Install ms2pdf from GitHub Releases for Windows.
# Usage:
#   irm https://github.com/sukujgrg/ms2pdf/releases/latest/download/install.ps1 | iex
#   $env:MS2PDF_VERSION='v0.1.1'; irm https://github.com/sukujgrg/ms2pdf/releases/latest/download/install.ps1 | iex
$ErrorActionPreference = 'Stop'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
$ProgressPreference = 'SilentlyContinue'

$Repo = 'sukujgrg/ms2pdf'
$Bin = 'ms2pdf.exe'
$InstallDir = if ($env:MS2PDF_INSTALL_DIR) { $env:MS2PDF_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'ms2pdf' }

function Die([string]$Message) {
	Write-Error $Message
	exit 1
}

if ($env:OS -ne 'Windows_NT') {
	Die 'use install.sh on macOS or Linux'
}

$arch = $env:PROCESSOR_ARCHITECTURE
switch -Regex ($arch) {
	'^AMD64$' { $assetArch = 'amd64' }
	'^ARM64$' { $assetArch = 'amd64' } # no windows_arm64 build; amd64 runs under emulation
	default { Die "unsupported architecture: $arch" }
}

$version = $env:MS2PDF_VERSION
if (-not $version -and $args.Count -ge 1) { $version = [string]$args[0] }
if (-not $version -or $version -eq 'latest') {
	$latest = Invoke-RestMethod -Headers @{ 'User-Agent' = 'ms2pdf-install' } "https://api.github.com/repos/$Repo/releases/latest"
	$version = $latest.tag_name
}
if ($version -notmatch '^v') { $version = "v$version" }

$asset = "ms2pdf_${version}_windows_${assetArch}.zip"
$base = "https://github.com/$Repo/releases/download/$version"
$tmp = Join-Path ([IO.Path]::GetTempPath()) ("ms2pdf-" + [guid]::NewGuid().ToString('n'))
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
	$zip = Join-Path $tmp $asset
	$sums = Join-Path $tmp 'SHA256SUMS'
	Invoke-WebRequest -UseBasicParsing -Uri "$base/$asset" -OutFile $zip
	Invoke-WebRequest -UseBasicParsing -Uri "$base/SHA256SUMS" -OutFile $sums

	$line = Select-String -Path $sums -Pattern ([regex]::Escape($asset)) | Select-Object -First 1
	if (-not $line) { Die "SHA256SUMS has no entry for $asset" }
	$want = ($line.Line -split '\s+')[0]
	$got = (Get-FileHash -Algorithm SHA256 -Path $zip).Hash
	if ($want.ToLowerInvariant() -ne $got.ToLowerInvariant()) { Die "checksum mismatch for $asset" }

	Expand-Archive -Path $zip -DestinationPath $tmp -Force
	$exe = Join-Path $tmp $Bin
	if (-not (Test-Path -LiteralPath $exe)) { Die "archive did not contain $Bin" }

	New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
	$dest = Join-Path $InstallDir $Bin
	Copy-Item -LiteralPath $exe -Destination $dest -Force
	Write-Host "installed $dest ($version)"

	$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
	if (-not $userPath) { $userPath = '' }
	$parts = $userPath -split ';' | Where-Object { $_ }
	if ($parts -notcontains $InstallDir) {
		[Environment]::SetEnvironmentVariable('Path', (@($parts) + $InstallDir) -join ';', 'User')
		$env:Path = "$env:Path;$InstallDir"
		Write-Host "added $InstallDir to user PATH; open a new terminal if ms2pdf is not found"
	}
}
finally {
	Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
}
