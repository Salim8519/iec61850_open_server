# Trial N8 (speed and network trials 2026-11-20, change 0221).
# Puts 198.18.21.50 on the ScadaBench Lab card through the installed lab network service, lets a
# gateway status watch it with the real Windows reader, takes the address off and puts it back,
# and prints how long the status took to say "Address gone" and to clear
# (studio/engine/internal/gateway/service/trial_n8_test.go). -Before shows today's code (no watch).
# No administrator prompt. Not part of verify.py.
param([int]$Runs = 10, [switch]$Before)

$repo = Resolve-Path (Join-Path $PSScriptRoot '..\..\..')
$tmp = Join-Path $repo '.local\gotmp'
New-Item -ItemType Directory -Force $tmp | Out-Null
$env:GOTMPDIR = $tmp; $env:TEMP = $tmp; $env:TMP = $tmp
if ($Before) { $env:TRIAL_BEFORE = '1' } else { Remove-Item Env:TRIAL_BEFORE -ErrorAction SilentlyContinue }
foreach ($run in 1..$Runs) {
  go -C (Join-Path $repo 'studio\engine') test -tags trial -run TestTrialN8 -v -count=1 ./internal/gateway/service/ |
    Where-Object { $_ -match '\[N8\]|FAIL' }
}
