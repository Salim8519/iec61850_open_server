"""Gateway commands (ADR-0057): the native worker has no IEC 61850 client, so every SCADA
command passes the backend checks first. Real IEDs are commanded only while the owner's
"commands to real IEDs" setting is on (default on, 2026-10-17)."""
import re
from pathlib import Path

from studio_app.gateway.commands import refusal

NATIVE = Path(__file__).resolve().parents[2] / "native"
POINT = {"id": "p", "kind": "command_double", "ied": "a", "ref": "BayACTRL/XCBR1.Pos"}


def ied(**changes):
    return {"id": "a", "name": "BayA", "role": "simulated", "state": "running",
            "controls": [{"ref": "BayACTRL/XCBR1.Pos", "status": "BayACTRL/XCBR1.Pos.stVal", "cdc": "DPC"}], **changes}


def test_native_gateway_has_no_iec61850_client():
    sources = "\n".join(path.read_text(encoding="utf-8") for path in (NATIVE / "gateway").glob("*.*pp"))
    for forbidden in ("IedConnection", "ControlObjectClient", "MmsConnection", "iec61850_client.h"):
        assert forbidden not in sources, forbidden
    cmake = (NATIVE / "CMakeLists.txt").read_text(encoding="utf-8")
    link = re.search(r"target_link_libraries\(studio_gateway [^)]*\)", cmake).group(0)
    assert "iec61850" not in link


def test_backend_refuses_unsafe_or_unknown_targets():
    assert refusal(POINT, ied()) == ""
    real = ied(role="real", state="real_live", live={"ip": "10.0.0.5", "port": 102})
    assert refusal(POINT, real) == ""
    assert "switched off for this gateway" in refusal(POINT, real, real_commands=False)
    assert "switched off in Devices" in refusal(POINT, ied(role="off"))
    assert "press Run first" in refusal(POINT, ied(state="stopped"))
    assert "no controllable object" in refusal(dict(POINT, ref="BayACTRL/XSWI9.Pos"), ied())
    assert "not a command" in refusal(dict(POINT, kind="double"), ied())
    assert "not in the gateway point list" in refusal(None, ied())
    assert "not in the project" in refusal(POINT, None)
