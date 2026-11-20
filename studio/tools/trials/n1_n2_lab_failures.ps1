# Trials N1 and N2 (speed and network trials 2026-11-20, change 0221).
# Drives the installed ScadaBench lab network service and reads Windows' own address list; only the
# project and the coordinator are stand-ins (studio/engine/internal/labsetup/trial_n1n2_test.go).
#   N1: one accepted and two refused addresses (the service refuses loopback 127.67.21.x).
#   N2: the service adds two addresses but its answers are lost; then Stop.
#   Timing: Run's address step and Stop's release for 7 and 20 IEDs (TRIAL_RUNS, default 10).
# Uses 198.18.21.x on the ScadaBench Lab card; addresses of anyone else stay untouched.
# No administrator prompt. Not part of verify.py.
param([int]$Runs = 10, [string]$Only = 'TestTrial')

$repo = Resolve-Path (Join-Path $PSScriptRoot '..\..\..')
$tmp = Join-Path $repo '.local\gotmp'
New-Item -ItemType Directory -Force $tmp | Out-Null
$env:GOTMPDIR = $tmp; $env:TEMP = $tmp; $env:TMP = $tmp; $env:TRIAL_RUNS = "$Runs"
go -C (Join-Path $repo 'studio\engine') test -tags trial -run $Only -v -count=1 ./internal/labsetup/ |
  Where-Object { $_ -notmatch '^=== ' }
