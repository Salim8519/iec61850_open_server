"""Append compact run events and lifecycle indexes without rewriting history."""
import json
import threading
from datetime import datetime, timezone
from uuid import uuid4

SEGMENT_BYTES = 10 * 1024 * 1024


def _append(path, value):
    with path.open("a", encoding="utf-8", newline="\n") as stream:
        stream.write(json.dumps(value, allow_nan=False, separators=(",", ":")) + "\n")
        stream.flush()


class RunLog:
    def __init__(self, root):
        self.run_id = uuid4().hex
        self.root = root
        self.path = root / self.run_id
        self.path.mkdir(parents=True, exist_ok=True)
        self.lock = threading.Lock()
        self.index = 0
        self.stream = None
        self.size = 0
        self.pending = 0
        self.closed = False
        started = {"kind": "run_started", "run_id": self.run_id,
                   "at": datetime.now(timezone.utc).isoformat(), "application": "0.1.0"}
        _append(self.path / "manifest.jsonl", started)
        _append(self.root / "runs.jsonl", started)

    def _open_segment(self):
        target = self.path / f"events-{self.index:04d}.jsonl"
        self.stream = target.open("a", encoding="utf-8", newline="\n")
        _append(self.path / "manifest.jsonl",
                {"kind": "segment_opened", "run_id": self.run_id,
                 "at": datetime.now(timezone.utc).isoformat(), "file": target.name})
        return target

    def write(self, event):
        with self.lock:
            record = json.dumps({"run_id": self.run_id, **event}, allow_nan=False,
                                separators=(",", ":")) + "\n"
            if self.stream is not None and self.size + len(record.encode("utf-8")) > SEGMENT_BYTES:
                if self.stream:
                    self.stream.close()
                    self.stream = None
                self.index += 1
                self.size = 0
            if self.stream is None:
                self._open_segment()
            self.stream.write(record)
            self.size += len(record.encode("utf-8"))
            self.pending += 1
            if self.pending >= 100 or event.get("kind") != "signal":
                self.stream.flush()
                self.pending = 0

    def close(self):
        with self.lock:
            if self.closed:
                return
            if self.stream:
                self.stream.close()
                self.stream = None
            closed = {"kind": "run_closed", "run_id": self.run_id,
                      "at": datetime.now(timezone.utc).isoformat()}
            _append(self.path / "manifest.jsonl", closed)
            _append(self.root / "runs.jsonl", closed)
            self.closed = True
