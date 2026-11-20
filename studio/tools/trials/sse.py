"""Record the engine's event stream (GET /api/v1/events) with the engine's own timestamps.

Every bus event (`runtime` running/stopped, `signal` with its source, `listen_check_failed`...)
carries the engine's wall-clock `timestamp`, the same scale as winclock.now().

    recorder = EventRecorder(engine).start()
    ...
    recorder.since(t0, kind="runtime")   # [{"id", "kind", "timestamp", ...}, ...]
    recorder.stop()
"""
import http.client
import json
import threading

from winclock import now


class EventRecorder:
    def __init__(self, engine):
        self.engine = engine
        self.events, self.lock = [], threading.Lock()
        self.connection = None
        self.ready = threading.Event()

    def start(self, timeout=10):
        threading.Thread(target=self._run, daemon=True).start()
        if not self.ready.wait(timeout):
            raise RuntimeError("event stream did not open")
        return self

    def _run(self):
        self.connection = http.client.HTTPConnection("127.0.0.1", self.engine.port, timeout=None)
        headers = {"Host": f"127.0.0.1:{self.engine.port}", "Cookie": self.engine.cookie,
                   "Accept": "text/event-stream"}
        try:
            self.connection.request("GET", "/api/v1/events", headers=headers)
            response = self.connection.getresponse()
            self.ready.set()
            data = []
            for raw in response:
                line = raw.decode("utf-8", "replace").rstrip("\r\n")
                if line.startswith("data: "):
                    data.append(line[6:])
                elif not line and data:
                    self._keep("\n".join(data))
                    data = []
        except (OSError, http.client.HTTPException, ValueError):
            pass  # closed by stop() or the engine exiting
        finally:
            self.ready.set()

    def _keep(self, text):
        event = json.loads(text)
        event["received"] = now()
        with self.lock:
            self.events.append(event)

    def since(self, moment, kind=None):
        with self.lock:
            return [event for event in self.events
                    if event.get("timestamp", 0) >= moment and (kind is None or event.get("kind") == kind)]

    def clear(self):
        with self.lock:
            self.events.clear()

    def stop(self):
        if self.connection:
            try:
                self.connection.sock and self.connection.sock.close()
            except OSError:
                pass
