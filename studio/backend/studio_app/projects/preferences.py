"""Per-installation UI preferences (for example "guided tour seen") in the data folder.

Kept by the backend, not the browser, because the packaged app's address and port
can change between launches. Only known keys with the right type are accepted.
"""
import json
import os
from pathlib import Path
import threading
from . import list_filters

TOUR_VERSION = 1
# The two SLD signal layers (spec signal-layers) are remembered as 0 (off) or 1 (on).
DEFAULTS = {"tour_seen_version": 0, "sld_goose_layer": 0, "sld_readers_layer": 0}
TYPES = {"tour_seen_version": int, "sld_goose_layer": int, "sld_readers_layer": int}


class PreferenceStore:
    def __init__(self, directory):
        self.path = Path(directory) / "preferences.json"
        self._lock = threading.Lock()

    def get(self):
        try:
            stored = json.loads(self.path.read_text(encoding="utf-8"))
        except (OSError, ValueError):
            stored = {}
        if not isinstance(stored, dict):
            stored = {}
        values = dict(DEFAULTS)
        for key, kind in TYPES.items():
            if isinstance(stored.get(key), kind) and not isinstance(stored.get(key), bool):
                values[key] = stored[key]
        return {**values, "list_filters": list_filters.clean_stored(stored.get("list_filters")),
                "tour_version": TOUR_VERSION}

    def update(self, changes):
        if not isinstance(changes, dict):
            raise ValueError("Preferences must be a JSON object")
        unknown = sorted(set(changes) - set(TYPES) - {"list_filters"})
        if unknown:
            raise ValueError(f"Unknown preference: {', '.join(unknown)}")
        for key, value in changes.items():
            if key == "list_filters":
                continue
            if not isinstance(value, TYPES[key]) or isinstance(value, bool):
                raise ValueError(f"Preference {key} must be a whole number")
            if not 0 <= value <= (1 if key.startswith("sld_") else 1000):
                raise ValueError(f"Preference {key} is out of range")
        with self._lock:
            stored = self.get()
            current = {key: value for key, value in stored.items() if key in TYPES}
            current["list_filters"] = stored["list_filters"]
            if "list_filters" in changes:
                current["list_filters"] = list_filters.merge_update(stored["list_filters"], changes["list_filters"])
            current.update({key: value for key, value in changes.items() if key != "list_filters"})
            self.path.parent.mkdir(parents=True, exist_ok=True)
            temporary = self.path.with_name(self.path.name + ".tmp")
            temporary.write_text(json.dumps(current, indent=2), encoding="utf-8")
            os.replace(temporary, self.path)
        return self.get()
