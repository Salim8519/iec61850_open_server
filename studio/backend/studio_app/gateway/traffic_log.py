"""Bounded, numbered logs of gateway traffic and commands for the Tools page."""
import collections
import threading
import time

TRAFFIC_LINES = 1000
COMMAND_LINES = 200


class NumberedLog:
    def __init__(self, capacity):
        self.lock = threading.Lock()
        self.entries = collections.deque(maxlen=capacity)
        self.sequence = 0

    def add(self, **entry):
        with self.lock:
            self.sequence += 1
            entry.update(seq=self.sequence, at=time.time())
            self.entries.append(entry)
            return entry

    def after(self, seq=0, limit=300):
        with self.lock:
            fresh = [dict(entry) for entry in self.entries if entry["seq"] > seq]
        return {"entries": fresh[-limit:], "last": self.sequence,
                "dropped": max(0, len(fresh) - limit)}

    def update(self, seq, **changes):
        with self.lock:
            for entry in self.entries:
                if entry["seq"] == seq:
                    entry.update(changes)
                    return

    def clear(self):
        with self.lock:
            self.entries.clear()
