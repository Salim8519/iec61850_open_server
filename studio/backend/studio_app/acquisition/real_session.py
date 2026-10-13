"""One read-only connection to a real IED: connect, compare, report or poll, recover."""
import threading
from studio_app.domain.access_policy import choose_access
from studio_app.domain.model_compare import compare
from studio_app.domain.network_settings import real_endpoint
from studio_app.runtime.worker_process import WorkerProcess

REQUIRED = {"live_discover", "live_reports", "live_poll", "read_only_client"}


class RealSession:
    def __init__(self, manager, ied, watched):
        self.manager = manager
        self.ied = ied
        self.watched = watched
        self.process = None
        self.stopping = threading.Event()
        self.lost = threading.Event()
        self.thread = threading.Thread(target=self._run, daemon=True, name=f"live-{ied['name']}")
        host, port, origin = real_endpoint(ied)
        self.status = {"ied_id": ied["id"], "name": ied["name"], "state": "starting", "host": host,
                       "port": port, "endpoint_source": origin, "message": "", "access": None,
                       "comparison": None, "polling": []}

    def start(self):
        self.thread.start()

    def _set(self, state, message=""):
        self.status.update(state=state, message=message)
        self.ied["state"] = "stopped" if state == "stopped" else f"real_{state}"
        self.manager.coordinator.emit("live_status", ied_id=self.ied["id"], state=state, message=message)

    def _event(self, event):
        kind = event.get("event")
        if kind == "live_report":
            self.manager.values.apply(self.ied["id"], event.get("values", {}), "live_report")
        elif kind == "live_connection" and event.get("state") == "closed" and not self.stopping.is_set():
            self.lost.set()
        elif kind == "worker_failed" and not self.stopping.is_set():
            self.lost.set()

    def _run(self):
        delay = 2
        host, port = self.status["host"], self.status["port"]
        if not host:
            self._set("no_address", "This IED has no address in the SCD. Type the device address first.")
            return
        while not self.stopping.is_set():
            self.lost.clear()
            try:
                self._session(host, port)
                delay = 2
            except Exception as error:
                if self.stopping.is_set():
                    break
                self._set("retrying", f"{error}. Trying again in {delay} s.")
            self._close_process()
            if self.stopping.wait(delay):
                break
            delay = min(delay * 2, 30)

    def _session(self, host, port):
        self._set("connecting", f"Connecting to {host}:{port}")
        self.process = WorkerProcess(self.manager.executable, self._event)
        if not REQUIRED <= set(self.process.request("hello").get("capabilities", [])):
            raise RuntimeError("The live client is outdated; rebuild the native programs")
        self.process.request("connect", timeout=15, ip=host, port=port)
        self._set("discovering", "Reading the device data model")
        discovery = self.process.request("discover", timeout=120)
        self.status["comparison"] = compare(self.ied, discovery)
        blocks = {}
        refs = [report["ref"] for report in self.ied.get("reports", [])]
        if refs and self.ied["live"]["access"] == "auto_reports":
            answer = self.process.request("rcb_status", timeout=30, rcbs=refs)
            blocks = {item["rcb"]: item for item in answer["blocks"]}
        access = choose_access(self.ied, blocks, self.ied["live"]["access"])
        used = []
        for rcb in access["reports"]:
            result = self.process.request("rcb_reserve", timeout=30, rcb=rcb)
            if result.get("reserved"):
                used.append(rcb)
            else:
                access["notes"].append(f"Report block {rcb.rsplit('.', 1)[-1]} became busy; polling instead.")
        access["reports"] = used
        self.status["access"] = access
        covered = {ref for report in self.ied.get("reports", []) if report["ref"] in used
                   for dataset in self.ied.get("datasets", []) if dataset.get("ref") == report.get("dataset_ref")
                   for ref in dataset.get("signal_refs", [])}
        self.status["polling"] = [item for item in self.watched if item["ref"] not in covered]
        self._set("live", "Receiving live values" + (" by report" if used else " by polling"))
        while not self.stopping.is_set() and not self.lost.is_set():
            if self.status["polling"]:
                self.read(self.status["polling"], "live_poll")
            self.stopping.wait(self.ied["live"]["poll_ms"] / 1000)
        if self.lost.is_set():
            self.manager.values.mark_stale(self.ied["id"], "Connection to the device was lost")
            raise RuntimeError("Connection to the device was lost")

    def read(self, items, source="live_poll"):
        process = self.process
        if process is None or self.status["state"] != "live":
            raise ValueError(f"{self.ied['name']} is not connected yet")
        answer = process.request("read", timeout=30, refs=items[:2000])
        values = {}
        for entry in answer["values"]:
            if "value" in entry:
                values[entry["ref"]] = entry["value"]
            elif "structure" in entry:
                values.update(entry["structure"])
            else:
                self.manager.coordinator.emit("live_read_error", ied_id=self.ied["id"], ref=entry["ref"],
                                              message=f"The device could not read this point (error {entry.get('error')})")
        return self.manager.values.apply(self.ied["id"], values, source)

    def _close_process(self):
        process, self.process = self.process, None
        if process is not None:
            try:
                process.request("disconnect", timeout=15)
            except Exception:
                pass
            process.close()

    def stop(self):
        self.stopping.set()
        self._close_process()
        self.thread.join(timeout=20)
        self._set("stopped", "Disconnected; every report block Studio used was given back")
