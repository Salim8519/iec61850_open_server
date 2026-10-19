"""Compare two transcripts after normalising both: one line per difference.

`<scenario> step 7 GET /api/v1/health body.$.ieds[2].status: python "ok" != go "stale"`
Classes: value, numtype, order, missing, extra, header, status, event. Key order is never
relaxed; causal events compare as ordered lists, periodic kinds by count (+-1) and shape.
"""
import base64
import json
from pathlib import Path

from parity.replay.jsonlex import Num, has_nan, loads
from parity.replay.normalise import Normaliser

EVENT_KINDS = Path(__file__).resolve().parents[1] / "golden" / "event_kinds.json"


def _show(value):
    return json.dumps(value) if not isinstance(value, Num) else str(value)


def compare(left, right, path="$"):
    if isinstance(left, Num) and isinstance(right, Num):
        if left == right:
            return
        same = left.value() == right.value() if left.is_float or right.is_float else False
        yield ("numtype" if same else "value", path, left, right)
    elif type(left) is not type(right) and not (isinstance(left, str) and isinstance(right, str)
                                                 and not isinstance(left, Num) and not isinstance(right, Num)):
        yield ("value", path, left, right)
    elif isinstance(left, dict):
        for key in left:
            if key not in right:
                yield ("missing", f"{path}.{key}", left[key], None)
        for key in right:
            if key not in left:
                yield ("extra", f"{path}.{key}", None, right[key])
        shared = [key for key in left if key in right]
        if shared != [key for key in right if key in left]:
            yield ("order", path, shared, [key for key in right if key in left])
        for key in shared:
            yield from compare(left[key], right[key], f"{path}.{key}")
    elif isinstance(left, list):
        for index in range(max(len(left), len(right))):
            if index >= len(right):
                yield ("missing", f"{path}[{index}]", left[index], None)
            elif index >= len(left):
                yield ("extra", f"{path}[{index}]", None, right[index])
            else:
                yield from compare(left[index], right[index], f"{path}[{index}]")
    elif left != right:
        yield ("value", path, left, right)


def _body(record, normaliser):
    kind, body = record.get("body_kind"), record.get("body")
    if kind == "json":
        return normaliser.tree(loads(body))
    if kind == "text":
        return normaliser.text(body)
    if kind == "zip":
        project = body.get("project.json")
        return {"entries": body["entries"], "project.json": normaliser.tree(loads(project)) if project else None}
    if kind == "bytes":
        return {"sha256": record.get("body_sha256"), "size": len(base64.b64decode(body))}
    return body


def _events(record, normaliser, periodic):
    causal, counts, shapes = [], {}, {}
    for frame in record.get("events", []):
        event = normaliser.event(frame["id"], loads(frame["data"]))
        kind = event["event"].get("kind")
        if kind in periodic:
            counts[kind] = counts.get(kind, 0) + 1
            shapes.setdefault(kind, list(event["event"]))
        else:
            causal.append(event)
    return causal, counts, shapes


def normalised(transcript, periodic):
    header, steps = transcript[0], transcript[1:]
    normaliser = Normaliser(header)
    result = []
    for record in steps:
        item = {"step": record["step"], "req": record.get("req") or "wait_event", "status": record.get("status"),
                "found": record.get("found", record.get("skipped")),
                "headers": {key: normaliser.text(value) if value else value
                            for key, value in (record.get("headers") or {}).items()},
                "set_cookie": normaliser.cookie(record.get("set_cookie"))}
        item["body"] = _body(record, normaliser) if "body_kind" in record else None
        item["events"], item["periodic"], item["shapes"] = _events(record, normaliser, periodic)
        result.append(item)
    return {"sse": header.get("sse"), "steps": result}


def diff(left, right, names=("python", "go"), scenario="?"):
    periodic = set(json.loads(EVENT_KINDS.read_text(encoding="utf-8"))["periodic"])
    a, b = normalised(left, periodic), normalised(right, periodic)
    lines = []

    def report(cls, where, x, y):
        lines.append((cls, f"{scenario} {where}: {names[0]} {_show(x)} != {names[1]} {_show(y)}"))
    for cls, path, x, y in compare(a["sse"], b["sse"], "sse"):
        report("header", path, x, y)
    if len(a["steps"]) != len(b["steps"]):
        report("missing", "steps", len(a["steps"]), len(b["steps"]))
    for x, y in zip(a["steps"], b["steps"]):
        where = f"step {x['step']} {x['req']}"
        for field, cls in (("status", "status"), ("found", "event"), ("headers", "header"), ("set_cookie", "header")):
            for _, path, p, q in compare(x[field], y[field], field):
                report(cls, f"{where} {path}", p, q)
        for cls, path, p, q in compare(x["body"], y["body"], "body.$"):
            report(cls, f"{where} {path}", p, q)
        for cls, path, p, q in compare(x["events"], y["events"], "events"):
            report("event", f"{where} {path}", p, q)
        for kind in sorted(set(x["periodic"]) | set(y["periodic"])):
            if abs(x["periodic"].get(kind, 0) - y["periodic"].get(kind, 0)) > 1:
                report("event", f"{where} periodic {kind} count", x["periodic"].get(kind, 0), y["periodic"].get(kind, 0))
            if kind in x["shapes"] and kind in y["shapes"] and x["shapes"][kind] != y["shapes"][kind]:
                report("event", f"{where} periodic {kind} shape", x["shapes"][kind], y["shapes"][kind])
    return lines


def nan_count(transcript):
    count = 0
    for record in transcript[1:]:
        if record.get("body_kind") == "json" and has_nan(loads(record["body"])):
            count += 1
        count += sum(has_nan(loads(frame["data"])) for frame in record.get("events", []))
    return count


def summary(lines):
    totals = {}
    for cls, _ in lines:
        totals[cls] = totals.get(cls, 0) + 1
    return totals
