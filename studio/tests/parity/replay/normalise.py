"""Normaliser: the only differences allowed between two transcripts.

Rules live in golden/normalise.json (versioned, one reason each); this module applies them to
parsed trees identically for both sides. Anything not covered is compared exactly.
"""
import json
from pathlib import Path
import re

from parity.replay.jsonlex import Num

RULES = Path(__file__).resolve().parents[1] / "golden" / "normalise.json"
UUID = re.compile(r"(?<![0-9a-f])[0-9a-f]{32}(?![0-9a-f])")
ISO = re.compile(r"^\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(\.\d+)?([+-]\d{2}:\d{2}|Z)?$")


def _matches(key, rule):
    return isinstance(key, str) and (key in rule.get("keys", ()) or
                                     any(key.startswith(prefix) for prefix in rule.get("prefixes", ())) or
                                     any(key.endswith(suffix) for suffix in rule.get("suffixes", ())))


class Normaliser:
    def __init__(self, context, rules=None):
        document = rules or json.loads(RULES.read_text(encoding="utf-8"))
        self.rules = {rule["name"]: rule for rule in document["rules"]}
        self.ids = {}
        self.first_event_id = None
        self.replacements = []
        for raw, token in ((context.get("data"), "<data>"), (context.get("bin"), "<bin>")):
            if raw:
                self.replacements += [(raw, token), (raw.replace("\\", "/"), token)]
        self.replacements.sort(key=lambda pair: -len(pair[0]))
        port = context.get("port")
        self.port = re.compile(rf"(?<=:){port}(?!\d)") if port else None

    def _uuid(self, match):
        value = match.group(0)
        if value not in self.ids:
            self.ids[value] = f"<uuid:{len(self.ids) + 1}>"
        return self.ids[value]

    def text(self, value):
        for raw, token in self.replacements:
            value = value.replace(raw, token)
        if self.port is not None:
            value = self.port.sub("<port>", value)
        value = re.sub(r"udp:\d+:", "udp:<port>:", value)
        return UUID.sub(self._uuid, value)

    def tree(self, value, key=None):
        if isinstance(value, dict):
            result = {self.text(name): self.tree(item, name) for name, item in value.items()}
            if result and all(UUID.fullmatch(name) for name in value):
                # Flask sorts keys; random ids sort differently per run (rule id_keyed_order).
                result = dict(sorted(result.items(), key=lambda pair: int(pair[0][6:-1])))
            return result
        if isinstance(value, list):
            return [self.tree(item) for item in value]
        if isinstance(value, Num):
            if _matches(key, self.rules["wall_time"]) and 1e9 <= value.value() <= 4e9:
                return "<ts>"
            if _matches(key, self.rules["durations"]) and value.value() >= 0:
                return "<dur:float>" if value.is_float else "<dur:int>"
            return value
        if isinstance(value, str):
            if _matches(key, self.rules["wall_time"]) and ISO.match(value):
                return "<ts>"
            return self.text(value)
        return value

    def event(self, frame_id, tree):
        """SSE ids become relative to the first event of the scenario."""
        number = int(frame_id)
        if self.first_event_id is None:
            self.first_event_id = number
        tree = self.tree(tree)
        if isinstance(tree, dict) and "id" in tree:
            tree["id"] = f"<event:{int(tree['id']) - self.first_event_id}>"
        return {"sse_id": number - self.first_event_id, "event": tree}

    def cookie(self, header):
        """Set-Cookie: presence and attributes only."""
        if not header:
            return None
        name, _, rest = header.partition("=")
        attributes = sorted(part.strip() for part in rest.split(";")[1:])
        return {"name": name, "attributes": attributes}
