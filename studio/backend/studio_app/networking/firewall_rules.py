"""Read-only Windows Firewall questions for the station gateway (ADR-0062): ScadaBench's own
gateway rules, and enabled inbound rules that block the gateway worker. No elevation needed."""
import json
import os
import subprocess


def _powershell(script, timeout=20):
    flags = subprocess.CREATE_NO_WINDOW if os.name == "nt" else 0
    completed = subprocess.run(["powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script],
                               capture_output=True, text=True, timeout=timeout, creationflags=flags)
    if completed.returncode != 0:
        raise RuntimeError("Windows Firewall could not be read: "
                           + (completed.stderr.strip().splitlines() or ["unknown error"])[-1])
    return completed.stdout


def _ps_string(value):
    return "'" + str(value).replace("'", "''") + "'"


def list_gateway_rules(run=_powershell):
    script = ("Get-NetFirewallRule -Name 'ScadaBench-Gateway-*' -ErrorAction SilentlyContinue | "
              "Select-Object -ExpandProperty Name")
    return [line.strip() for line in run(script).splitlines() if line.strip()]


def blocking_rules(program, run=_powershell):
    """Enabled inbound TCP Block rules naming exactly this program (paths compare without case)."""
    script = ("$found = @(Get-NetFirewallRule -Direction Inbound -Action Block -Enabled True "
              "-ErrorAction SilentlyContinue | Where-Object { ($_ | Get-NetFirewallApplicationFilter).Program "
              f"-eq {_ps_string(program)} -and ($_ | Get-NetFirewallPortFilter).Protocol -in 'TCP','Any' }} "
              "| ForEach-Object { @{ name = $_.Name; display = $_.DisplayName } }); "
              "ConvertTo-Json -InputObject $found -Compress")
    text = run(script, timeout=60).strip()
    found = json.loads(text) if text else []
    found = [found] if isinstance(found, dict) else found
    return [{"name": str(rule["name"]), "display": str(rule.get("display", ""))} for rule in found]


def gateway_state(unblock_names, run=_powershell):
    """One PowerShell call for the clean-up list (change 0089: one call per item took ~10 s):
    ScadaBench's gateway rules, and which of the given Windows rules exist and are switched off."""
    names = ",".join(_ps_string(name) for name in unblock_names) or "$null"
    script = ("$g = @(Get-NetFirewallRule -Name 'ScadaBench-Gateway-*' -ErrorAction SilentlyContinue | "
              "ForEach-Object { $_.Name }); "
              f"$d = @(foreach ($n in @({names})) {{ if ($n) {{ $r = Get-NetFirewallRule -Name $n -ErrorAction "
              "SilentlyContinue; if ($r -and \"$($r.Enabled)\" -eq 'False') { $n } } }); "
              "ConvertTo-Json -InputObject @{ gateway = $g; disabled = $d } -Compress")
    text = run(script, timeout=60).strip()
    found = json.loads(text) if text else {}
    return set(found.get("gateway") or []), set(found.get("disabled") or [])
