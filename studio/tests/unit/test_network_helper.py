"""The elevated helper refuses anything Studio did not exactly prepare (NET-09)."""
import hashlib
import json

import pytest
from studio_app.networking.helper.runner import execute, main
from studio_app.networking.helper.validation import HelperRefused, validate_request

PROGRAM = r"C:\Studio\native\studio_ied.exe"
ALIAS = {"kind": "alias", "adapter_guid": "{11111111-1111-1111-1111-111111111111}", "adapter_index": 7,
         "adapter_name": "Lab NIC", "ip": "192.168.50.21", "prefix": 24, "owned": True, "action": "add"}
RULE = {"kind": "firewall", "name": "SubstationStudio-MMS-0123456789ab", "group": "Substation Studio",
        "addresses": ["192.168.50.21"], "ports": [102], "program": PROGRAM, "owned": True, "action": "add"}


def payload(items, action="apply"):
    data = json.dumps({"action": action, "fingerprint": "f", "items": items}).encode()
    return data, hashlib.sha256(data).hexdigest()


def test_rejects_tampered_plan(tmp_path):
    data, digest = payload([ALIAS])
    with pytest.raises(HelperRefused, match="changed"):
        validate_request(data.replace(b"192.168.50.21", b"192.168.50.99"), digest, PROGRAM)
    request = tmp_path / "apply.json"
    request.write_bytes(data + b" ")
    assert main([str(request), digest, PROGRAM]) == 2
    assert json.loads(request.with_suffix(".result.json").read_text())["refused"] is True


def test_rejects_items_outside_allow_list():
    bad = [
        {**ALIAS, "ip": "127.0.0.5"}, {**ALIAS, "ip": "169.254.1.2"}, {**ALIAS, "prefix": 4},
        {**ALIAS, "owned": False}, {**ALIAS, "kind": "route"}, {**RULE, "group": "Other"},
        {**RULE, "name": "Allow everything"}, {**RULE, "program": r"C:\Windows\System32\cmd.exe"},
        {**RULE, "ports": [0]}, {**ALIAS, "adapter_guid": "Wi-Fi; Remove-Item C:\\"},
    ]
    for item in bad:
        data, digest = payload([item])
        with pytest.raises(HelperRefused):
            validate_request(data, digest, PROGRAM)
    data, digest = payload([ALIAS], action="format-disk")
    with pytest.raises(HelperRefused):
        validate_request(data, digest, PROGRAM)


def test_allowed_request_runs_each_item_once():
    data, digest = payload([ALIAS, RULE])
    request = validate_request(data, digest, PROGRAM)
    commands = []
    results = execute(request, run=lambda command: (commands.append(command), (True, ""))[1])
    assert len(commands) == 2 and all(entry["ok"] for entry in results)
    assert commands[0].startswith("New-NetIPAddress") and commands[1].startswith("New-NetFirewallRule")
