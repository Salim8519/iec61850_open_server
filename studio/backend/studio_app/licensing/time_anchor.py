"""Remember the latest wall-clock time ScadaBench has seen, to detect clock rollback."""
from datetime import datetime
import os
from pathlib import Path


def _read_int(path):
    try:
        return int(Path(path).read_text(encoding="ascii").strip() or 0)
    except (OSError, ValueError):
        return 0


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
    def __init__(self, anchor_file, runs_file=None):
        self.anchor_file = Path(anchor_file)
        self.runs_file = Path(runs_file) if runs_file else None

    def max_seen(self):
        latest = _read_int(self.anchor_file)
        if self.runs_file is not None:
            latest = max(latest, _latest_run_timestamp(self.runs_file))
        return latest

    def advance(self, now_ts):
        """Move the anchor forward only; never backwards."""
        if now_ts <= _read_int(self.anchor_file):
            return
        try:
            self.anchor_file.parent.mkdir(parents=True, exist_ok=True)
            temporary = self.anchor_file.with_suffix(".tmp")
            temporary.write_text(str(int(now_ts)), encoding="ascii")
            os.replace(temporary, self.anchor_file)
        except OSError:
            pass
