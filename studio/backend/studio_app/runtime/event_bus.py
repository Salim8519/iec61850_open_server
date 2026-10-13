"""Bounded, sequenced application events with reconnect cursor support."""
import threading
import time
from collections import deque


class EventBus:
    def __init__(self, capacity=5000):
        self.items = deque(maxlen=capacity)
        self.sequence = 0
        self.changed = threading.Condition()

    def publish(self, kind, **payload):
        with self.changed:
            self.sequence += 1
            event = {"id": self.sequence, "kind": kind, "timestamp": time.time(), **payload}
            self.items.append(event)
            self.changed.notify_all()
            return event

    def after(self, cursor, wait=0):
        with self.changed:
            if wait and self.sequence <= cursor:
                self.changed.wait(timeout=wait)
            if self.items and cursor and cursor < self.items[0]["id"] - 1:
                return [{"id": self.sequence, "kind": "resync", "timestamp": time.time()}]
            return [item.copy() for item in self.items if item["id"] > cursor]
