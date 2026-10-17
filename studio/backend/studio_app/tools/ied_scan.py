"""Find IEDs: probe TCP port 102 on a confirmed range, then identify every open port."""
import socket
import threading
import time
from concurrent.futures import ThreadPoolExecutor
from studio_app.domain.scan_range import plan_range
from studio_app.runtime.worker_process import WorkerProcess

PARALLEL = 64
CONNECT_TIMEOUT_S = 0.5


def port_open(host, port, timeout=CONNECT_TIMEOUT_S):
    """A plain TCP handshake that is closed at once; no data is sent."""
    started = time.monotonic()
    try:
        with socket.create_connection((host, port), timeout=timeout):
            return round((time.monotonic() - started) * 1000, 1)
    except OSError:
        return None


class ScanService:
    def __init__(self, executable, probe=port_open, worker=WorkerProcess):
        self.executable = executable
        self.probe = probe
        self.worker = worker
        self.lock = threading.Lock()
        self.stop_event = threading.Event()
        self.thread = None
        self.state = {"state": "idle", "range": "", "port": 102, "total": 0, "probed": 0, "found": [],
                      "message": "", "started": None, "seconds": 0}

    def preview(self, text, port=102):
        planned = plan_range(text)
        self._check_port(port)
        return {key: planned[key] for key in ("range", "count", "first", "last")} | {"port": port}

    @staticmethod
    def _check_port(port):
        if not isinstance(port, int) or not 1 <= port <= 65535:
            raise ValueError("The port must be a number between 1 and 65535 (IEC 61850 uses 102)")

    def start(self, text, port=102):
        planned = plan_range(text)
        self._check_port(port)
        if not self.executable.exists():
            raise RuntimeError("The live client (studio_live.exe) is not built yet; rebuild the native programs")
        if self.thread and self.thread.is_alive():
            raise ValueError("A scan is already running; stop it first")
        self.stop_event = threading.Event()
        with self.lock:
            self.state = {"state": "scanning", "range": planned["range"], "port": port, "total": planned["count"],
                          "probed": 0, "found": [], "message": f"Looking for IEDs on {planned['count']} address(es)",
                          "started": time.time(), "seconds": 0}
        self.thread = threading.Thread(target=self._run, args=(planned["hosts"], port, self.stop_event),
                                       daemon=True, name="find-ieds")
        self.thread.start()
        return self.status()

    def _run(self, hosts, port, stop):
        open_hosts = []
        try:
            with ThreadPoolExecutor(max_workers=PARALLEL) as pool:
                futures = {host: pool.submit(self._probe_one, host, port, stop) for host in hosts}
                for host in hosts:
                    result = futures[host].result()
                    with self.lock:
                        self.state["probed"] += 1
                    if result is not None:
                        open_hosts.append(host)
                        self._add({"ip": host, "port": port, "tcp_ms": result, "identified": None,
                                   "vendor": "", "model": "", "revision": "", "devices": [], "reason": "Identifying…"})
            if not stop.is_set() and open_hosts:
                self._identify(open_hosts, port, stop)
            self._finish(stop)
        except Exception as error:
            with self.lock:
                self.state.update(state="failed", message=f"The scan stopped: {error}")

    def _probe_one(self, host, port, stop):
        return None if stop.is_set() else self.probe(host, port)

    def _identify(self, hosts, port, stop):
        with self.lock:
            self.state["message"] = f"Asking {len(hosts)} device(s) who they are"
        process = self.worker(self.executable)
        try:
            for host in hosts:
                if stop.is_set():
                    break
                try:
                    answer = process.request("identify", timeout=10, ip=host, port=port, timeout_ms=3000)
                except Exception as error:
                    answer = {"identified": False, "reason": str(error)}
                self._update(host, answer)
        finally:
            process.close()

    def _add(self, entry):
        with self.lock:
            self.state["found"].append(entry)

    def _update(self, host, answer):
        with self.lock:
            for entry in self.state["found"]:
                if entry["ip"] == host:
                    entry.update({key: answer[key] for key in ("identified", "vendor", "model", "revision",
                                                               "devices", "connect_ms") if key in answer})
                    notes = [answer.get(key) for key in ("reason", "identity_note", "devices_note") if answer.get(key)]
                    entry["reason"] = "; ".join(notes)

    def _finish(self, stop):
        with self.lock:
            self.state["seconds"] = round(time.time() - self.state["started"], 1)
            found = self.state["found"]
            named = sum(1 for entry in found if entry.get("identified"))
            if stop.is_set():
                self.state.update(state="stopped", message=f"Stopped. {len(found)} device(s) found so far.")
            else:
                self.state.update(state="done", message=(f"Done. {len(found)} address(es) answer on port "
                                                         f"{self.state['port']}, {named} identified as IEC 61850."
                                                         if found else "Done. No device answered on this port."))

    def stop(self):
        self.stop_event.set()
        if self.thread is not None:
            self.thread.join(timeout=15)
        return self.status()

    def status(self):
        with self.lock:
            state = dict(self.state, found=[dict(entry) for entry in self.state["found"]])
        if state["started"] and state["state"] == "scanning":
            state["seconds"] = round(time.time() - state["started"], 1)
        return state
