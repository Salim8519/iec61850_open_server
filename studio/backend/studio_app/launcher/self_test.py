"""Offline packaged-engine check using temporary engineering data and loopback MMS."""
import json
import io
from pathlib import Path
import socket
import subprocess
import tempfile
import time
from xml.etree import ElementTree as ET
from studio_app.app_factory import create_app
from studio_app.settings import Settings
from studio_app.scl.templates import make_template
from studio_app.scl.importer import import_scl


def fixture():
    namespace = "{http://www.iec.ch/61850/2003/SCL}"
    root = ET.fromstring(make_template("SelfTest"))
    types = root.find(namespace + "DataTypeTemplates")
    for item in types.iter(namespace + "DO"):
        if item.get("name") == "AnIn2":
            item.set("type", "SelfTestArray")
    wrapper = ET.Element(namespace + "DOType", id="SelfTestArray", cdc="MV")
    ET.SubElement(wrapper, namespace + "SDO", name="samples", type="StudioMV", count="3")
    position = next(index for index, item in enumerate(types) if item.tag == namespace + "DAType")
    types.insert(position, wrapper)
    return ET.tostring(root, encoding="utf-8")


def read_native(settings, ied, reference):
    observed = subprocess.run([str(settings.native_executable.with_name("studio_probe.exe")),
                               ied["ip"], str(ied["port"]), "read", reference, "MX"],
                              capture_output=True, text=True, timeout=10,
                              creationflags=subprocess.CREATE_NO_WINDOW)
    assert observed.returncode == 0, observed.stdout + observed.stderr
    return float(json.loads(observed.stdout)["value"])


def check(settings):
    settings.directory.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="studio-check-", dir=settings.directory) as temporary:
        directory = Path(temporary).resolve()
        if not directory.is_relative_to(settings.directory.resolve()):
            raise RuntimeError("Self-test directory escaped the selected data folder")
        local = Settings(directory, settings.native_executable, settings.frontend)
        app = create_app(local, "isolated-self-test")
        runtime = app.extensions["runtime"]
        try:
            client = app.test_client()
            assert client.get("/api/v1/project").status_code == 401
            assert client.post("/api/v1/session", json={"token": "isolated-self-test"}).status_code == 200
            source = fixture()
            response = client.post("/api/v1/scl/import", data={
                "file": (io.BytesIO(source), "self-test.cid"), "selected_names": json.dumps(["SelfTest"])})
            if response.status_code != 200:
                raise RuntimeError(response.get_json())
            ied = app.extensions["projects"].project["ieds"][0]
            assert ied["validation"]["status"] == "valid", ied["validation"]
            with socket.socket() as listener:
                listener.bind(("127.54.0.254", 0))
                port = listener.getsockname()[1]
            ied.update(ip="127.54.0.254", port=port)
            alarm = "SelfTestCTRL/XCBR1.Loc.stVal"
            assert client.put("/api/v1/project/test-panel", json=[{"id": "alarm", "label": "Local mode",
                "ied_id": ied["id"], "ref": alarm}]).status_code == 200
            runtime.start()
            runtime.pulses.start(ied["id"], alarm, True, 100)
            deadline = time.monotonic() + 3
            while runtime.pulses.pending and time.monotonic() < deadline:
                time.sleep(0.02)
            assert runtime.supervisor.read(ied["id"], alarm) is False
            reference = "SelfTestCTRL/GGIO1.AnIn1.mag.f"
            runtime.update(ied["id"], reference, 42.5)
            assert read_native(settings, ied, reference) == 42.5
            array_ref = "SelfTestCTRL/GGIO1.AnIn2.samples(2).mag.f"
            runtime.update(ied["id"], array_ref, 17.25)
            assert read_native(settings, ied, array_ref) == 17.25
            exported = client.post(f"/api/v1/ieds/{ied['id']}/export-scl", json={"overrides": {reference: 9.5}})
            assert exported.status_code == 200, exported.get_json()
            restored_export = import_scl(exported.data, selected_names=["SelfTest"])[0]
            assert next(item for item in restored_export["signals"] if item["ref"] == reference)["value"] == 9.5
            identity = app.extensions["projects"].project["id"]
        finally:
            runtime.close()
        recovered = create_app(local, "isolated-recovery-test")
        try:
            assert recovered.extensions["projects"].project["id"] == identity
            assert recovered.extensions["startup_recovery"]["status"] == "restored"
            assert recovered.extensions["runtime"].state == "stopped"
            assert not recovered.extensions["runtime"].supervisor.workers
            client = recovered.test_client()
            assert client.post("/api/v1/session", json={"token": "isolated-recovery-test"}).status_code == 200
            key = recovered.extensions["projects"].project["ieds"][0]["id"]
            assert recovered.extensions["projects"].project["test_panel"][0]["ref"] == alarm
            assert client.get(f"/api/v1/ieds/{key}/removal-preview").status_code == 200
            deleted = client.delete(f"/api/v1/ieds/{key}", json={"confirm": True})
            assert deleted.status_code == 200, deleted.get_json()
            assert recovered.extensions["projects"].project["ieds"] == []
            assert recovered.extensions["projects"].project["test_panel"] == []
            backup = Path(deleted.get_json()["recovery_file"])
            assert backup.is_relative_to(directory)
            recovered.extensions["projects"].restore(backup.read_bytes())
            assert recovered.extensions["projects"].project["ieds"][0]["id"] == key
            assert len(recovered.extensions["projects"].project["test_panel"]) == 1
        finally:
            recovered.extensions["runtime"].close()
        return {"ok": True, "schema": "SCL2007B4", "mms_value": 42.5,
                "array_value": 17.25, "edited_export_verified": True, "stopped_recovery_verified": True,
                "ied_deletion_verified": True,
                "saved_panel_and_pulse_verified": True,
                "customer_data_used": False, "clean_machine_verified": False}
