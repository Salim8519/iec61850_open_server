"""Background load of a soak: one UI-like SSE client, /health polling, periodic canvas commands.

Each feed keeps per-window lists that the sampler drains every interval (`drain()`), so
drift over hours is visible instead of being averaged away.
"""
import collections
import http.client
import json
import threading
import time

from engine_bench import canvas


class SoakSse:
    """EventSource-like client: reconnects with the last id, counts kinds, id jumps and lag."""

    def __init__(self, engine, push_topics=()):
        """push_topics: names subscribed on the Go push stream (/api/v1/push) like a UI tab;
        empty means the plain /api/v1/events stream. Push frames count as kind "push:<topic>"."""
        self.engine, self.last_id, self.disconnects, self.id_jumps = engine, 0, 0, 0
        self.push_topics, self.topic_answer = list(push_topics), None
        self.kinds, self.lags, self._stop, self._lock = collections.Counter(), [], threading.Event(), threading.Lock()
        self._connection = None

    def start(self):
        threading.Thread(target=self._loop, name="soak-sse", daemon=True).start()
        return self

    def _loop(self):
        while not self._stop.is_set():
            try:
                self._stream()
            except (OSError, ValueError, http.client.HTTPException):
                pass
            if not self._stop.is_set():
                self.disconnects += 1
                time.sleep(1)

    def _stream(self):
        self._connection = http.client.HTTPConnection("127.0.0.1", self.engine.port, timeout=60)
        path = (f"/api/v1/push?client=benchsoak&after={self.last_id}" if self.push_topics
                else f"/api/v1/events?after={self.last_id}")
        self._connection.request("GET", path, headers={
            "Host": f"127.0.0.1:{self.engine.port}", "Cookie": self.engine.cookie})
        response, buffer = self._connection.getresponse(), b""
        if self.push_topics:  # the tab's watched list, sent once its stream is attached
            self.topic_answer = self.engine.json("PUT", "/api/v1/push/topics", {
                "client": "benchsoak", "topics": [{"t": name, "params": {}} for name in self.push_topics],
                "resync": []}, timeout=30)
        while not self._stop.is_set():
            chunk = response.read1(65536)
            if not chunk:
                return
            buffer += chunk
            while b"\n\n" in buffer:
                frame, buffer = buffer.split(b"\n\n", 1)
                self._frame(frame)

    def _frame(self, frame):
        if frame.startswith(b"event: push\n"):
            for line in frame.split(b"\n"):
                if line.startswith(b"data: "):
                    with self._lock:
                        self.kinds["push:" + str(json.loads(line[6:]).get("t"))] += 1
            return
        for line in frame.split(b"\n"):
            if line.startswith(b"id: "):
                number = int(line[4:])
                if self.last_id and number != self.last_id + 1:
                    self.id_jumps += 1
                self.last_id = number
            elif line.startswith(b"data: "):
                event = json.loads(line[6:])
                with self._lock:
                    self.kinds[event.get("kind")] += 1
                    if event.get("kind") == "signal" and "timestamp" in event:
                        self.lags.append(time.time() - event["timestamp"])

    def drain(self):
        with self._lock:
            kinds, lags = dict(self.kinds), self.lags
            self.kinds, self.lags = collections.Counter(), []
        return kinds, lags

    def close(self):
        self._stop.set()
        try:
            self._connection.sock.close()
        except (AttributeError, OSError):
            pass


class Periodic:
    """Calls `action()` every `every` seconds; keeps (time, seconds, error) per call."""

    def __init__(self, every, action, name):
        self.every, self.action, self.results = every, action, []
        self._stop, self._lock = threading.Event(), threading.Lock()
        self._thread = threading.Thread(target=self._run, name=name, daemon=True)

    def start(self):
        self._thread.start()
        return self

    def _run(self):
        next_at = time.monotonic()
        while not self._stop.is_set():
            started = time.perf_counter()
            try:
                error = self.action()
            except Exception as failure:
                error = f"{type(failure).__name__}: {failure}"[:200]
            with self._lock:
                self.results.append((time.time(), time.perf_counter() - started, error))
            next_at += self.every
            self._stop.wait(max(0.0, next_at - time.monotonic()))

    def drain(self):
        with self._lock:
            results, self.results = self.results, []
        return results

    def close(self):
        self._stop.set()
        self._thread.join(30)


def health_poll(engine):
    def action():
        status, _, _ = engine.request("GET", "/api/v1/health", timeout=30)
        return None if status == 200 else f"status {status}"
    return action


def breaker_rounds(engine, nodes):
    """Round robin over the breakers; each breaker alternates close/open."""
    state = {"index": 0, "closed": {node: False for node in nodes}}

    def action():
        node = nodes[state["index"] % len(nodes)]
        state["index"] += 1
        action_name = "open" if state["closed"][node] else "close"
        _, error = canvas.operate(engine, node, action_name)
        if not error:
            state["closed"][node] = not state["closed"][node]
        return error
    return action
