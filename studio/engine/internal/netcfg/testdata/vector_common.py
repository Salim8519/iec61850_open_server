"""Shared setup for the netcfg vector generators: import paths and JSON helpers.

The generators import the frozen Python engine from this worktree (studio/backend), replace
every Windows call with a fake and record inputs and Python's exact outputs.
"""
import base64
import json
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
STUDIO = HERE.parents[3]
sys.path.insert(0, str(STUDIO / "backend"))

from flask import Flask, jsonify  # noqa: E402

APP = Flask("netcfg-vectors")
PROGRAM = r"C:\Studio\native\studio_ied.exe"
GATEWAY = r"C:\Studio\native\studio_gateway.exe"
GUID = "{11111111-1111-1111-1111-111111111111}"
OTHER_GUID = "{22222222-2222-2222-2222-222222222222}"


def b64(data):
    return base64.b64encode(data).decode("ascii")


def dumps(value):
    return json.dumps(value)


def flask_body(value):
    with APP.app_context():
        return jsonify(value).get_data(as_text=True)


def adapter(guid=GUID, name="Lab NIC", index=7, addresses=(("192.168.50.10", 24),), **extra):
    """An adapter in the normalized shape (as the integration test fakes write it)."""
    item = {"guid": guid, "name": name, "description": extra.get("description", "USB Ethernet"),
            "status": extra.get("status", "Up"), "index": index, "mac": "", "speed": "1 Gbps",
            "kind": extra.get("kind", "ethernet"), "default_route": False,
            "addresses": [{"ip": ip, "prefix": prefix, "skip_as_source": False, "origin": "Manual"}
                          for ip, prefix in addresses]}
    return item


def error_of(error):
    """Describe a raised exception: class, text, StudioError code and details."""
    return {"error": type(error).__name__, "message": str(error), "code": getattr(error, "code", None),
            "details": getattr(error, "details", None)}


def outcome(action, encode=dumps):
    try:
        return {"ok": encode(action())}
    except Exception as error:  # noqa: BLE001 - the class and text are the vector
        return error_of(error)


def write(name, value):
    (HERE / "generated").mkdir(exist_ok=True)
    (HERE / "generated" / name).write_text(json.dumps(value, indent=0) + "\n", encoding="utf-8", newline="\n")
