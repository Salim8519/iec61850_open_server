"""Run inside the elevated helper process: validate, apply, and write results."""
import json
import os
import subprocess
from pathlib import Path
from studio_app.networking.change_plan import apply_command, remove_command
from studio_app.networking.journal import item_key
from .validation import HelperRefused, validate_request


def _run(command):
    flags = subprocess.CREATE_NO_WINDOW if os.name == "nt" else 0
    completed = subprocess.run(["powershell.exe", "-NoProfile", "-NonInteractive", "-Command",
                                "$ErrorActionPreference='Stop'; " + command + " | Out-Null"],
                               capture_output=True, text=True, timeout=60, creationflags=flags)
    message = (completed.stderr or completed.stdout).strip().splitlines()
    return completed.returncode == 0, (message[-1] if message else "")


def execute(request, run=_run):
    results = []
    for item in request["items"]:
        command = apply_command(item) if request["action"] == "apply" else remove_command(item)
        ok, message = run(command)
        results.append({"key": item_key(item), "ok": ok, "message": message[:500]})
    return results


def main(argv):
    """argv: request path, expected SHA-256, worker program path."""
    request_path, expected, program = Path(argv[0]), argv[1], argv[2]
    result_path = request_path.with_suffix(".result.json")
    try:
        request = validate_request(request_path.read_bytes(), expected, program)
        outcome = {"refused": False, "results": execute(request)}
    except HelperRefused as error:
        outcome = {"refused": True, "reason": str(error), "results": []}
    result_path.write_text(json.dumps(outcome), encoding="utf-8")
    return 0 if not outcome["refused"] else 2
