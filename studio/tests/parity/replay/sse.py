"""Read the real `GET /api/v1/events` stream (framing included) while a scenario runs."""
import http.client
import threading
import time


class SseTap:
    def __init__(self, port, cookie):
        self.port, self.cookie = port, cookie
        self.frames, self.raw = [], bytearray()
        self._lock = threading.Lock()
        self._cut = 0
        self._last_arrival = time.monotonic()
        self._connection = None
        self._started = threading.Event()

    def start(self):
        threading.Thread(target=self._run, name="golden-sse", daemon=True).start()
        if not self._started.wait(10):
            raise RuntimeError("SSE stream did not open")
        return self

    def _run(self):
        self._connection = http.client.HTTPConnection("127.0.0.1", self.port, timeout=None)
        self._connection.request("GET", "/api/v1/events", headers={"Host": f"127.0.0.1:{self.port}",
                                                                    "Cookie": self.cookie})
        self._socket = self._connection.sock
        response = self._connection.getresponse()
        self.status, self.headers = response.status, dict(response.getheaders())
        self._started.set()
        buffer = b""
        try:
            while True:
                chunk = response.read1(65536)
                if not chunk:
                    return
                with self._lock:
                    self.raw.extend(chunk)
                buffer += chunk
                while b"\n\n" in buffer:
                    frame, buffer = buffer.split(b"\n\n", 1)
                    self._frame(frame.decode("utf-8"))
        except (OSError, ValueError, http.client.HTTPException):
            return

    def _frame(self, text):
        record = {"comment": None, "id": None, "data": None, "retry": None}
        for line in text.split("\n"):
            field, _, value = line.partition(":")
            value = value[1:] if value.startswith(" ") else value
            if field == "":
                record["comment"] = value
            elif field in ("id", "data", "retry"):
                record[field] = value
        with self._lock:
            self.frames.append(record)
            self._last_arrival = time.monotonic()

    def settle(self, quiet=0.25, limit=3.0):
        """Wait until no frame arrived for `quiet` seconds (at most `limit`), then return the
        event frames since the previous cut: the events caused by the last step."""
        deadline = time.monotonic() + limit
        time.sleep(quiet)
        while time.monotonic() < deadline:
            with self._lock:
                idle = time.monotonic() - self._last_arrival
            if idle >= quiet:
                break
            time.sleep(quiet - idle)
        with self._lock:
            taken = self.frames[self._cut:]
            self._cut = len(self.frames)
        return [frame for frame in taken if frame["data"] is not None]

    def wait_for(self, predicate, timeout):
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            with self._lock:
                frames = list(self.frames[self._cut:])
            if any(frame["data"] is not None and predicate(frame) for frame in frames):
                return True
            time.sleep(0.05)
        return False

    def head(self):
        """The first frame exactly as sent (`retry: 1000\n\n`)."""
        with self._lock:
            raw = bytes(self.raw)
        end = raw.find(b"\n\n")
        return raw if end < 0 else raw[:end + 2]

    def close(self):
        socket = getattr(self, "_socket", None)
        if socket is not None:
            try:
                socket.shutdown(2)
                socket.close()
            except OSError:
                pass
