"""Golden cases for Go package acquisition/health (goose_consistency, health_model_checks,
station_health, health_export).

Run from this folder with the studio venv: python gen_health.py
Each want is json.dumps(result) (Python insertion order), or the text for csv/html.
"""
import copy
import json
from datetime import datetime

from common import project, record
from studio_app.domain import goose_consistency as gc, health_export, health_model_checks as hm, station_health as sh
from studio_app.domain import test_mode

NOW = "2026-10-20T08:00:00+00:00"
AT = "2026-10-20T07:59:50+00:00"


def roundtrip(value):
    return json.loads(json.dumps(value))


def case(name, function, *args, text=False, now_at=None):
    """Call function on a JSON copy of args; now_at (index) turns that string into a datetime."""
    args = roundtrip(list(args))
    call = roundtrip(args)
    if now_at is not None:
        call[now_at] = datetime.fromisoformat(call[now_at])
    result = function(*call)
    return name, args, result if text else json.dumps(result)


def sig(ref):
    return {"ref": ref, "fc": "ST"}


def control(name, **extra):
    return {"gocb_ref": f"R1/LLN0$GO${name}", "go_id": name.upper(), "dataset_ref": "R1/LLN0$DS1", "conf_rev": 3, **extra}


REAL = {"id": "r1", "name": "REAL, \"one\"", "role": "real",
        "signals": [sig(ref) for ref in ("R1/LLN0.Mod.stVal", "R1/LLN0.Beh.stVal", "R1/XCBR1.Beh.stVal", "R1/LPHD1.Sim.stVal",
                                          "R2/LLN0.Beh.stVal", "R3/LLN0.Mod.stVal", "R3/LPHD.Sim.stVal", "R4/LLN0.Beh.stVal")],
        "goose": {"publishes": [control("g1"), control("g2"), control("g3"), control("g4"), control("g5", conf_rev=9),
                                control("g6", go_id=None, dataset_ref=None)]},
        "reports": [{"ref": "R1/LLN0.BR.rep1", "dataset_ref": "R1/LLN0$DS1", "conf_rev": 1},
                    {"ref": "R1/LLN0.BR.rep2", "dataset_ref": "R1/LLN0$DS2", "conf_rev": 2},
                    {"ref": "R1/LLN0.RP.rep3", "dataset_ref": None}, {"ref": "R1/LLN0.RP.rep4", "dataset_ref": "X"},
                    {"ref": "R1/LLN0.RP.rep5", "dataset_ref": "Y", "conf_rev": 1}, {"ref": "R1/LLN0.RP.rep6", "dataset_ref": "Z"},
                    {"ref": "R1/LLN0.RP.rep7", "dataset_ref": "", "conf_rev": None}, {"ref": "R1/LLN0.RP.rep8", "conf_rev": 1.0}]}
SIM = {"id": "s1", "name": "SIM<1>", "role": "simulated", "signals": [sig("S1/LLN0.Beh.stVal")],
       "goose": {"publishes": [{"gocb_ref": "S1/LLN0$GO$gs", "go_id": "GS", "dataset_ref": "S1/LLN0$D", "conf_rev": 1}]}}
OFF = {"id": "o1", "name": "OFF", "role": "off", "signals": []}
NOROLE = {"id": "n1", "name": "NO ROLE é", "signals": []}
NULLROLE = {"id": "z1", "name": "NULL ROLE", "role": None, "signals": [sig("Z/LLN0.Beh.stVal")]}
BARE = {"id": "b1", "name": "BARE", "role": "real", "signals": [sig("B/LLN0.Mod.q")]}

GOCB_VALUES = {
    "R1/LLN0.g1.GoEna": True, "R1/LLN0.g1.GoID": "G1", "R1/LLN0.g1.DatSet": "R1/LLN0.DS1", "R1/LLN0.g1.ConfRev": 3,
    "R1/LLN0.g3.GoEna": True, "R1/LLN0.g3.GoID": "X", "R1/LLN0.g3.DatSet": "Other", "R1/LLN0.g3.ConfRev": 4.0,
    "R1/LLN0.g4.GoEna": False, "R1/LLN0.g4.GoID": "G4", "R1/LLN0.g4.DatSet": "R1/LLN0$DS1", "R1/LLN0.g4.ConfRev": 3,
    "R1/LLN0.g5.GoEna": False, "R1/LLN0.g5.ConfRev": 9.0, "R1/LLN0.g6.GoID": "", "R1/LLN0.g6.DatSet": "",
    "R1/LLN0.g6.ConfRev": 3}
BLOCKS = {"R1/LLN0.BR.rep1": {"readable": True, "dataset": "R1/LLN0.DS1", "conf_rev": 1},
          "R1/LLN0.BR.rep2": {"readable": True, "dataset": "R1/LLN0$DSX", "conf_rev": 5},
          "R1/LLN0.RP.rep3": {"readable": True}, "R1/LLN0.RP.rep4": {"readable": False, "dataset": "Q"},
          "R1/LLN0.RP.rep6": {}, "R1/LLN0.RP.rep7": {"readable": 1, "dataset": None, "conf_rev": 0},
          "R1/LLN0.RP.rep8": {"readable": True, "dataset": "", "conf_rev": 1}}
MODE_VALUES = {"R1/LLN0.Mod.stVal": 1, "R1/LLN0.Beh.stVal": 3, "R2/LLN0.Beh.stVal": 7, "R1/LPHD1.Sim.stVal": True,
               "R4/LLN0.Beh.stVal": True}


def health(rcbs=True, checked=AT, errors=None, values=None):
    blocks = {"blocks": BLOCKS, "checked_at": checked, "source": "mms_read"} if rcbs else None
    return {"rcbs": blocks, "gocbs": {"values": GOCB_VALUES, "errors": {}, "checked_at": checked, "source": "mms_read"},
            "mode": {"values": MODE_VALUES if values is None else values,
                     "errors": {"R3/LLN0.Mod.stVal": "denied"} if errors is None else errors,
                     "checked_at": checked, "source": "mms_read"}}


COMPARISONS = [
    {"summary": {"missing_on_device": 2, "type_mismatch": 0, "extra_on_device": 5},
     "datasets": [{"ref": "R1/LLN0$DS1", "status": "match"}, {"ref": "R1/LLN0$DS2", "status": "missing_on_device"},
                  {"ref": "R1/LLN0$DS3", "status": "members_differ", "missing": ["a", "b"], "extra": ["c"]}],
     "reports": [{"ref": "rep1", "status": "match"}, {"ref": "rep2", "status": "missing_on_device"}]},
    {"summary": {"missing_on_device": 0, "type_mismatch": 3, "extra_on_device": 0}, "datasets": [], "reports": []},
    {"summary": {"missing_on_device": 0, "type_mismatch": 0, "extra_on_device": 4},
     "datasets": [{"ref": "D", "status": "match"}], "reports": [{"ref": "r", "status": "other"}]},
    {"summary": {"missing_on_device": 0, "type_mismatch": 0, "extra_on_device": 0}, "datasets": [], "reports": []}]


def watch(gocb, ied_id, state, **extra):
    return {"gocb_ref": gocb, "publisher": {"ied_id": ied_id, "name": ied_id}, "state": state,
            "observed_at": AT, "conf_rev": 3, "go_id": "G", "dataset_ref": "R1/LLN0$DS1", **extra}


WATCHES = [watch("R1/LLN0$GO$g1", "r1", "receiving", frames=12), watch("R1/LLN0$GO$g2", "r1", "receiving", received_test=True),
           watch("R1/LLN0$GO$g3", "r1", "receiving"),
           watch("R1/LLN0$GO$g4", "r1", "problem", problem="conf_rev_mismatch", received_conf_rev=4),
           watch("R1/LLN0$GO$g5", "r1", "problem", problem="dataset_mismatch", received_dataset="R1/LLN0$X"),
           watch("R1/LLN0$GO$g6", "r1", "problem", problem="go_id_mismatch", received_go_id="Q"),
           watch("S1/LLN0$GO$gs", "s1", "problem", problem="invalid_frame"),
           watch("S1/LLN0$GO$g7", "s1", "problem", problem="needs_commission"),
           watch("S1/LLN0$GO$g8", "s1", "problem"), watch("S1/LLN0$GO$g9", "s1", "timeout"),
           watch("S1/LLN0$GO$g10", "s1", "timeout", problem="weird"), watch("S1/LLN0$GO$g11", "s1", "waiting"),
           watch("S1/LLN0$GO$g12", "s1", None, frames=3.5), dict(watch("X/LLN0$GO$x", "x", "receiving"), publisher=None),
           watch("Z/LLN0$GO$z", None, "receiving", frames=1.0)]
LISTENING = {"state": "listening", "message": "", "watches": WATCHES,
             "unknown": [{"gocb_ref": "U/LLN0$GO$u", "test": True, "mac": "01-0C"}, {"gocb_ref": "V"}], "unknown_limit": 64}
STOPPED = {"state": "stopped", "message": "Npcap is missing", "watches": WATCHES}
FAILED = {"state": "error", "watches": WATCHES[:3], "unknown": [{"gocb_ref": "W"}]}

SESSIONS = {
    "none": {}, "retrying": {"r1": {"state": "retrying", "message": "Timed out after 5 s"}},
    "no_address": {"r1": {"state": "no_address"}}, "connecting": {"r1": {"state": "connecting", "health": health()}},
    "live-no-health": {"r1": {"state": "live", "health": None}},
    "live-polling": {"r1": {"state": "live", "host": "10.0.0.5", "port": 102, "health": health(rcbs=False)}},
    "live-full": {"r1": {"state": "live", "host": "10.0.0.5", "port": 102, "health": health(), "comparison": COMPARISONS[0]},
                  "z1": {"state": "live", "health": health(values={}, errors={})}, "b1": {"state": "live", "health": health()}},
    "live-stale": {"r1": {"state": "live", "host": None, "health": health(checked="2026-10-20T07:58:00+00:00", errors={}),
                          "comparison": COMPARISONS[2]}},
    "live-naive": {"r1": {"state": "live", "health": health(checked="2026-10-20T07:00:00"), "comparison": COMPARISONS[3]}},
}


def health_cases():
    ieds = [REAL, SIM, OFF, NOROLE, NULLROLE, BARE]
    cases = []
    for label, sessions in SESSIONS.items():
        for goose_label, goose in (("listening", LISTENING), ("stopped", STOPPED)):
            cases.append(case(f"build_health/{label}-{goose_label}", sh.build_health, ieds, True, sessions, goose, NOW,
                              now_at=4))
    cases += [case("build_health/not-running", sh.build_health, ieds, False, SESSIONS["live-full"], {}, NOW, now_at=4),
              case("build_health/failed-listener", sh.build_health, ieds, True, {}, FAILED, NOW, now_at=4),
              case("build_health/empty", sh.build_health, [], True, {}, {}, NOW, now_at=4),
              case("build_health/offset-now", sh.build_health, [SIM], True, {}, {}, "2026-10-20T12:00:00.75+04:00", now_at=4),
              case("build_health/second-offset", sh.build_health, [OFF], False, {}, {}, "2026-10-20T08:00:00-05:30:15",
                   now_at=4)]
    for name in ("open", "model"):
        public = [dict(ied, signals=test_mode.mode_signals(ied["signals"])) for ied in project(name)["ieds"]]
        cases.append(case(f"build_health/public-{name}", sh.build_health, public, True, {}, {"state": "listening"}, NOW,
                          now_at=4))
    return cases


AGES = ["2026-10-20T07:59:00+00:00", "2026-10-20T07:59:45+00:00", "2026-10-20T07:59:30+00:00",
        "2026-10-20T07:59:29.999999+00:00", "2026-10-20T07:00:00", None, 12345, "", "garbage", "2026-10-20T08:00:00+0000",
        "2026-10-20T08:00:00Z", "2026-10-20 07:59:00+00:00", "2026-10-20x07:59:00+00:00", "2026-10-20T07:59+00:00",
        "2026-10-20T07+00:00", "2026-10-20T07:59:00.5+00:00", "2026-10-20T07:59:00,5+00:00",
        "2026-10-20T07:59:00.1234567+00:00", "20261020T075900+0000", "2026-10-20T075900Z", "2026-10-20T08:00:00+00",
        "2026-10-20T08:00:00+00:00:30", "2026-10-20T08:00:00+00:00:30.5", "2026-10-20", "2026-W43-2T08:00:00+00:00",
        "2026-10-20T8:00:00+00:00", "2026-10-20T08:00:00-05:00", "2026-10-20T07:59:00.123Z", "2026-10-20T24:00:00+00:00",
        "2026-10-20T08:00:00+24:00", "2026-10-20T08:00:00 +00:00", "2026-10-20T08:00:00.+00:00", "2026-10-20T08:00:60+00:00",
        "+2026-10-20T08:00:00+00:00", "2026-10-20T08:00:00.12+0530", "2026-10-20T08:00:00z", "2026-10-20t07:00:00+00:00",
        "2026-10-20T07:59:00.000000001+00:00", "2026-10-20T07Z", "2026-10-20T0759Z", "2026-10-20T08:0000Z",
        "2026-10-20T08:00:00.5", "2026-10-2008:00:00+00:00", "2026-1020T08:00:00+00:00", "202610-20T08:00:00+00:00",
        "2026-10-20T08:00:00+05", "2026-10-20T08:00:00+0530", "2026-10-20T08:00:00+05:30:15.123456",
        "2026-10-20T08:00:00-00:00", "２０２６-10-20T08:00:00+00:00", "2026-10-20T08:00:00+05:99",
        "2026-10-20T08:00:00x+00:00", "2026-10-20T", "2026-W53-1T00:00:00+00:00", "2020-W53-1T00:00:00+00:00",
        "2021-W53-1T00:00:00+00:00", "2026W432T08:00:00+00:00", "2026W43T08:00:00+00:00", "2026-W43T08:00:00+00:00",
        "2026-02-29T00:00:00+00:00", "2024-02-29T00:00:00+00:00", "0000-01-01T00:00:00+00:00", "0001-01-01T00:00:00+00:00",
        "9999-12-31T23:59:59.999999-23:59", "2026-10-20é08:00:00+00:00", "2026-10-20T08:00:00+00:00x",
        "2026-10-20T08:00:00Zx", "2026-10-20T08:00:00-", "20261020é08:00:00+00:00", "2026-10-20€07:59:00+00:00",
        "2026-10-20\U0001F60008:00:00+00:00", "2026-10-20T08:00:00é+00:00", "2026W43é08:00:00+00:00",
        "2026-W43-2é08:00:00+00:00", "2026-10-20T08:00:00.1234567é+00:00", "2026-10-20éé08:00:00+00:00",
        "2026-10-20T08:00:00:5+00:00", "2026-10-20T080000123+00:00", "2026-10-20T08.5+00:00", "2026-10-20T08:00:00.+05",
        "2026-W00-1T00:00:00+00:00", "2026-W01-8T00:00:00+00:00", "2026-W01-0T00:00:00+00:00", "9999-W52-6T00:00:00+00:00",
        "2026-13-01T00:00:00+00:00", "2026-04-31T00:00:00+00:00", "2026-10-20T07:59:00-00:00:00.000001", "2026W", "1234567",
        "2026-W5", "2026-W53", "2026-10-20T07:59:00+00:00\n", "\n2026-10-20T07:59:00+00:00", "2026-10-20T07:59:00+00:0"]


def direct_cases():
    cases = [case("result/defaults", gc.result, "area", "item", "ok", "fine"),
             case("result/full", gc.result, "a", {"x": 1}, "warning", "m", AT, "wire", {"k": [1, 2]}),
             case("result/empty-detail", gc.result, "a", None, "problem", "m", None, "scd", {})]
    for parts in ([], ["a"], ["dataset is x", "confRev is 1"], ["été"], ["ǆx"], ["1 thing"], [""]):
        cases.append(case(f"sentence/{len(cases)}", gc.sentence, parts))
    reads = {"values": GOCB_VALUES, "checked_at": AT}
    cases += [case("device_gocb_checks/real", gc.device_gocb_checks, REAL, reads),
              case("device_gocb_checks/sim", gc.device_gocb_checks, SIM, {"values": {}, "checked_at": None}),
              case("device_gocb_checks/no-goose", gc.device_gocb_checks, {"goose": None}, reads)]
    for number, listener in enumerate((LISTENING, STOPPED, FAILED, {"state": "listening"})):
        cases += [case(f"wire_check/{number}-{index}", gc.wire_check, item, listener)
                  for index, item in enumerate(WATCHES)]
    for status in (LISTENING, {}, {"unknown": [{"a": 1}], "unknown_limit": 0}, {"unknown_limit": "64"},
                   {"unknown": [], "unknown_limit": 2.5}):
        cases.append(case(f"unknown_publishers/{len(cases)}", gc.unknown_publishers, status))
    cases += [case(f"model_rows/{index}", hm.model_rows, REAL, item, AT) for index, item in enumerate(COMPARISONS)]
    cases += [case("report_setting_rows/none", hm.report_setting_rows, REAL, None),
              case("report_setting_rows/blocks", hm.report_setting_rows, REAL, {"blocks": BLOCKS, "checked_at": AT}),
              case("report_setting_rows/no-reports", hm.report_setting_rows, SIM, {"blocks": {}, "checked_at": AT}),
              case("report_setting_rows/unreadable", hm.report_setting_rows, REAL, {"blocks": {}, "checked_at": None})]
    modes = [health()["mode"], {"values": {}, "errors": {}, "checked_at": None},
             {"values": {"R1/LLN0.Mod.stVal": 2, "R1/LLN0.Beh.stVal": 5, "R2/LLN0.Beh.stVal": 4.0,
                         "R3/LPHD.Sim.stVal": 1, "R1/LPHD1.Sim.stVal": False, "R4/LLN0.Beh.stVal": "on"},
              "errors": ["R2/LLN0.Beh.stVal"], "checked_at": AT}]
    cases += [case(f"mode_rows/{index}", hm.mode_rows, REAL, mode) for index, mode in enumerate(modes)]
    cases += [case("mode_rows/none", hm.mode_rows, BARE, modes[0]), case("mode_rows/sim", hm.mode_rows, SIM, modes[1])]
    for results in ([], ["ok"], ["ok", "warning", "not_checked"], ["problem", "ok", "problem"], ["not_checked", "ok"]):
        cases.append(case(f"worst/{len(cases)}", sh.worst, results))
    cases += [case(f"age/{index}", sh._age, value, NOW, now_at=1) for index, value in enumerate(AGES)]
    return cases


def export_cases(built):
    tricky = {"verdict": "warning", "checked_at": "2026-10-20T08:00:00+00:00 <b>", "ieds": [
        {"name": "Feeder, \"A\"\nline2", "role": "real", "checks": [
            {"area": "mode", "item": "LD1/LLN0.Mod.stVal", "result": "problem", "message": "It's <bad> & \"odd\"\r\nyes",
             "observed_at": None, "source": "mms_read", "stale": True},
            {"area": "x", "item": {"a": 1, "b": [True, None]}, "result": "weird", "message": ["m", 1.5],
             "observed_at": 1700000000, "source": None, "stale": False},
            {"area": "é漢\U0001F600", "item": 2.5, "result": None, "message": "tab\tand space ", "observed_at": 1e20}]},
        {"name": "Empty", "role": "simulated", "checks": []}]}
    cases = [("csv_text/tricky", [tricky], health_export.csv_text(copy.deepcopy(tricky))),
             ("html_report/tricky", [tricky, "Station <&> \"Q\" 'x' é", "run-1 <x>"],
              health_export.html_report(copy.deepcopy(tricky), "Station <&> \"Q\" 'x' é", "run-1 <x>"))]
    for name, args, want in built:
        if name.endswith(("live-full-listening", "not-running", "public-model", "live-stale-stopped")):
            built_health = json.loads(want)
            cases.append((f"csv_text/{name}", [built_health], health_export.csv_text(copy.deepcopy(built_health))))
            cases.append((f"html_report/{name}", [built_health, "Demo", "abc-1"],
                          health_export.html_report(copy.deepcopy(built_health), "Demo", "abc-1")))
    return cases


if __name__ == "__main__":
    built = health_cases()
    record("health_build.json", built)
    record("health_checks.json", direct_cases())
    record("health_export.json", export_cases(built))
