"""Show what the frozen Python app keeps of a runtime value (speed lane I2, gap 4).

Usage: studio/.venv/Scripts/python.exe -I values_restart_python.py NATIVE_DIR PORT_BASE OUT.json

Runs the runtime parity project (IEDs at PORT_BASE+1, +2) on the real Coordinator over a
ProjectService folder, writes BayA AnIn2 = 42.5, stops, runs again, stops, saves like
POST /project/save, closes, then recovers the folder like a restart (recover_startup).
Reports the value GET /ieds/{id}/signals shows at each stage (the project's signal dict,
which the coordinator's run index points into) and the IED keys of the saved archive.
"""
import json
import sys
import tempfile
import zipfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[5]
sys.path.insert(0, str(ROOT / "studio" / "backend"))

from studio_app.projects.service import ProjectService  # noqa: E402
from studio_app.projects.startup_recovery import recover_startup  # noqa: E402
from studio_app.runtime.event_bus import EventBus  # noqa: E402
from studio_app.simulation.coordinator import Coordinator  # noqa: E402

REF = "BayACTRL/GGIO1.AnIn2.mag.f"


def shown(projects, ied_id):
    """The "value" GET /ieds/{id}/signals?ref=REF answers (api/signals.list_signals)."""
    ied = projects.find_ied(ied_id)
    return next(signal["value"] for signal in ied["signals"] if signal["ref"] == REF)


def main():
    native, port_base, out = Path(sys.argv[1]), int(sys.argv[2]), Path(sys.argv[3])
    project = json.loads((Path(__file__).resolve().parents[2] / "runtime" / "testdata" / "project.json")
                         .read_text(encoding="utf-8"))
    for index, ied in enumerate(project["ieds"]):
        ied["port"] = port_base + index + 1
    ied_id = project["ieds"][0]["id"]
    result = {}
    with tempfile.TemporaryDirectory() as directory:
        folder = Path(directory)
        projects = ProjectService(folder, lambda revision, source: None)
        projects.project = project
        result["initial"] = shown(projects, ied_id)
        c = Coordinator(projects, EventBus(capacity=100000), native / "studio_ied.exe", folder)
        try:
            c.start()
            c.update(ied_id, REF, 42.5)
            c.stop()
            result["after_stop"] = shown(projects, ied_id)
            c.start()
            result["second_run"] = c.signal(ied_id, REF)["value"]
            c.stop()
            projects.save()  # POST /project/save (any later edit saves the same way)
        finally:
            c.close()
        with zipfile.ZipFile(folder / "autosave.subsim") as archive:
            saved = json.loads(archive.read("project.json"))
        result["archive_ied_keys"] = sorted({key for ied in saved["ieds"] for key in ied})
        restarted = ProjectService(folder, lambda revision, source: None)
        result["recovery"] = (recover_startup(restarted) or {}).get("status")
        result["after_restart"] = shown(restarted, ied_id)
    out.write_text(json.dumps(result, sort_keys=True), encoding="utf-8")


if __name__ == "__main__":
    main()
