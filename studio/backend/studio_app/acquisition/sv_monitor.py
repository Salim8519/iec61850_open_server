"""Listen to Sampled Values on the lab adapter and keep a supervised view of each stream."""
import threading
from datetime import datetime, timezone
from studio_app.domain.sv_supervision import assess, native_stream, watch_list
from studio_app.runtime.worker_process import WorkerProcess

UNKNOWN_FIELDS = ("app_id", "mac", "src_mac", "vlan_id", "sv_id", "simulate")
RECEIVED = ("asdus", "rate", "sv_id", "conf_rev", "smp_rate", "smp_mod", "smp_synch", "lost", "restarts",
            "invalid_quality", "data_size", "simulate", "values", "raw")


class SvMonitor:
    def __init__(self, manager):
        self.manager = manager
        self.process = None
        self.watches, self.not_supervised = {}, []
        self.status = {"state": "off", "message": "", "unknown": []}
        self.lock = threading.Lock()

    def start(self, project, npcap_installed):
        network = project["network"]
        supervised, extra = watch_list(project["ieds"], network.get("nominal_hz", 50))
        self.watches = {watch["key"]: watch for watch in supervised}
        self.not_supervised = extra
        if not self.watches:
            self.status = {"state": "nothing_to_watch", "unknown": [],
                           "message": "No IED in this project sends Sampled Values."}
            return
        if network["mode"] != "lab" or not network.get("adapter_guid"):
            self.status = {"state": "off", "unknown": [],
                           "message": "Sampled Values need a network adapter. Set up the lab network to use them."}
            return
        if not npcap_installed:
            self.status = {"state": "needs_npcap", "unknown": [],
                           "message": "Install the free Npcap driver to receive Sampled Values."}
            return
        for watch in self.watches.values():
            watch.update(state="waiting", problems=[])
        self.process = WorkerProcess(self.manager.executable, self._event)
        self.process.request("sv_start", timeout=30, adapter_guid=network["adapter_guid"],
                             streams=[native_stream(watch) for watch in self.watches.values()])
        self.status = {"state": "listening", "unknown": [], "nominal_hz": network.get("nominal_hz", 50),
                       "message": f"Listening for Sampled Values on {network.get('adapter_label') or 'the lab adapter'}"}

    def _event(self, event):
        kind = event.get("event")
        if kind == "sv_rx":
            watch = self.watches.get(event.get("key"))
            if watch is None:
                return
            with self.lock:
                previous = {key: watch.get(f"received_{key}") for key in ("lost", "invalid_quality")}
                verdict = assess(watch, event, previous)
                changed = verdict != {"state": watch["state"], "problems": watch["problems"]}
                watch.update(verdict)
                watch.update({f"received_{key}": event.get(key) for key in RECEIVED})
                watch["observed_at"] = datetime.now(timezone.utc).isoformat(timespec="seconds")
            if changed:
                self.manager.coordinator.emit("sv_status", key=watch["key"], **verdict)
        elif kind == "sv_unknown":
            with self.lock:
                self.status["unknown"].append({key: event.get(key) for key in UNKNOWN_FIELDS})
        elif kind == "sv_unknown_cap":
            with self.lock:
                self.status["unknown_limit"] = event.get("limit")
        elif kind == "worker_failed":
            self.status.update(state="failed", message="The Sampled Values listener stopped unexpectedly")

    def snapshot(self):
        with self.lock:
            return {**self.status, "watches": [dict(watch) for watch in self.watches.values()],
                    "not_supervised": [{"key": item["key"], "publisher": item["publisher"]}
                                       for item in self.not_supervised]}

    def stop(self):
        process, self.process = self.process, None
        if process is not None:
            process.close()
        self.status = {"state": "off", "message": "", "unknown": []}
