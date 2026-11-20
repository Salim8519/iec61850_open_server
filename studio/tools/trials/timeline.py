"""Turn one Run or Stop's observations into named steps on its critical path (pure functions).

Every step is an interval between two observed moments, so the steps add up to the total the
API saw (click to HTTP answer). Sources: Windows (address list, listeners, kernel process
times), the engine's lab journal (`network/lab_journal.jsonl`) and its event stream.
"""
from datetime import datetime
import json
from pathlib import Path

RUN_STEPS = ["addresses", "listen check + GOOSE bus", "process start", "model load", "saved values",
             "first values read", "initial states", "test modes", "rest (listeners, reply)"]
STOP_STEPS = ["workers stop", "address release", "stopped shown", "reply"]


def journal(data_dir, event, since):
    """{ip: moment} of lab-journal records with this event ("applied"/"removed") after since."""
    path = Path(data_dir) / "network/lab_journal.jsonl"
    found = {}
    if not path.is_file():
        return found
    for line in path.read_text(encoding="utf-8").splitlines():
        record = json.loads(line)
        moment = datetime.fromisoformat(record["at"]).timestamp()
        if record.get("event") == event and moment >= since:
            found[record["item"].get("ip", "")] = moment
    return found


def _signals(events, *sources):
    return sorted(e["timestamp"] for e in events if e.get("kind") == "signal" and e.get("source") in sources)


def _runtime(events, state):
    moments = [e["timestamp"] for e in events if e.get("kind") == "runtime" and e.get("state") == state]
    return min(moments) if moments else None


def _chain(click, marks, end, names):
    """Durations between consecutive marks; a missing mark (None) gives a 0 s step."""
    steps, previous = {}, click
    for name, mark in zip(names, marks + [end]):
        moment = previous if mark is None else max(mark, previous)
        steps[name] = round(moment - previous, 4)
        previous = moment
    return steps


def run_steps(click, answered, events, applied, created, listened, port):
    """Steps of one Run. applied: journal {ip: t}; created: {pid: (created, exited)} of new
    studio_ied processes; listened: watcher {(ip, port): t}; events: engine events since click."""
    births = sorted(c for c, _ in created.values() if c >= click)
    ready = sorted(t for (_, p), t in listened.items() if p == port)
    running = _runtime(events, "running")
    snapshot = _signals(events, "initial_native_snapshot")
    initial = _signals(events, "initial_state")
    modes = _signals(events, "test_mode", "behaviour")
    tests_end = modes[-1] if modes else None
    marks = [max(applied.values()) if applied else None,
             births[0] if births else None, births[-1] if births else None,
             ready[-1] if ready else None, running,
             (initial or modes)[0] if (initial or modes) else (snapshot[-1] if snapshot else None),
             modes[0] if (initial and modes) else (initial[-1] if initial else None),
             tests_end]
    steps = _chain(click, marks, answered, RUN_STEPS)
    facts = {"total": round(answered - click, 4), "running_event": round(running - click, 4) if running else None,
             "ieds_listening": len(ready), "workers_started": len(births), "snapshot_reads": len(snapshot),
             "initial_state_writes": len(initial), "test_mode_writes": len(modes)}
    return steps, facts


def stop_steps(click, answered, events, removed, workers, gone):
    """Steps of one Stop. removed: journal {ip: t}; workers: {pid: (created, exited)};
    gone: watcher {ip: t} of addresses Windows no longer lists."""
    exits = sorted(x for _, x in workers.values() if x and x >= click)
    released = max(list(removed.values()) + list(gone.values())) if (removed or gone) else None
    stopped = _runtime(events, "stopped")
    steps = _chain(click, [exits[-1] if exits else None, released, stopped], answered, STOP_STEPS)
    facts = {"total": round(answered - click, 4), "stopped_event": round(stopped - click, 4) if stopped else None,
             "workers_exited": len(exits), "addresses_removed": len(removed), "addresses_gone": len(gone)}
    return steps, facts
