"""Write generated/helper.json: helper request validation, the runner's commands and results.

Usage (from the worktree root):
  studio/.venv/Scripts/python.exe -I -B studio/engine/internal/netcfg/testdata/make_helper.py
PowerShell is never started: subprocess.run is replaced by a recorder with canned answers.
"""
import hashlib
import json
import subprocess
import sys
import tempfile
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from vector_common import GATEWAY, PROGRAM, b64, error_of, write  # noqa: E402
from studio_app.networking import elevation  # noqa: E402
from studio_app.networking.change_plan import gateway_plan  # noqa: E402
from studio_app.networking.helper import runner  # noqa: E402
from studio_app.networking.helper.validation import validate_request  # noqa: E402

ALIAS = {"kind": "alias", "adapter_guid": "{11111111-1111-1111-1111-111111111111}", "adapter_index": 7,
         "adapter_name": "Lab NIC", "ip": "192.168.50.21", "prefix": 24, "owned": True, "action": "add"}
RULE = {"kind": "firewall", "name": "SubstationStudio-MMS-0123456789ab", "group": "Substation Studio",
        "addresses": ["192.168.50.21"], "ports": [102], "program": PROGRAM, "owned": True, "action": "add"}
BLOCK = {"name": r"TCP Query User{CD50883C-152F-4A3A-864C-00A52154808F}c:\studio\native\studio_gateway.exe",
         "display": "studio_gateway.exe"}
ALLOW, UNBLOCK = gateway_plan(GATEWAY, [], [BLOCK])["items"]
IPS = [None, 3232235777, True, False, -1, 2 ** 32, 1.5, [], {}, "127.0.0.5", "0.0.0.0", "224.1.1.1", "169.254.1.2",
       "1.2.3", "01.2.3.4", "1.2.3.4/24", "255.255.255.255", "10.0.0.1", "it's", "é"]
VARIANTS = (
    [{**ALIAS, "ip": ip} for ip in IPS] + [{k: v for k, v in ALIAS.items() if k != "ip"}]
    + [{**ALIAS, "adapter_guid": g} for g in ("{11111111-1111-1111-1111-11111111111a}", ALIAS["adapter_guid"] + "\n",
                                              ALIAS["adapter_guid"] + "\n\n", None, "Wi-Fi; Remove-Item C:\\")]
    + [{**ALIAS, "adapter_index": i} for i in (0, -1, True, "7", 7.0, 2 ** 70, None)]
    + [{**ALIAS, "prefix": p} for p in (7, 8, 30, 31, True, "24", 24.0)]
    + [{**ALIAS, "owned": o} for o in (False, 0, "", "yes", 1, [], [0], None)] + [{**ALIAS, "kind": "route"}]
    + [{**RULE, "name": n} for n in ("SubstationStudio-MMS-0123456789AB", RULE["name"] + "\n", "Allow everything", 5)]
    + [{**RULE, "group": g} for g in ("Other", None, "substation studio")]
    + [{**RULE, "ports": p} for p in ([], [0], [65536], [True], ["102"], 102, [1, 65535], [102.0])]
    + [{**RULE, "addresses": a} for a in ([], "x", ["1.2.3.4", "127.0.0.1"], [5], None)]
    + [{**RULE, "program": p} for p in (PROGRAM.upper(), PROGRAM.replace("\\", "/"), r"C:\Studio\x\..\native\studio_ied.exe",
                                         r"C:\Windows\System32\cmd.exe", None, "")]
    + [{**ALLOW, "program": p} for p in (GATEWAY.lower(), PROGRAM, r"C:\Windows\System32\svchost.exe")]
    + [{**ALLOW, "name": "Something-else"}, {**ALLOW, "group": "Substation Studio"}]
    + [{**UNBLOCK, "name": n} for n in ("x'; Remove-NetFirewallRule -Name *; '", "TCP Query Usér{ÄB}ç:\\x.exe (1)",
                                         "a" * 400, "a" * 401, "rule\n", "ru\nle", "", "ab\"c", "UDP_x-1.2:3()")]
    + [{**UNBLOCK, "program": PROGRAM}]
    + [{**ALIAS, "kind": k} for k in (None, 5, ["x"], "it's", "Alias")] + [[ALIAS], "alias"]
)


def payload(items, action="apply", top=None):
    data = json.dumps(top if top is not None else {"action": action, "fingerprint": "f", "items": items}).encode()
    return data


def validation_cases():
    raw = [payload([item]) for item in VARIANTS] + [payload([ALIAS, RULE, ALLOW, UNBLOCK])]
    raw += [payload([ALIAS], "format-disk"), payload([ALIAS], "remove"), payload([]), payload([ALIAS] * 64),
            payload([ALIAS] * 65), payload(None, top={"action": "apply", "items": "x"}),
            payload(None, top={"action": ["apply"], "items": []}), payload(None, top=[1]), payload(None, top="x"),
            payload(None, top={"items": [ALIAS]}), b"{bad", b"\xef\xbb\xbf" + payload([ALIAS]), b'{"a": "\xff"}',
            payload([ALIAS]) + b" ", b""]
    checks = [(data, hashlib.sha256(data).hexdigest()) for data in raw]
    good = payload([ALIAS])
    checks += [(good, "0" * 64), (good, hashlib.sha256(good).hexdigest().upper()),
               (good.replace(b"192.168.50.21", b"192.168.50.99"), hashlib.sha256(good).hexdigest())]
    cases = []
    for data, expected in checks:
        try:
            result = {"ok": json.dumps(validate_request(data, expected, PROGRAM))}
        except Exception as error:  # noqa: BLE001 - the class and text are the vector
            result = error_of(error)
        cases.append({"payload": b64(data), "digest": expected, **result})
    return cases


ANSWERS = [(0, "", ""), (1, "", "New-NetIPAddress : The object already exists.\r\n  + CategoryInfo : x\r\n"),
           (1, "out only\nlast out\n", ""), (0, "warning\n", "  \n"), (1, "", "é" * 600 + "\n"),
           (5, "", "line\x1cnext\u2028final  ")]


def runner_cases(folder):
    cases = []
    requests = [("apply", [ALIAS, RULE, ALLOW, UNBLOCK]), ("remove", [ALIAS, RULE, ALLOW, UNBLOCK]),
                ("apply", [{**ALIAS, "ip": 3232235777}]), ("apply", [ALIAS] * 6), ("bad", [ALIAS])]
    for number, (action, items) in enumerate(requests):
        data = payload(None, top=[1]) if action == "bad" else payload(items, action)
        calls, answers = [], list(ANSWERS)

        def fake(argv, capture_output, text, timeout, creationflags):
            calls.append({"argv": argv, "timeout": timeout})
            code, out, err = answers.pop(0)
            return subprocess.CompletedProcess(argv, code, out, err)
        runner.subprocess.run = fake
        request = Path(folder) / f"{action}-{number}.json"
        request.write_bytes(data)
        try:
            result = {"exit": runner.main([str(request), hashlib.sha256(data).hexdigest(), PROGRAM])}
        except Exception as error:  # noqa: BLE001
            result = {"exit": 1, "raised": type(error).__name__}
        result_path = request.with_suffix(".result.json")
        cases.append({"payload": b64(data), "calls": calls, **result,
                      "result": b64(result_path.read_bytes()) if result_path.exists() else None})
    return cases


def main():
    sys.frozen = True
    sys.executable = r"C:\Program Files\ScadaBench\ScadaBench.exe"
    command = elevation.helper_command(Path(r"C:\data dir\network\requests\apply-ab.json"), "f" * 64, PROGRAM)
    arguments = [[], [""], ["a b"], ['say "hi"'], ["back\\slash\\"], ["ends with\\"], ['quote\\"'], ["tab\tx"],
                 ['\\\\server\\share\\x y\\'], ["é ü", "plain", "a\\\\\"b"], ["--network-helper", r"C:\a b\r.json"]]
    with tempfile.TemporaryDirectory() as folder:
        runs = runner_cases(folder)
    write("helper.json", {"validation": validation_cases(), "runner": runs, "helper_command": list(command[:1]) +
                          [command[1]], "list2cmdline": [[a, subprocess.list2cmdline(a)] for a in arguments]})


if __name__ == "__main__":
    main()
