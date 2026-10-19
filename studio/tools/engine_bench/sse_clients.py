"""Browser-like SSE clients: count delivered `signal` events and their lag (S2, S10)."""
import http.client
import json
import threading
import time


class SseClient:
    def __init__(self, engine):
        self.engine, self.signals, self.lags, self.resyncs, self.skipped_ticks = engine, 0, [], 0, 0
        self._stop, self._socket = threading.Event(), None

    def start(self):
        threading.Thread(target=self._run, name="bench-sse", daemon=True).start()
        return self

    def _run(self):
        connection = http.client.HTTPConnection("127.0.0.1", self.engine.port, timeout=None)
        connection.request("GET", "/api/v1/events?after=0", headers={"Host": f"127.0.0.1:{self.engine.port}",
                                                                    "Cookie": self.engine.cookie})
        self._socket = connection.sock
        response = connection.getresponse()
        buffer = b""
        try:
            while not self._stop.is_set():
                chunk = response.read1(65536)
                if not chunk:
                    return
                buffer += chunk
                while b"\n\n" in buffer:
                    frame, buffer = buffer.split(b"\n\n", 1)
                    self._frame(frame)
        except (OSError, ValueError, http.client.HTTPException):
            return

    def _frame(self, frame):
        for line in frame.split(b"\n"):
            if not line.startswith(b"data: "):
                continue
            event = json.loads(line[6:])
            if event.get("kind") == "signal":
                self.signals += 1
                if len(self.lags) < 200000:
                    self.lags.append(time.time() - event["timestamp"])
            elif event.get("kind") == "automation_late":
                self.skipped_ticks += int(event.get("skipped_ticks") or 0)
            elif event.get("kind") == "resync":
                self.resyncs += 1

    def mark(self):
        return self.signals

    def close(self):
        self._stop.set()
        if self._socket is not None:
            try:
                self._socket.close()
            except OSError:
                pass


def percentile(values, share):
    if not values:
        return None
    ordered = sorted(values)
    return ordered[min(len(ordered) - 1, int(share * len(ordered)))]
