"""Station health verdicts (HEALTH-02, -03, -08, -12)."""
import logging
from datetime import datetime, timedelta, timezone

from studio_app.domain.station_health import build_health

NOW = datetime(2026, 10, 14, 10, 0, 30, tzinfo=timezone.utc)
SIGNALS = [{"ref": "P1LD0/LLN0.Mod.stVal", "fc": "ST"}, {"ref": "P1LD0/LLN0.Beh.stVal", "fc": "ST"},
           {"ref": "P1LD0/LPHD1.Sim.stVal", "fc": "ST"}]
COMPARISON = {"summary": {"match": 3, "missing_on_device": 0, "extra_on_device": 0, "type_mismatch": 0,
                          "datasets_different": 0, "reports_missing": 0}, "signals": [], "datasets": [], "reports": []}


def real(**extra):
    return {"id": "r1", "name": "P1", "role": "real", "signals": SIGNALS, "reports": [], "goose": {}, **extra}


def session(mode=None, checked_at="2026-10-14T10:00:20+00:00"):
    values = mode or {"P1LD0/LLN0.Mod.stVal": 1, "P1LD0/LLN0.Beh.stVal": 1, "P1LD0/LPHD1.Sim.stVal": False}
    return {"state": "live", "host": "10.0.0.1", "port": 102, "comparison": COMPARISON, "health": {
        "rcbs": {"blocks": {}, "checked_at": checked_at, "source": "mms_read"},
        "gocbs": {"values": {}, "errors": {}, "checked_at": checked_at, "source": "mms_read"},
        "mode": {"values": values, "errors": {}, "checked_at": checked_at, "source": "mms_read"}}}


def run(ieds, sessions, goose=None, now=NOW, running=True):
    return build_health(ieds, running, sessions, goose or {"state": "off", "message": "GOOSE off"}, now)


def test_not_checked_is_never_ok():
    stopped = run([real()], {}, running=False)
    entry = stopped["ieds"][0]
    assert entry["verdict"] == "not_checked" and stopped["verdict"] == "not_checked"
    assert all(row["message"] for row in entry["checks"] if row["result"] == "not_checked")
    simulated = run([{"id": "s1", "name": "S1", "role": "simulated", "signals": [], "goose": {}}], {})
    assert simulated["ieds"][0]["verdict"] == "not_checked"
    assert "Simulated by ScadaBench" in simulated["ieds"][0]["checks"][0]["message"]
    healthy = run([real()], {"r1": session()})
    assert healthy["ieds"][0]["verdict"] == "ok"


def test_verdict_is_worst_result():
    test_mode = session({"P1LD0/LLN0.Mod.stVal": 3, "P1LD0/LLN0.Beh.stVal": 3, "P1LD0/LPHD1.Sim.stVal": False})
    broken = {**COMPARISON, "summary": {**COMPARISON["summary"], "missing_on_device": 2}}
    result = run([real(), real(id="r2", name="P2")], {"r1": test_mode, "r2": {**session(), "comparison": broken}})
    verdicts = {entry["name"]: entry["verdict"] for entry in result["ieds"]}
    assert verdicts == {"P1": "warning", "P2": "problem"} and result["verdict"] == "problem"
    assert result["summary"]["problem"] == 1 and result["summary"]["warning"] == 1
    mode = [row for row in result["ieds"][0]["checks"] if row["item"] == "P1LD0/LLN0.Mod.stVal"][0]
    assert "test" in mode["message"]


def test_results_carry_time_source_and_staleness():
    fresh = run([real()], {"r1": session()})["ieds"][0]["checks"]
    assert all(row["observed_at"] and row["source"] == "mms_read" for row in fresh)
    assert not any(row["stale"] for row in fresh)
    later = run([real()], {"r1": session()}, now=NOW + timedelta(minutes=5))["ieds"][0]["checks"]
    assert all(row["stale"] for row in later if row["area"] == "mode")
    assert not any(row["stale"] for row in later if row["area"] != "mode")


def test_logs_exclude_customer_data(caplog):
    secret = "P1LD0/LLN0.Mod.stVal"
    with caplog.at_level(logging.DEBUG):
        run([real()], {"r1": session()})
        run([real()], {}, running=False)
    assert secret not in caplog.text and "10.0.0.1" not in caplog.text
