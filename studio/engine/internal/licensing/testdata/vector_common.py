"""Shared setup for the licensing vector generators: import paths, signer, JSON helpers.

The generators import the frozen Python engine from this worktree (studio/backend) and
the test signer (studio/tests/unit/license_signer.py, test seed only, never a release key).
"""
import base64
import json
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
STUDIO = HERE.parents[3]
sys.path.insert(0, str(STUDIO / "backend"))
sys.path.insert(0, str(STUDIO / "tests" / "unit"))

import license_signer  # noqa: E402  (test-only signer)
from flask import Flask, jsonify  # noqa: E402

APP = Flask("licensing-vectors")
OTHER_SEED = bytes.fromhex("17" * 32)


def b64(data):
    return base64.b64encode(data).decode("ascii")


def compact(value):
    return json.dumps(value, separators=(",", ":"))


def flask_body(value):
    with APP.app_context():
        return jsonify(value).get_data(as_text=True)


def signed(seed=license_signer.TEST_SEED, **fields):
    """license.cert JSON text signed with the test key (or another seed)."""
    return license_signer.bundle(seed, **fields)


def signed_raw(payload_text, seed=license_signer.TEST_SEED):
    signature = base64.urlsafe_b64encode(license_signer.sign(payload_text.encode("utf-8"), seed))
    return {"payload": payload_text, "signature": signature.decode().rstrip("=")}


def write(name, value):
    (HERE / "generated").mkdir(exist_ok=True)
    (HERE / "generated" / name).write_text(json.dumps(value, indent=0) + "\n", encoding="utf-8", newline="\n")


def outcome(action):
    """Run action and describe its result or the exception class and text."""
    try:
        return {"ok": action()}
    except Exception as error:  # noqa: BLE001 - the class name is the vector
        return {"error": type(error).__name__, "message": str(error), "code": getattr(error, "code", None)}
