"""Append-only record of Windows network changes that Studio itself made."""
import json
import threading
from datetime import datetime, timezone


def item_key(item):
    if item["kind"] == "alias":
        return f"alias:{item['adapter_guid']}:{item['ip']}"
    return f"firewall:{item['name']}"


class NetworkJournal:
    def __init__(self, directory):
        self.path = directory / "network" / "journal.jsonl"
        self.lock = threading.Lock()

    def append(self, event, item, fingerprint="", result=""):
        if event not in {"applied", "removed", "failed", "skipped"}:
            raise ValueError("Unknown journal event")
        record = {"at": datetime.now(timezone.utc).isoformat(), "event": event,
                  "fingerprint": fingerprint, "item": item, "result": str(result)[:500]}
        with self.lock:
            self.path.parent.mkdir(parents=True, exist_ok=True)
            with self.path.open("a", encoding="utf-8") as stream:
                stream.write(json.dumps(record, sort_keys=True) + "\n")
        return record

    def records(self):
        if not self.path.exists():
            return []
        with self.lock:
            lines = self.path.read_text(encoding="utf-8").splitlines()
        return [json.loads(line) for line in lines if line.strip()]

    def outstanding(self):
        """Owned items Studio applied and has not removed yet, newest state wins."""
        state = {}
        for record in self.records():
            item = record["item"]
            if not item.get("owned"):
                continue
            if record["event"] == "applied":
                state[item_key(item)] = item
            elif record["event"] == "removed":
                state.pop(item_key(item), None)
        return list(state.values())

    def leftovers(self, is_present):
        """Outstanding items that still exist; reported to the user, never auto-removed."""
        return [item for item in self.outstanding() if is_present(item)]
