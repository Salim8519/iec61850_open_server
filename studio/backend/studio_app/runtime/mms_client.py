"""Invoke the bundled MMS client for browser-requested control operations."""
import json
import os
import subprocess


def operate(executable, ied, reference, value):
    if not isinstance(value, bool):
        raise ValueError("Control value must be boolean")
    if reference not in {control["ref"] for control in ied["controls"]}:
        raise ValueError("Unknown control object")
    if ied["state"] != "running":
        raise ValueError("IED must be running to operate equipment")
    client = executable.with_name("studio_probe.exe")
    if not client.exists():
        raise RuntimeError("Bundled MMS client is unavailable")
    completed = subprocess.run([str(client), ied["ip"], str(ied["port"]), "control", reference,
                                "true" if value else "false"], capture_output=True, text=True,
                               timeout=15, creationflags=subprocess.CREATE_NO_WINDOW if os.name == "nt" else 0)
    if completed.returncode != 0:
        raise RuntimeError("MMS control failed or command termination was not received")
    return json.loads(completed.stdout)
