"""Start the GOOSE engine and the studio_live decoder the way a Run does (ADR-0073)."""
import os
import time
from pathlib import Path

from studio_app.runtime.worker_process import WorkerProcess

ROOT = Path(__file__).resolve().parents[3]
BIN = Path(os.environ.get("STUDIO_NATIVE_BIN", ROOT / "studio/native/build/Release"))
READY = (BIN / "studio_goose.exe").exists() and (BIN / "studio_live.exe").exists()


def wait_for(events, predicate, seconds):
    deadline = time.monotonic() + seconds
    while time.monotonic() < deadline:
        found = [event for event in list(events) if predicate(event)]
        if found:
            return found
        time.sleep(0.05)
    return []


class EngineRig:
    """studio_goose.exe plus studio_live.exe; workers publish to `interface(ied_id)`."""

    def __init__(self):
        self.engine_events, self.received = [], []
        self.engine = WorkerProcess(BIN / "studio_goose.exe", self.engine_events.append)
        self.listener = WorkerProcess(BIN / "studio_live.exe", self.received.append)
        self.interfaces, self.sources = {}, {}

    def publish(self, ied_id, gocb_refs):
        opened = self.engine.request("open", ieds=[{"id": ied_id, "gocb_refs": list(gocb_refs)}])
        mac = opened["sources"][ied_id]
        self.sources[ied_id] = mac
        self.interfaces[ied_id] = f"udp:{opened['port']}:{mac.replace('-', '')}"
        return self.interfaces[ied_id]

    def listen(self, subscriptions):
        inbox = self.listener.request("goose_start", timeout=30, subscriptions=subscriptions)["inbox_port"]
        self.engine.request("deliver", port=inbox)

    def close(self):
        self.listener.close()
        self.engine.close()


def subscription(control, **changes):
    return {"key": control["gocb_ref"], "gocb_ref": control["gocb_ref"], "app_id": control["app_id"],
            "mac": control["mac"], "go_id": control["go_id"], "dataset_ref": control["dataset_ref"],
            "conf_rev": control["conf_rev"], **changes}
