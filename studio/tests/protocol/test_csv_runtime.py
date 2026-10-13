import json
import os
from pathlib import Path
import subprocess
import time

import pytest
from studio_app.projects.service import ProjectService
from studio_app.runtime.event_bus import EventBus
from studio_app.simulation.coordinator import Coordinator

ROOT = Path(__file__).resolve().parents[3]
BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", ROOT / "studio/native/build/Release"))
pytestmark = pytest.mark.skipif(not (BIN / "studio_ied.exe").exists(), reason="Native build required")


def test_csv_trace_reaches_independent_mms_client_and_pauses(tmp_path):
    projects = ProjectService(tmp_path)
    ied = projects.add_template("CSV")[0]
    ied.update(ip="127.50.0.218", port=18107)
    reference = "CSVCTRL/GGIO1.AnIn1.mag.f"
    bus = EventBus()
    runtime = Coordinator(projects, bus, BIN / "studio_ied.exe", tmp_path)
    try:
        runtime.configure_source(ied["id"], reference, {"mode": "csv", "time_column": "time",
                                 "value_column": "value", "csv_text": "time,value\n0,12.5\n0.4,42.5\n"})
        assert all("csv_text" not in event.get("spec", {}) for event in bus.after(0))
        runtime.start()
        runtime.pause()
        before = runtime.elapsed
        time.sleep(0.5)
        assert runtime.elapsed == before
        runtime.pause()
        deadline = time.monotonic() + 3
        while runtime.supervisor.read(ied["id"], reference) != 42.5 and time.monotonic() < deadline:
            time.sleep(0.05)
        observed = subprocess.run([str(BIN / "studio_probe.exe"), ied["ip"], str(ied["port"]),
                                   "read", reference, "MX"], capture_output=True, text=True, timeout=10)
        assert observed.returncode == 0, observed.stdout + observed.stderr
        assert float(json.loads(observed.stdout)["value"]) == 42.5
    finally:
        runtime.close()
