"""One verdict per IED and for the station: does the real station match its SCD?

Pure: it only combines SCD data with results the live sessions and the GOOSE listener
already have. "Not checked" always says why and never counts as OK.
"""
from datetime import datetime
from .goose_consistency import device_gocb_checks, result, unknown_publishers, wire_check
from .health_model_checks import mode_rows, model_rows, report_setting_rows

RANK = {"ok": 0, "not_checked": 1, "warning": 2, "problem": 3}
MAX_AGE_S = {"mode": 30}


def worst(results):
    return max(results, key=lambda verdict: RANK[verdict], default="not_checked")


def _age(observed_at, now):
    try:
        return (now - datetime.fromisoformat(observed_at)).total_seconds()
    except (TypeError, ValueError):
        return None


def _mark_stale(rows, now):
    for row in rows:
        age = _age(row["observed_at"], now)
        row["stale"] = bool(row["area"] in MAX_AGE_S and age is not None and age > MAX_AGE_S[row["area"]])
    return rows


def _wire_rows(ied, goose):
    watches = [watch for watch in goose.get("watches", [])
               if (watch.get("publisher") or {}).get("ied_id") == ied["id"]]
    return [wire_check(watch, goose) for watch in watches]


def _real_rows(ied, session, goose):
    if session is None:
        return [result("connection", "Connection", "not_checked", "Press Run to connect to the device.")]
    state, status = session.get("state"), session
    if state in {"retrying", "no_address"}:
        return [result("connection", "Connection", "problem", status.get("message") or "The device does not answer.")]
    if state != "live" or not status.get("health"):
        return [result("connection", "Connection", "not_checked", "Not connected yet. ScadaBench is reaching the device.")]
    health = status["health"]
    when = health["mode"]["checked_at"]
    rows = [result("connection", "Connection", "ok", f"Connected to {status.get('host')}:{status.get('port')}.",
                   when, "mms_read")]
    if status.get("comparison"):
        rows += model_rows(ied, status["comparison"], when)
    rows += report_setting_rows(ied, health["rcbs"])
    rows += device_gocb_checks(ied, health["gocbs"])
    rows += mode_rows(ied, health["mode"])
    return rows


def ied_rows(ied, running, session, goose):
    role = ied.get("role", "simulated")
    if role == "off":
        return [result("role", "IED", "not_checked", "This IED is set to Off.")]
    if not running:
        return [result("connection", "Run", "not_checked", "Press Run to check the station.")]
    if role == "simulated":
        return [result("model", "Data model", "not_checked", "Simulated by ScadaBench; model checks apply to real devices.")] \
            + _wire_rows(ied, goose)
    return _real_rows(ied, session, goose) + _wire_rows(ied, goose)


def build_health(ieds, running, sessions, goose, now):
    """Return {verdict, summary, ieds: [{id, name, role, verdict, checks}], unknown, unknown_note}."""
    entries = []
    for ied in ieds:
        rows = _mark_stale(ied_rows(ied, running, sessions.get(ied["id"]), goose), now)
        entries.append({"id": ied["id"], "name": ied["name"], "role": ied.get("role", "simulated"),
                        "verdict": worst(row["result"] for row in rows), "checks": rows})
    unknown, note = unknown_publishers(goose) if running else ([], "")
    station = worst(entry["verdict"] for entry in entries)
    summary = {name: sum(entry["verdict"] == name for entry in entries) for name in RANK}
    return {"verdict": station, "summary": summary, "ieds": entries, "unknown": unknown, "unknown_note": note,
            "running": running, "checked_at": now.isoformat(timespec="seconds")}
