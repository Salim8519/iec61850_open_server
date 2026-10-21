"""Stop every ScadaBench process on this PC with one command.

Usage: python studio/tools/stop_all.py [--dry-run]
Stops the app (ScadaBench.exe, the portable exe, studio_server.exe), the native and Go workers
(IED, live, probe, control, gateway, master, GOOSE, SNMP, XSD), the development launchers and
the Vite dev server of this repository. Each process is stopped with its children.
Never stopped: the network helper (it may be changing adapter settings; let it finish), and
Python or Node processes that do not belong to this repository.
"""
from pathlib import Path
import json
import os
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[2]
APP_EXES = {"scadabench.exe", "scadabench-portable.exe", "studio_server.exe", "studio_ied.exe",
            "studio_live.exe", "studio_probe.exe", "studio_control.exe", "studio_gateway.exe",
            "studio_master.exe", "studio_goose.exe", "studio_snmp.exe", "studio_xsd.exe"}
# Python and Node belong to ScadaBench only when they run its launchers, backend or Vite.
DEV_MARKERS = ("run_dev.py", "run_dev_engine.py", "studio_app", "vite")
LIST_COMMAND = ("Get-CimInstance Win32_Process | Select-Object ProcessId, ParentProcessId, Name, "
                "CommandLine | ConvertTo-Json -Compress")


def processes():
    """Every process with its id, parent, name and command line (empty when Windows hides it)."""
    answer = subprocess.run(["powershell", "-NoProfile", "-Command", LIST_COMMAND],
                            capture_output=True, text=True, check=True)
    rows = json.loads(answer.stdout or "[]")
    return [rows] if isinstance(rows, dict) else rows


def is_scadabench(row):
    """True for ScadaBench's own processes; the network helper is always left alone."""
    name = (row.get("Name") or "").lower()
    command = row.get("CommandLine") or ""
    if "--network-helper" in command:
        return False
    if name in APP_EXES:
        return True
    if name in ("python.exe", "pythonw.exe", "node.exe"):
        lowered = command.lower().replace("/", "\\")
        return str(ROOT).lower() in lowered and any(marker in lowered for marker in DEV_MARKERS)
    return False


def targets(rows, own=os.getpid()):
    """The ScadaBench processes, without this script, its parents and children of a target."""
    chosen = {row["ProcessId"]: row for row in rows if is_scadabench(row) and row["ProcessId"] != own}
    # taskkill /T stops the children too, so only the top of each tree is stopped by id.
    return [row for row in chosen.values() if row.get("ParentProcessId") not in chosen]


def stop(row):
    """Stop one process tree; returns the reason when Windows refuses."""
    answer = subprocess.run(["taskkill", "/PID", str(row["ProcessId"]), "/T", "/F"],
                            capture_output=True, text=True)
    return "" if answer.returncode == 0 else (answer.stderr or answer.stdout).strip()


def main(arguments):
    dry = "--dry-run" in arguments
    found = targets(processes())
    if not found:
        print("No ScadaBench processes are running.")
        return 0
    failed = 0
    for row in found:
        label = f"{row['Name']} (PID {row['ProcessId']})"
        if dry:
            print(f"Would stop {label}")
            continue
        reason = stop(row)
        failed += bool(reason)
        print(f"Stopped {label}" if not reason else f"Could not stop {label}: {reason}")
    if failed:
        print("Some processes belong to another user or run as administrator; "
              "run this command from an administrator window to stop them.")
    return 1 if failed else 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
