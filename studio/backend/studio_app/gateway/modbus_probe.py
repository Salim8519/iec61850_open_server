"""'Find registers' for the Modbus client: one bounded probe at a time, in the background.

The worker reads blocks and splits only where the server refuses (at most 10 requests per
second); this keeps the progress and the answering ranges for the page. It only reads.
"""
import threading

PROBE_TIMEOUT = 1500  # seconds; 10,000 addresses at 10 requests per second fit with margin


class ModbusProbe:
    def __init__(self):
        self.lock = threading.Lock()
        self.current = {"state": "idle"}

    def snapshot(self):
        with self.lock:
            return dict(self.current)

    @property
    def running(self):
        return self.snapshot()["state"] == "running"

    def start(self, process, table, start, end, done):
        with self.lock:
            if self.current["state"] == "running":
                raise ValueError("Find registers is already running")
            self.current = {"state": "running", "table": table, "start": start, "end": end, "requests": 0,
                            "address": start, "found": [], "message": ""}
        threading.Thread(target=self._run, args=(process, table, start, end, done), daemon=True,
                         name="modbus-probe").start()
        return self.snapshot()

    def progress(self, event):
        with self.lock:
            if self.current["state"] == "running":
                self.current.update(requests=event.get("requests", 0), address=event.get("address", 0))

    def _run(self, process, table, start, end, done):
        try:
            answer = process.request("probe", timeout=PROBE_TIMEOUT, table=table, start=start, end=end)
            with self.lock:
                self.current.update(state="done", found=answer["found"], requests=answer["requests"])
        except Exception as error:
            with self.lock:
                self.current.update(state="failed", message=str(error))
        done(self.snapshot())
