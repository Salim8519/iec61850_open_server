"""Stand-in real IED for the Go live-session evidence test, plus an independent client.

Starts studio_ied.exe (template "ReportTest") on 127.64.0.31:20131, writes the imported
IED model to the JSON file named on the command line and prints "ready". Then it answers
one JSON command per stdin line with one JSON line:
  {"op": "update", "ref": r, "value": v}        write a value inside the simulated device
  {"op": "read", "refs": [{"ref", "fc"}]}       read through a separate studio_live client
  {"op": "rcb_status", "rcbs": [...]}           report-block state seen by that client
Usage: python stand_in.py <model.json> (studio venv; native binaries in the worktree build).
"""
import json
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[6]
sys.path.insert(0, str(ROOT / "studio" / "backend"))
BIN = ROOT / "studio" / "native" / "build" / "Release"
HOST, PORT = "127.64.0.31", 20131

from studio_app.runtime.worker_process import WorkerProcess  # noqa: E402
from studio_app.scl.importer import import_scl  # noqa: E402
from studio_app.scl.templates import make_template  # noqa: E402


def main():
    model = import_scl(make_template("ReportTest"))[0]
    config = Path(tempfile.mkdtemp(prefix="stand-in-")) / "device.cfg"
    config.write_text(model["cfg"], encoding="utf-8")
    device = WorkerProcess(BIN / "studio_ied.exe", lambda event: None)
    client = None
    try:
        device.request("start", cfg=str(config), ip=HOST, port=PORT, controls=model["controls"], edition=2)
        Path(sys.argv[1]).write_text(json.dumps({key: value for key, value in model.items()
                                                 if key not in ("source", "cfg")}), encoding="utf-8")
        print(json.dumps({"ready": True, "host": HOST, "port": PORT}), flush=True)
        for line in sys.stdin:
            command = json.loads(line)
            op = command.pop("op")
            try:
                if op == "update":
                    result = device.request("update", **command)
                else:
                    if client is None:
                        client = WorkerProcess(BIN / "studio_live.exe", lambda event: None)
                        client.request("connect", timeout=15, ip=HOST, port=PORT)
                    result = client.request(op, timeout=30, **command)
                print(json.dumps({"ok": True, "result": result}), flush=True)
            except Exception as error:  # reported to the Go test, which fails on it
                print(json.dumps({"ok": False, "error": str(error)}), flush=True)
    finally:
        if client is not None:
            client.close()
        device.close()


main()
