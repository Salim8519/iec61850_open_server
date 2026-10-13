"""Independent MMS evidence for unmapped test signals and guarded pulse returns."""
import json
import os
from pathlib import Path
import subprocess
import time
import pytest
from studio_app.app_factory import create_app
from studio_app.settings import Settings
from studio_app.scl.templates import make_template
from studio_app.runtime.mms_client import operate

ROOT = Path(__file__).resolve().parents[3]
BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", ROOT / "studio/native/build/Release"))
pytestmark = pytest.mark.skipif(not (BIN / "studio_ied.exe").exists(), reason="Native build required")


@pytest.fixture
def lab(tmp_path):
    app = create_app(Settings(tmp_path, BIN / "studio_ied.exe", tmp_path), "pulse-test")
    source = make_template("Pulse").replace(b'</DataSet>', b'<FCDA ldInst="CTRL" lnClass="XCBR" '
        b'lnInst="1" doName="Loc" daName="stVal" fc="ST" /></DataSet>')
    projects = app.extensions["projects"]
    ied = projects.add_scl(source, "pulse.cid")[0]
    ied.update(ip="127.50.0.211", port=18112)
    reference = "PulseCTRL/XCBR1.Loc.stVal"
    projects.test_panel([{"id": "local", "label": "Local mode", "ied_id": ied["id"], "ref": reference,
                         "report_ref": "PulseCTRL/LLN0.BR.Buffered"}])
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "pulse-test"})
    runtime = app.extensions["runtime"]
    runtime.start()
    try:
        yield app, runtime, ied, client, reference
    finally:
        runtime.close()


def wait_until(predicate, timeout=3):
    deadline = time.monotonic() + timeout
    while not predicate() and time.monotonic() < deadline:
        time.sleep(0.02)
    assert predicate()


@pytest.mark.parametrize("rcb", ["RP.EventsRCB", "BR.Buffered"])
def test_unmapped_pulse_is_readable_and_reported_while_paused(lab, rcb):
    app, runtime, ied, client, reference = lab
    runtime.pause()
    receiver = subprocess.Popen([str(BIN / "studio_probe.exe"), ied["ip"], str(ied["port"]),
        "reports", "PulseCTRL/LLN0." + rcb], stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    try:
        assert json.loads(receiver.stdout.readline())["ready"]
        response = client.post("/api/v1/signals/pulse", json={"ied_id": ied["id"], "ref": reference,
                                                               "duration_ms": 700, "active_value": True})
        assert response.status_code == 200, response.json
        observed = subprocess.run([str(BIN / "studio_probe.exe"), ied["ip"], str(ied["port"]),
            "read", reference, "ST"], capture_output=True, text=True, timeout=10)
        assert observed.returncode == 0, observed.stderr
        assert json.loads(observed.stdout)["value"] in {"true", "1"}
        wait_until(lambda: runtime.supervisor.read(ied["id"], reference) is False)
        output, errors = receiver.communicate(timeout=10)
        assert receiver.returncode == 0, errors
        entries = [entry for report in json.loads(output)["reports"] for entry in report["entries"]]
        changed = [entry["value"] for entry in entries if entry["index"] == 5 and entry["reason"] & 1]
        assert "true" in changed and "false" in changed
        assert app.extensions["projects"].project["diagram"]["nodes"] == []
    finally:
        if receiver.poll() is None:
            receiver.terminate()
        receiver.communicate(timeout=5)


def test_newer_manual_or_native_same_value_write_wins(lab):
    app, runtime, ied, client, reference = lab
    runtime.pulses.start(ied["id"], reference, True, 150)
    runtime.update(ied["id"], reference, True)
    assert not runtime.pulses.pending
    time.sleep(0.2)
    assert runtime.supervisor.read(ied["id"], reference) is True
    runtime.update(ied["id"], reference, False)
    runtime.pulses.start(ied["id"], reference, True, 150)
    runtime.supervisor.update(ied["id"], reference, True)
    wait_until(lambda: any(event["kind"] == "pulse_failed" for event in app.extensions["events"].after(0)))
    assert runtime.supervisor.read(ied["id"], reference) is True


def test_mms_control_invalidates_native_guard(lab):
    _, runtime, ied, _, _ = lab
    ref = "PulseCTRL/XCBR1.Pos.stVal"
    before = runtime.supervisor.read_versioned(ied["id"], ref)
    operate(BIN / "studio_ied.exe", ied, "PulseCTRL/XCBR1.Pos", True)
    with pytest.raises(RuntimeError, match="stale pulse"):
        runtime.supervisor.update_guarded(ied["id"], [{"ref": ref, "value": 1}],
                                           {"ref": ref, "revision": before["revision"]})
    assert runtime.supervisor.read(ied["id"], ref) == 2


def test_stop_cancels_and_resume_source_releases_override(lab):
    _, runtime, ied, client, reference = lab
    analog = "PulseCTRL/GGIO1.AnIn1.mag.f"
    runtime.configure_source(ied["id"], analog, {"mode": "constant", "offset": 5})
    runtime.update(ied["id"], analog, 8)
    assert runtime.sources[(ied["id"], analog)]["overridden"]
    response = client.post("/api/v1/signals/release-override", json={"ied_id": ied["id"], "ref": analog})
    assert response.status_code == 200
    wait_until(lambda: runtime.supervisor.read(ied["id"], analog) == 5)
    runtime.pulses.start(ied["id"], reference, True, 60000)
    runtime.stop()
    assert not runtime.pulses.pending
    response = client.post("/api/v1/signals/pulse", json={"ied_id": ied["id"], "ref": reference})
    assert response.status_code == 400
