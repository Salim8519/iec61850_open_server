"""IED Explorer: one read-only association to any IEC 61850 server, with a polled watch list."""
import ipaddress
import threading
import time
from studio_app.domain.explorer_tree import node_view
from studio_app.domain.explorer_values import describe
from studio_app.runtime.worker_process import WorkerProcess

MAX_WATCH = 200
MIN_INTERVAL_MS, MAX_INTERVAL_MS = 500, 60000


def check_endpoint(host, port):
    try:
        address = ipaddress.IPv4Address(str(host).strip())
    except ValueError as error:
        raise ValueError("Type the device address as four numbers, for example 192.168.1.10") from error
    if not isinstance(port, int) or not 1 <= port <= 65535:
        raise ValueError("The port must be a number between 1 and 65535 (IEC 61850 uses 102)")
    return str(address), port


class ExplorerService:
    def __init__(self, executable, emit=lambda kind, **payload: None):
        self.executable = executable
        self.emit = emit
        self.lock = threading.RLock()
        self.spawn_lock = threading.Lock()
        self.process = None
        self.endpoint = None
        self.state = "disconnected"
        self.message = ""
        self.cache = {}
        self.watch = []
        self.values = {}
        self.interval_ms = 1000
        self.polling = False
        self.poll_stop = threading.Event()
        self.poller = None
        self.reads = 0

    def warm(self):
        """Start the worker in the background, so Connect does not wait for a process start."""
        if self.process is None and self.executable.exists():
            threading.Thread(target=self._warm, daemon=True, name="explorer-warm").start()
        return self.status()

    def _warm(self):
        try:
            self._worker()
        except Exception as error:
            self.message = str(error)

    def _worker(self):
        with self.spawn_lock:
            if self.process is not None and self.process.process.poll() is None:
                return self.process
            process = WorkerProcess(self.executable, self._event)
            if "explore" not in process.request("hello").get("capabilities", []):
                process.close()
                raise RuntimeError("The live client is outdated; rebuild the native programs")
            self.process = process
            return process

    def connect(self, host, port=102):
        host, port = check_endpoint(host, port)
        self.disconnect()
        if not self.executable.exists():
            raise RuntimeError("The live client (studio_live.exe) is not built yet; rebuild the native programs")
        with self.lock:
            self.state, self.message, self.endpoint = "connecting", f"Connecting to {host}:{port}", (host, port)
        try:
            self._worker().request("connect", timeout=15, ip=host, port=port, timeout_ms=5000)
        except Exception as error:
            self.state, self.message = "failed", str(error)
            raise RuntimeError(f"Could not open an IEC 61850 association to {host}:{port}. Check the address, "
                               "the cable and that the device allows another client.") from error
        with self.lock:
            self.state, self.message = "connected", f"Connected to {host}:{port} (read-only)"
        self.emit("explorer_connected", host=host, port=port)
        return self.status()

    def _event(self, event):
        if event.get("event") == "live_connection" and event.get("state") == "closed" and self.state == "connected":
            self.state, self.message = "lost", "The device closed the connection. Press Connect to try again."
            self.poll_stop.set()

    def _request(self, op, timeout=30, **values):
        process = self.process
        if process is None or self.state != "connected":
            raise ValueError("Connect to a device first")
        return process.request(op, timeout=timeout, **values)

    def devices(self):
        return self._cached(("devices",), lambda: self._request("explore")["names"])

    def nodes(self, device):
        return self._cached(("nodes", device), lambda: self._request("explore", device=device)["names"])

    def node(self, reference):
        if reference.count("/") != 1:
            raise ValueError("A logical node reference looks like LD/LN, for example IED1CTRL/XCBR1")
        return self._cached(("node", reference),
                            lambda: node_view(reference, self._request("explore", timeout=60, node=reference)))

    def _cached(self, key, load):
        if key not in self.cache:
            self.cache[key] = load()
        return self.cache[key]

    def read(self, items):
        if not items or len(items) > MAX_WATCH:
            raise ValueError(f"Read between 1 and {MAX_WATCH} attributes at a time")
        answer = self._request("read", refs=[{"ref": item["ref"], "fc": item["fc"]} for item in items])
        self.reads += 1
        now = time.time()
        results = {}
        for entry in answer["values"]:
            if "structure" in entry:
                for ref, raw in entry["structure"].items():
                    results[ref] = {"raw": raw, **describe(ref, raw), "at": now}
            elif "value" in entry:
                results[entry["ref"]] = {"raw": entry["value"], **describe(entry["ref"], entry["value"]), "at": now}
            else:
                error = entry.get("error")
                text = (f"device refused the read (access error {entry.get('access_error')})"
                        if error == "access_error" else f"read failed (MMS error {error})")
                results[entry["ref"]] = {"raw": None, "text": text, "status": "error", "at": now}
        with self.lock:
            self.values.update(results)
        return results

    def set_watch(self, items, interval_ms=None):
        clean, seen = [], set()
        for item in items:
            key = (str(item["ref"]), str(item["fc"]))
            if key not in seen:
                seen.add(key)
                clean.append({"ref": key[0], "fc": key[1], "type": str(item.get("type", ""))})
        if len(clean) > MAX_WATCH:
            raise ValueError(f"The watch list holds at most {MAX_WATCH} attributes")
        if interval_ms is not None:
            if not isinstance(interval_ms, int) or not MIN_INTERVAL_MS <= interval_ms <= MAX_INTERVAL_MS:
                raise ValueError("Choose a poll interval between 0.5 and 60 seconds")
            self.interval_ms = interval_ms
        with self.lock:
            self.watch = clean
        return self.status()

    def start_polling(self):
        if not self.watch:
            raise ValueError("Add attributes to the watch list first")
        self._request("hello")
        self.stop_polling()
        self.poll_stop = threading.Event()
        self.polling = True
        self.poller = threading.Thread(target=self._poll, args=(self.poll_stop,), daemon=True, name="explorer-poll")
        self.poller.start()
        return self.status()

    def _poll(self, stop):
        while not stop.is_set():
            started = time.monotonic()
            try:
                if self.watch:
                    self.read(list(self.watch))
            except Exception as error:
                self.message = f"Polling paused: {error}"
                break
            stop.wait(max(0.05, self.interval_ms / 1000 - (time.monotonic() - started)))
        self.polling = False

    def stop_polling(self):
        self.poll_stop.set()
        poller, self.poller = self.poller, None
        if poller is not None and poller is not threading.current_thread():
            poller.join(timeout=10)
        self.polling = False
        return self.status()

    def disconnect(self):
        """End the association; the worker stays warm for the next Connect."""
        self.stop_polling()
        with self.lock:
            was_connected = self.state in ("connected", "lost")
            self.cache, self.values = {}, {}
            if self.state != "disconnected":
                self.state, self.message = "disconnected", "Disconnected"
        if was_connected and self.process is not None:
            try:
                self.process.request("disconnect", timeout=10)
            except Exception:
                self.close()
        return self.status()

    def close(self):
        process, self.process = self.process, None
        if process is not None:
            process.close()

    def status(self):
        with self.lock:
            host, port = self.endpoint or ("", 102)
            return {"state": self.state, "message": self.message, "host": host, "port": port,
                    "watch": [{**item, **self.values.get(item["ref"], {"text": "", "status": "pending", "at": None})}
                              for item in self.watch],
                    "interval_ms": self.interval_ms, "polling": self.polling, "reads": self.reads}
