"""Run in lab mode when the Windows address lookup fails, on the frozen Python app (gap 7).

Usage: studio/.venv/Scripts/python.exe -I lab_preflight_python.py OUT.json

create_app on a temp folder (empty stand-in studio_ied.exe: the preflight runs before any
worker starts), one template IED, network mode "lab"; ip_helper.addresses_on raises
OSError(87, "GetAdaptersAddresses failed") as a failed GetAdaptersAddresses does. Reports
POST /api/v1/runtime/start (status, body) and the "runtime" events.
"""
import json
import os
import sys
import tempfile
from pathlib import Path

os.environ["SCADABENCH_UPDATE_CHECK"] = "off"
ROOT = Path(__file__).resolve().parents[5]
sys.path.insert(0, str(ROOT / "studio" / "backend"))

from studio_app.app_factory import create_app  # noqa: E402
from studio_app.licensing.service import LicenseService  # noqa: E402
from studio_app.settings import Settings  # noqa: E402
from studio_app.updates.service import UpdateService  # noqa: E402

GUID = "{0F1E2D3C-4B5A-6978-8796-A5B4C3D2E1F0}"


def failing_lookup(adapter_guid):
    raise OSError(87, "GetAdaptersAddresses failed")


def main():
    out = Path(sys.argv[1])
    with tempfile.TemporaryDirectory() as directory:
        folder = Path(directory)
        native = folder / "native"
        native.mkdir()
        (native / "studio_ied.exe").write_bytes(b"")
        data = folder / "data"
        app = create_app(Settings(data, native / "studio_ied.exe", folder / "frontend"), "lab-token",
                         license_service=LicenseService(folder / "home", data, False),
                         update_service=UpdateService(data, lambda: None))
        app.extensions["startup"].wait(30)
        projects, runtime = app.extensions["projects"], app.extensions["runtime"]
        projects.add_template("LabBay")
        projects.project["network"].update(mode="lab", adapter_guid=GUID)
        runtime.supervisor.lab_addresses = failing_lookup
        client = app.test_client()
        assert client.post("/api/v1/session", json={"token": "lab-token"}).status_code == 200
        cursor = app.extensions["events"].after(0)[-1]["id"] if app.extensions["events"].after(0) else 0
        answer = client.post("/api/v1/runtime/start")
        events = [{key: value for key, value in event.items() if key not in {"id", "timestamp"}}
                  for event in app.extensions["events"].after(cursor) if event["kind"] == "runtime"]
        result = {"status": answer.status_code, "body": answer.get_json(), "events": events,
                  "state": runtime.state}
        runtime.close()
    out.write_text(json.dumps(result, sort_keys=True), encoding="utf-8")


if __name__ == "__main__":
    main()
