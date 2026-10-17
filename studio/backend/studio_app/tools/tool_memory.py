"""Remember what the user last did in each tool (owner rule: every setting is remembered).

Kept in the data folder beside the preferences, written atomically, and checked on
read so a damaged file only resets the tool memory instead of breaking the app.
"""
import json
import os
import threading
from pathlib import Path

MAX_HOSTS = 50
MAX_WATCH = 200


def _text(value, limit=64):
    return value if isinstance(value, str) and len(value) <= limit else None


def _watch(items):
    if not isinstance(items, list):
        return []
    clean = [{"ref": item["ref"], "fc": item["fc"], "type": str(item.get("type", ""))[:32]}
             for item in items[:MAX_WATCH] if isinstance(item, dict) and _text(item.get("ref"), 200)
             and _text(item.get("fc"), 4)]
    return clean


def _clean(raw):
    raw = raw if isinstance(raw, dict) else {}
    explorer = raw.get("explorer") if isinstance(raw.get("explorer"), dict) else {}
    find = raw.get("find") if isinstance(raw.get("find"), dict) else {}
    watches = explorer.get("watch_by_host") if isinstance(explorer.get("watch_by_host"), dict) else {}
    port = explorer.get("port")
    interval = explorer.get("interval_ms")
    return {
        "explorer": {"host": _text(explorer.get("host")) or "", "port": port if isinstance(port, int) else 102,
                     "interval_ms": interval if isinstance(interval, int) and 500 <= interval <= 60000 else 1000,
                     "watch_by_host": {key: _watch(value) for key, value in list(watches.items())[-MAX_HOSTS:]
                                       if _text(key)}},
        "find": {"range": _text(find.get("range")) or "", "port": find.get("port") if isinstance(find.get("port"), int) else 102,
                 "choice": _text(find.get("choice"), 200) or ""},
    }


class ToolMemory:
    def __init__(self, directory):
        self.path = Path(directory) / "tool-settings.json"
        self.lock = threading.Lock()

    def get(self):
        try:
            return _clean(json.loads(self.path.read_text(encoding="utf-8")))
        except (OSError, ValueError):
            return _clean({})

    def update(self, tool, **changes):
        with self.lock:
            current = self.get()
            current[tool].update(changes)
            cleaned = _clean(current)
            self.path.parent.mkdir(parents=True, exist_ok=True)
            temporary = self.path.with_name(self.path.name + ".tmp")
            temporary.write_text(json.dumps(cleaned, indent=2), encoding="utf-8")
            os.replace(temporary, self.path)
            return cleaned[tool]
