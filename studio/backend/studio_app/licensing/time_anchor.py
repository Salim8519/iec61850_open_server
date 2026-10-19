"""Remember the latest wall-clock time ScadaBench has seen, to detect clock rollback."""
from datetime import datetime
import os
from pathlib import Path


def _latest_run_timestamp(runs_file, window=8192):
    """Read only the tail of runs.jsonl; its "at" fields are ISO-8601 UTC times."""
    try:
        with open(runs_file, "rb") as stream:
            stream.seek(0, os.SEEK_END)
            size = stream.tell()
            stream.seek(max(0, size - window))
            tail = stream.read().decode("utf-8", "replace")
    except OSError:
        return 0
    latest = 0
    for marker in tail.split('"at":"')[1:]:
        try:
            latest = max(latest, int(datetime.fromisoformat(marker.split('"', 1)[0]).timestamp()))
        except ValueError:
            continue
    return latest


class TimeAnchor:
    """Latest time seen: the three anchor copies (clock_store.py) plus the run log tail."""

    def __init__(self, value, runs_file=None):
        self.value = value
        self.runs_file = Path(runs_file) if runs_file else None

    def max_seen(self):
        latest = max(self.value.read_ints(), default=0)
        if self.runs_file is not None:
            latest = max(latest, _latest_run_timestamp(self.runs_file))
        return latest

    def advance(self, now_ts):
        """Move the anchor forward only, and restore any copy that was deleted or lowered."""
        copies = self.value.read_ints()
        target = max([int(now_ts), *copies])
        if len(copies) < self.value.places() or any(copy != target for copy in copies):
            self.value.write(target)
