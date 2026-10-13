"""Fixed deadlines with bounded recovery and explicit discarded-tick accounting."""
import math


class Cadence:
    def __init__(self, now, period=0.1, max_backlog=5):
        if not math.isfinite(period) or period <= 0 or max_backlog < 1:
            raise ValueError("Cadence requires a positive period and backlog limit")
        self.period = period
        self.max_backlog = max_backlog
        self.reset(now)

    def reset(self, now):
        self.deadline = now + self.period

    def take(self, now):
        if now < self.deadline:
            return None
        lateness = now - self.deadline
        due = math.floor((lateness + self.period * 1e-8) / self.period) + 1
        skipped = max(0, due - self.max_backlog)
        self.deadline += (skipped + 1) * self.period
        return {"lateness_seconds": lateness, "skipped_ticks": skipped}
