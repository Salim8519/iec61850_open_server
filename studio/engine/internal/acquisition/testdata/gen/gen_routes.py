"""Golden scenario for the live routes: real Flask HTTP answers, stopped and running.

Model substation (FEED1, BUS1). Stopped: every route. Then FEED1 becomes a real device,
BUS1 gets a Sampled Values stream, the run starts with fake native workers (route_fakes),
GOOSE/SV/engine/MMS events are injected and every route is read again. Writes
testdata/generated/routes_scenario.json: the steps, each answer and the events in between.
"""
import copy
import json
import tempfile
import time
from pathlib import Path

import common
import route_fakes as fakes
from studio_app.acquisition import goose_monitor, health_checks, health_snapshot, manager, real_session, sv_monitor
from studio_app.api import live as live_api
from studio_app.networking.npcap import npcap_status
from studio_app.projects.goose_links import _declarations
from studio_app.projects.scl_sources import stored_sources
from studio_app.simulation import mms_access
from studio_app.version import __version__

app, client, folder = common.app_with("model")
service, coordinator = app.extensions["projects"], app.extensions["runtime"]
live = coordinator.live
steps, captured = [], []
publish = coordinator.events.publish


def capture(kind, **payload):
    captured.append([kind, json.dumps(payload)])
    return publish(kind, **payload)


def patch(script):
    coordinator.events.publish = capture
    for module in (goose_monitor, sv_monitor, health_checks, health_snapshot):
        module.datetime = fakes.FixedDatetime
    mms_access.time = fakes.Monotonic
    real_session.MODE_REFRESH_S = 3600
    worker = fakes.worker_class(script)
    real_session.WorkerProcess = goose_monitor.WorkerProcess = sv_monitor.WorkerProcess = worker
    live.engine.starter = worker
    for name in ("studio_live.exe", "studio_goose.exe"):
        (live.executable.parent / name).write_bytes(b"fake")
    root = Path(tempfile.mkdtemp())
    (root / "System32" / "Npcap").mkdir(parents=True)
    (root / "System32" / "Npcap" / "wpcap.dll").write_bytes(b"")
    npcap = npcap_status(root)
    live_api.npcap_status = manager.npcap_status = live.share.npcap = lambda: npcap
    live.share.adapters = lambda: copy.deepcopy(fakes.ADAPTERS)
    return npcap


def project_step():
    project = copy.deepcopy(service.project)
    sources = list(stored_sources(service.project).values())
    declarations = _declarations(sources) if sources else {"ieds": {}, "blocks": {}, "subscriptions": {},
                                                            "problems": {}, "remote": {}}
    for ied in project["ieds"]:
        ied.pop("source", None)
        ied.pop("cfg", None)
    steps.append({"step": "project", "project": project, "run_id": f"{service.session_id}-{service.revision}",
                  "declarations": {**declarations, "blocks": [[list(key), block] for key, block in
                                                               declarations["blocks"].items()]}})


def http(method, path, payload=None, raw=None, ctype="application/json"):
    del captured[:]
    data = raw if raw is not None else (json.dumps(payload) if payload is not None else None)
    answer = client.open(path, method=method, data=data, content_type=ctype if data is not None else None)
    steps.append({"step": "http", "method": method, "path": path, "body": data, "ctype": ctype if data is not None else "",
                  "status": answer.status_code, "type": answer.headers.get("Content-Type"),
                  "disposition": answer.headers.get("Content-Disposition"),
                  "response": answer.get_data(as_text=True), "events": list(captured)})


def inject(target, event, ied_id=None):
    del captured[:]
    if target == "mms":
        coordinator.mms_access.update(ied_id, copy.deepcopy(event))
    else:
        {"goose": live.goose, "sv": live.sv, "engine": live.engine}[target]._event(copy.deepcopy(event))
    steps.append({"step": "event", "target": target, "ied_id": ied_id, "event": event, "events": list(captured)})


def ids():
    return {ied["name"]: ied["id"] for ied in service.project["ieds"]}


def stopped_routes():
    feed, bus = ids()["FEED1"], ids()["BUS1"]
    for path in ("/live", "/goose", "/goose/links", "/mms/access", "/sv", "/health", "/health.csv",
                 "/health/report.html", "/goose/share", f"/live/{feed}/comparison", f"/live/{feed}/comparison.csv",
                 "/ieds/nope/signals", f"/ieds/{bus}/signals?q=pos&limit=3"):
        http("GET", "/api/v1" + path)
    http("POST", "/api/v1/health/refresh")
    http("POST", f"/api/v1/live/{feed}/read", {"refs": []})
    http("POST", f"/api/v1/live/{feed}/read", {"refs": ["x"]})
    http("POST", f"/api/v1/live/{feed}/read", raw="[1]")
    http("POST", "/api/v1/goose/conflicts", {"choice": "maybe"})
    http("POST", "/api/v1/goose/conflicts", {"choice": "continue"})
    http("POST", "/api/v1/goose/conflicts", {"choice": "stop", "ied_id": bus})
    http("PUT", "/api/v1/goose/share", {"reach": "outside"})
    http("PUT", "/api/v1/goose/share", {"reach": "outside", "confirm": True, "adapter_guid": "{old}"})
    http("PUT", "/api/v1/goose/share", {"reach": "inside", "uplink_guid": "{66666666-7777-8888-9999-000000000000}"})
    http("PUT", f"/api/v1/ieds/{feed}/test-mode", {"devices": {"FEED1LD1": 9}})
    http("PUT", f"/api/v1/ieds/{feed}/test-mode", {"devices": {"FEED1LD1": 3}, "simulation": True})
    http("PUT", f"/api/v1/ieds/{bus}/test-mode", raw="not json", ctype="text/plain")
    http("PUT", "/api/v1/ieds/nope/test-mode", {})


def prepare_run():
    feed, bus = ids()["FEED1"], ids()["BUS1"]
    for ied in service.project["ieds"]:
        if ied["id"] == feed:
            ied["role"] = "real"
            ied["live"] = {"ip": "127.64.0.9", "port": 20102, "access": "auto_reports", "poll_ms": 60000}
        else:
            ied["sv"] = {"publishes": [{"svcb_ref": "BUS1LD1/LLN0$MS$cb1", "app_id": 16385, "mac": "01-0C-CD-04-00-01",
                                        "smp_mod": 0, "smp_rate": 80, "layout": "9-2LE", "conf_rev": 1,
                                        "channels": ["Ia", "Ib"], "scale": [0.001, 0.001]}]}
    signals = [signal["ref"] for ied in service.project["ieds"] if ied["id"] == feed for signal in ied["signals"]
               if signal["fc"] == "ST" and signal["ref"].endswith(".stVal")]
    # One watched point: Python orders several through a set (hash order per process).
    service.project["test_panel"] = [{"ied_id": feed, "ref": signals[1]}]
    service.read_refs = signals[:6] + ["unknown"]
    service.project["network"].update(mode="lab", adapter_guid="{11111111-2222-3333-4444-555555555555}",
                                      adapter_label="Ethernet")
    return feed, bus


def running_scenario(script):
    feed, bus = prepare_run()
    project_step()
    coordinator.state = "running"
    coordinator.ied_indexes = {ied["id"]: {s["ref"]: s for s in ied["signals"]} for ied in service.project["ieds"]}
    del captured[:]
    live.start(service.project)
    deadline = time.monotonic() + 20
    while time.monotonic() < deadline and (live.sessions[feed].status["state"] != "live" or script.reads < 3):
        time.sleep(0.05)
    steps.append({"step": "start", "events": sorted(captured, key=json.dumps)})
    for event in scenario_events(feed, bus):
        inject(*event)
    for path in ("/live", "/goose", "/goose/links", "/mms/access", "/sv", "/health", "/health.csv",
                 "/health/report.html", "/goose/share", f"/live/{feed}/comparison", f"/live/{feed}/comparison.csv",
                 f"/live/{bus}/comparison", f"/ieds/{feed}/signals?limit=4&q=stval"):
        http("GET", "/api/v1" + path)
    http("POST", f"/api/v1/live/{feed}/read", {"refs": service.read_refs})
    http("POST", "/api/v1/health/refresh")
    http("POST", "/api/v1/goose/conflicts", {"choice": "stop", "ied_id": bus})
    http("GET", "/api/v1/goose/share")
    del captured[:]
    live.stop()
    steps.append({"step": "stop", "events": list(captured)})


def scenario_events(feed, bus):
    gocb = "FEED1LD1/LLN0$GO$Feed1SwiPosCtl"
    other = "FEED2LD1/LLN0$GO$Feed2SwiPosCtl"
    rx = {"event": "goose_rx", "key": gocb, "gocb_ref": gocb, "state": "receiving", "problem": "", "st_num": 4,
          "sq_num": 0, "frames": 3, "origin": "wire", "src_mac": "00-11-22-33-44-55", "conf_rev": 1, "go_id": "x",
          "dataset": "FEED1LD1/LLN0$GooseDs", "test": True, "values": [True, [1, False]]}
    return [
        ("goose", rx),
        ("goose", {**rx, "key": other, "gocb_ref": other, "state": "problem", "problem": "conf_rev_mismatch",
                   "test": False, "values": [1, 2]}),
        ("goose", {"event": "goose_stats", "watches": [{"key": gocb, "frames": 9, "age_ms": 120, "sq_num": 5},
                                                       {"key": "nope", "frames": 1, "age_ms": 1}]}),
        ("goose", {"event": "goose_unknown", "gocb_ref": "X/LLN0$GO$y", "app_id": 7, "mac": "01-0C-CD-01-00-99",
                   "test": True}),
        ("goose", {"event": "goose_unknown_cap", "limit": 50}),
        ("engine", {"event": "goose_conflict", "ied_id": bus, "gocb_ref": "BUS1LD1/LLN0$GO$Bus1TripEventCb",
                    "src_mac": "00-AA", "sharing": False}),
        ("engine", {"event": "goose_share", "state": "checking"}),
        ("engine", {"event": "goose_uplink", "state": "open", "message": "listening"}),
        ("engine", {"event": "goose_storm", "ied_id": bus, "limit_per_second": 500}),
        ("sv", {"event": "sv_rx", "key": "BUS1LD1/LLN0$MS$cb1", "state": "receiving", "asdus": 1, "rate": 4000,
                "conf_rev": 2, "smp_synch": 0, "lost": 1, "simulate": True, "sv_id": "bus1"}),
        ("sv", {"event": "sv_unknown", "app_id": 9, "mac": "m", "sv_id": "s", "simulate": True}),
        ("mms", {"clients": [{"peer": "127.0.0.1:50000", "reads": {"LLN0": 3, "XCBR1": 7}, "total_reads": 10,
                              "rcbs": ["b", "a"]}, {"peer": "10.0.0.5:1234", "writes": {"Pos": 1}}]}, bus),
        ("mms", {"clients": [{"peer": "10.0.0.5:1234", "connected": False}]}, bus),
    ]


def main():
    feed_signals = {s["ref"]: s for ied in service.project["ieds"] for s in ied["signals"]}
    feed = next(ied for ied in service.project["ieds"] if ied["name"] == "FEED1")
    signals = [s["ref"] for s in feed["signals"]]
    script = fakes.Script(feed_signals, fakes.discovery_for(feed, signals[3], signals[5], "FEED1LD1/EXTRA1.Ind.stVal"))
    npcap = patch(script)
    project_step()
    stopped_routes()
    running_scenario(script)
    result = {"version": __version__, "npcap": npcap, "adapters": fakes.ADAPTERS, "now": fakes.FIXED.isoformat(),
              "monotonic": fakes.Monotonic.value, "steps": steps, "script": script.log}
    common.OUT.mkdir(parents=True, exist_ok=True)
    (common.OUT / "routes_scenario.json").write_text(json.dumps(result), encoding="utf-8", newline="\n")
    print(f"routes_scenario.json: {len(steps)} steps, {len(script.log)} native requests")
    common.close(app, folder)


main()
