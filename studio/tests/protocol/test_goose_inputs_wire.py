"""IED to IED: a simulated subscriber's input follows the publisher's GOOSE (GOOSE-16).

Two studio_ied workers publish through the app's GOOSE engine to studio_live, exactly as
a Run does (ADR-0073); no adapter or Npcap is needed. The subscriber's value is checked
with studio_probe, an independent MMS client.
"""
import json
import os
import subprocess
import time
from pathlib import Path

import pytest
from studio_app.acquisition.goose_inputs import GooseInputs
from studio_app.domain.goose_supervision import watch_list
from studio_app.runtime.worker_process import WorkerProcess
from studio_app.scl.importer import import_scl
from goose_engine_rig import READY, EngineRig

ROOT = Path(__file__).resolve().parents[3]
BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", ROOT / "studio/native/build/Release"))
pytestmark = pytest.mark.skipif(not READY, reason="Native build and GOOSE engine required")
WRONG = 'intAddr="Tr" desc="Trip"'
TARGET = "IED1_XCBRGenericIO/CILO1.EnaCls.stVal"


def station():
    text = (ROOT / "scd/open_substation.scd").read_text(encoding="utf-8")
    text = text.replace(WRONG, 'intAddr="CILO1.EnaCls.stVal" desc="Trip"').replace(
        'srcLNClass="LLN0" srcCBName="GoCB"', 'srcLNClass="LLN0" srcCBName="gcbEvents"')
    ieds = {ied["name"]: ied for ied in import_scl(text.encode(), "inputs.scd", ["IED1_XCBR", "IED2_PTOC"])}
    for index, ied in enumerate(ieds.values()):
        ied.update(id=ied["name"], role="simulated", ip=f"127.50.0.{240 + index}", port=18140 + index)
    return ieds


class Coordinator:
    def __init__(self, workers, ieds):
        self.workers, self.emitted = workers, []
        self.sample_writer = self
        self.locks = {name: None for name in workers}
        self.projects = self
        self.ieds = ieds

    def apply(self, ied_id, items, source):
        assert source == "goose_input"
        self.workers[ied_id].request("update_batch", updates=[{"ref": ref, "value": value} for ref, value, _ in items])

    def emit(self, kind, **payload):
        self.emitted.append((kind, payload))

    def find_ied(self, ied_id):
        return self.ieds[ied_id]


def read_target(ied):
    completed = subprocess.run([str(BIN / "studio_probe.exe"), ied["ip"], str(ied["port"]), "read",
                                TARGET, "ST"], capture_output=True, text=True, timeout=20)
    return json.loads(completed.stdout)["value"]


def wait_for(items, predicate, seconds):
    deadline = time.monotonic() + seconds
    while time.monotonic() < deadline:
        found = [item for item in list(items) if predicate(item)]
        if found:
            return found
        time.sleep(0.05)
    return []


def run(tmp_path, conf_rev_shift=0):
    ieds = station()
    workers = {}
    rig = EngineRig()
    received = rig.received
    try:
        bus = rig.publish("IED2_PTOC", ["IED2_PTOCGenericIO/LLN0$GO$gcbEvents"])
        for name, ied in ieds.items():
            config = tmp_path / f"{name}.cfg"
            config.write_text(ied["cfg"], encoding="utf-8")
            workers[name] = WorkerProcess(BIN / "studio_ied.exe", lambda event: None)
            workers[name].request("start", cfg=str(config), ip=ied["ip"], port=ied["port"],
                                  controls=ied["controls"], edition=2,
                                  goose={"publish": name == "IED2_PTOC", "interface": bus})
        watches = watch_list(list(ieds.values()))
        watch = watches["IED2_PTOCGenericIO/LLN0$GO$gcbEvents"]
        coordinator = Coordinator(workers, ieds)
        inputs = GooseInputs(type("Manager", (), {"coordinator": coordinator})())
        inputs.prepare(list(ieds.values()), watches)
        assert watch["subscribers"][0]["applied_inputs"] == 1
        rig.listen([{**{key: watch[key] for key in ("key", "gocb_ref", "app_id", "mac", "go_id", "dataset_ref")},
                     "conf_rev": watch["conf_rev"] + conf_rev_shift}])
        seen = 0

        def pump():
            nonlocal seen
            for event in received[seen:]:
                if event.get("event") == "goose_rx":
                    inputs.apply(watch, event)
            seen = len(received)
        deadline = time.monotonic() + 40
        while time.monotonic() < deadline and not coordinator.emitted:
            pump(); time.sleep(0.05)
        assert coordinator.emitted, "no GOOSE frame reached the listener"
        before = read_target(ieds["IED1_XCBR"])
        workers["IED2_PTOC"].request("update", ref="IED2_PTOCGenericIO/PTRC1.Tr.general", value=True)
        deadline = time.monotonic() + 10
        while time.monotonic() < deadline and len(coordinator.emitted) < 2:
            pump(); time.sleep(0.05)
        time.sleep(0.3)
        return coordinator.emitted, before, read_target(ieds["IED1_XCBR"])
    finally:
        for worker in workers.values():
            worker.close()
        rig.close()


def test_subscriber_input_follows_the_publisher(tmp_path):
    emitted, before, after = run(tmp_path)
    applied = [payload for kind, payload in emitted if kind == "goose_input" and payload["applied"]]
    assert len(applied) >= 2 and applied[-1]["applied"] == [TARGET]
    assert before == "false" and after == "true"


def test_conf_rev_mismatch_never_changes_the_subscriber(tmp_path):
    emitted, before, after = run(tmp_path, conf_rev_shift=1)
    assert {kind for kind, _ in emitted} == {"goose_input_refused"}
    assert "conf_rev_mismatch" in emitted[0][1]["reason"]
    assert before == after == "false"
