"""Remember the license server's signed time for trials, and when the anchor went missing.

Only genuine stamps (server_time.verify) are ever used; an unreadable or foreign copy is
ignored, never held against the user. Both values live in three places (clock_store.py).
"""
import json
from . import server_time
from .clock_store import ClockValue


class TrialClock:
    def __init__(self, home, registry=None, keys=None):
        self.stamps = ClockValue("server_time", home, registry)
        self.missing = ClockValue("grace_start", home, registry)
        self.keys = keys

    def confirmed(self):
        """Time of the newest genuine stamp in any place, or 0."""
        times = [server_time.verify(copy, self.keys) for copy in self.stamps.read_all()]
        return max([moment for moment in times if moment], default=0)

    def remember(self, stamp):
        """Store a genuine, newer stamp everywhere and clear the missing-anchor mark."""
        moment = server_time.verify(stamp, self.keys)
        if moment is None:
            return False
        if moment >= self.confirmed():
            self.stamps.write(json.dumps(stamp if isinstance(stamp, dict) else json.loads(stamp)))
        if self.suspicious_since():
            self.missing.write(0)
        return True

    def suspicious_since(self):
        """When this PC noticed every anchor copy gone (0 = never, or cleared by a stamp)."""
        return min([value for value in self.missing.read_ints() if value > 0], default=0)

    def mark_suspicious(self, now_ts):
        if not self.suspicious_since():
            self.missing.write(int(now_ts))
