# Builds the release record (JSON) for the version scadabench.com offers: the version from
# website\src\data\site.ts, a one-line summary from the changelog, and the SHA-256 of the
# installer and portable exe in studio\dist-protected. Writes the JSON to standard output.
$ErrorActionPreference = "Stop"
$root = Resolve-Path (Join-Path $PSScriptRoot "..\..")
$site = Get-Content (Join-Path $root "website\src\data\site.ts") -Raw
if ($site -notmatch "const VERSION = '([0-9]+\.[0-9]+\.[0-9]+)'") { throw "Could not read VERSION from website\src\data\site.ts" }
$version = $Matches[1]
$changelog = Get-Content (Join-Path $root "website\src\pages\changelog.astro") -Raw
$notes = ""
if ($changelog -match "<h2>[^<]*?·\s*([^<]+)</h2>") { $notes = $Matches[1].Trim() }
$dist = Join-Path $root "studio\dist-protected"
function Download($name, $url) {
    $file = Join-Path $dist $name
    if (-not (Test-Path $file)) { return $null }
    return [ordered]@{ url = $url; sha256 = (Get-FileHash $file -Algorithm SHA256).Hash.ToLowerInvariant() }
}
$installer = Download "ScadaBench-Setup-$version.exe" "https://scadabench.com/download/ScadaBench-Setup-$version.exe"
if (-not $installer) { throw "No ScadaBench-Setup-$version.exe in studio\dist-protected, so apps are not told about $version." }
$record = [ordered]@{ version = $version; date = (Get-Date -Format "yyyy-MM-dd"); notes = $notes
    page = "https://scadabench.com/download"; installer = $installer }
$portable = Download "ScadaBench-Portable-$version.exe" "https://download.scadabench.com/download/ScadaBench-Portable-$version.exe"
if ($portable) { $record.portable = $portable }
$record | ConvertTo-Json -Compress
