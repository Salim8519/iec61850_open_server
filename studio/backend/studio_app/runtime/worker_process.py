"""Lifecycle and request/reply tracking for one native IED process."""
import os
import logging
import queue
import subprocess
import threading
from uuid import uuid4
from .worker_protocol import read_frame, write_frame


class WorkerProcess:
    def __init__(self, executable, on_event=None):
        self.on_event = on_event or (lambda event: None)
        self.pending = {}
        self.lock = threading.RLock()
        self.closing = False
        flags = subprocess.CREATE_NO_WINDOW if os.name == "nt" else 0
        self.process = subprocess.Popen([str(executable)], stdin=subprocess.PIPE,
                                        stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                        creationflags=flags, bufsize=0)
        self.reader = threading.Thread(target=self._read, daemon=True)
        self.reader.start()
        self.errors = threading.Thread(target=self._stderr, daemon=True)
        self.errors.start()

    def _read(self):
        failure = None
        try:
            while True:
                message = read_frame(self.process.stdout)
                if "event" in message:
                    self._deliver(message)
                else:
                    with self.lock:
                        target = self.pending.get(message.get("id"))
                    if target is not None:
                        target.put(message)
        except (EOFError, OSError, ValueError) as error:
            failure = str(error)
        finally:
            with self.lock:
                for target in self.pending.values():
                    target.put({"ok": False, "error": failure or "Worker stopped"})
            if not self.closing:
                self._deliver({"event": "worker_failed", "error": failure})

    def _deliver(self, event):
        try:
            self.on_event(event)
        except Exception:
            logging.getLogger(__name__).exception("Native event consumer failed")

    def _stderr(self):
        for line in iter(self.process.stderr.readline, b""):
            self._deliver({"event": "native_diagnostic", "message": line.decode("utf-8", "replace").strip()})

    def request(self, op, timeout=10, **values):
        request_id = uuid4().hex
        response = queue.Queue()
        with self.lock:
            if self.process.poll() is not None:
                raise RuntimeError(f"Native worker exited: {self.process.returncode}")
            self.pending[request_id] = response
            try:
                write_frame(self.process.stdin, {"version": 1, "id": request_id, "op": op, **values})
            except Exception:
                self.pending.pop(request_id, None)
                raise
        try:
            result = response.get(timeout=timeout)
            if not result.get("ok"):
                raise RuntimeError(result.get("error", "Native request failed"))
            return result.get("result")
        except queue.Empty as error:
            raise TimeoutError(f"Native {op} request timed out") from error
        finally:
            with self.lock:
                self.pending.pop(request_id, None)

    def close(self):
        self.closing = True
        if self.process.poll() is None:
            try:
                self.request("stop", timeout=3)
            except (RuntimeError, TimeoutError, OSError):
                pass
            try:
                self.process.wait(timeout=3)
            except subprocess.TimeoutExpired:
                self.process.terminate()
                try:
                    self.process.wait(timeout=3)
                except subprocess.TimeoutExpired:
                    self.process.kill()
                    self.process.wait()
        self.reader.join(timeout=1)
        self.errors.join(timeout=1)
        for stream in (self.process.stdin, self.process.stdout, self.process.stderr):
            stream.close()
