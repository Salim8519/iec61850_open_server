"""Observe GOOSE on the lab adapter and route it to supervision and live values."""
import threading
from datetime import datetime, timezone
from studio_app.domain.goose_supervision import dataset_values, lgos_updates, watch_list
from studio_app.runtime.worker_process import WorkerProcess
from .goose_inputs import GooseInputs

UNKNOWN_FIELDS = ("gocb_ref", "app_id", "mac", "src_mac", "vlan_id", "dataset", "go_id", "conf_rev", "st_num", "test")


class GooseMonitor:
    def __init__(self, manager):
        self.manager = manager
        self.process = None
        self.watches = {}
        self.status = {"state": "off", "message": "", "unknown": []}
        self.lock = threading.Lock()
        self.inputs = GooseInputs(manager)

    def start(self, ieds, network, npcap_installed):
        self.watches = watch_list(ieds)
        for watch in self.watches.values():
            watch.update(state="stopped", problem="", frames=0)
        if network["mode"] != "lab" or not network.get("adapter_guid"):
            self.status = {"state": "off", "unknown": [],
                           "message": "GOOSE needs a network adapter. Set up the lab network to use it."}
            return
        if not npcap_installed:
            self.status = {"state": "needs_npcap", "unknown": [],
                           "message": "Install the free Npcap driver to send and receive GOOSE."}
            return
        if not self.watches:
            self.status = {"state": "nothing_to_watch", "unknown": [],
                           "message": "No IED in this project publishes or subscribes to GOOSE."}
            return
        for watch in self.watches.values():
            watch.update(state="waiting", problem="", frames=0, st_num=None, sq_num=None)
        self.inputs.prepare(ieds, self.watches)
        self.process = WorkerProcess(self.manager.executable, self._event)
        subscriptions = [{key: watch[key] for key in ("key", "gocb_ref", "app_id", "mac", "go_id",
                                                       "dataset_ref", "conf_rev")} for watch in self.watches.values()]
        self.process.request("goose_start", timeout=30, adapter_guid=network["adapter_guid"],
                             subscriptions=subscriptions)
        self.status = {"state": "listening", "unknown": [],
                       "message": f"Listening for GOOSE on {network.get('adapter_label') or 'the lab adapter'}"}

    def _event(self, event):
        kind = event.get("event")
        coordinator = self.manager.coordinator
        if kind == "goose_rx":
            watch = self.watches.get(event.get("key"))
            if watch is None:
                return
            with self.lock:
                watch.update({key: event[key] for key in ("state", "problem", "st_num", "sq_num", "frames")
                              if key in event})
                # Received header fields are kept apart from the expected SCD values of the watch.
                watch.update({f"received_{key}": event[key] for key in ("conf_rev", "go_id", "dataset", "test")
                              if key in event})
                watch["observed_at"] = datetime.now(timezone.utc).isoformat(timespec="seconds")
            payload = {key: value for key, value in event.items() if key not in {"event", "values"}}
            coordinator.emit("goose_status", **payload)
            self._route(watch, event)
            self.inputs.apply(watch, event)
        elif kind == "goose_stats":
            with self.lock:
                for item in event.get("watches", []):
                    if item["key"] in self.watches:
                        # Retransmissions raise sqNum without a state change; the stats carry it
                        # (change 0091: the page showed sqNum 0 while frames 4, 5... arrived).
                        self.watches[item["key"]].update(frames=item["frames"], age_ms=item["age_ms"],
                                                         **({"sq_num": item["sq_num"]} if "sq_num" in item else {}))
        elif kind == "goose_unknown":
            with self.lock:
                self.status["unknown"].append({key: event.get(key) for key in UNKNOWN_FIELDS})
            coordinator.emit("goose_unknown", gocb_ref=event.get("gocb_ref"), app_id=event.get("app_id"))
        elif kind == "goose_unknown_cap":
            with self.lock:
                self.status["unknown_limit"] = event.get("limit")
        elif kind == "worker_failed":
            self.status.update(state="failed", message="The GOOSE listener stopped unexpectedly")
            coordinator.emit("goose_listener_failed", message=event.get("error") or "")

    def _route(self, watch, event):
        coordinator = self.manager.coordinator
        for subscriber in watch["subscribers"]:
            ied_id = subscriber["ied_id"]
            updates = lgos_updates(subscriber.get("lgos"), event, coordinator.ied_indexes.get(ied_id, {}))
            if updates and ied_id in coordinator.sample_writer.locks:
                try:
                    coordinator.sample_writer.apply(ied_id, [(ref, value, None) for ref, value in updates],
                                                    "goose_supervision")
                except ValueError as error:
                    coordinator.emit("goose_supervision_failed", ied_id=ied_id, message=str(error))
        publisher = watch.get("publisher") or {}
        if publisher.get("role") == "real" and "values" in event:
            ied = coordinator.projects.find_ied(publisher["ied_id"])
            control = next(c for c in ied["goose"]["publishes"] if c["gocb_ref"] == watch["gocb_ref"])
            mapped = dataset_values(ied, control, event["values"])
            if mapped:
                self.manager.values.apply(ied["id"], mapped, "live_goose")

    def snapshot(self):
        with self.lock:
            return {**self.status, "watches": [dict(watch) for watch in self.watches.values()]}

    def stop(self):
        process, self.process = self.process, None
        if process is not None:
            process.close()
        self.status = {"state": "off", "message": "", "unknown": []}
