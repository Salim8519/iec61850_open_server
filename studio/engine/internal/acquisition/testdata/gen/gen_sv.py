"""Golden cases for Go package svdom (domain/sv_supervision.py). Writes sv_supervision.json.

The public projects publish no Sampled Values, so the streams are synthetic and cover
int and float arithmetic (4000 vs 4000.0), every smp_mod, the stream limit and every
problem of `assess`. `want` is json.dumps of the Python result (insertion order kept).
"""
import copy
import json

from common import project, record
from studio_app.domain import sv_supervision as sv


def case(name, function, *args):
    return name, list(args), json.dumps(function(*copy.deepcopy(args)))


def stream(number, smp_mod=0, smp_rate=80, layout="9-2LE", **extra):
    return {"svcb_ref": f"MU{number}LD0/LLN0$MS$cb{number}", "app_id": 0x4000 + number,
            "mac": f"01-0C-CD-04-00-{number:02X}", "smp_mod": smp_mod, "smp_rate": smp_rate, "layout": layout,
            "conf_rev": 1, "channels": ["Ia", "Ib"], "scale": [0.001, 0.001], **extra}


RATES = [(0, 80), (0, 256), (1, 4000), (1, 4800), (1, 4000.0), (1, 0), (2, 1), (None, 80), (True, 4000),
         (0, 80.0), (1, 4001), (0, True), (1, -4000), (1.0, 12000), (0.0, 96)]
NOMINALS = [50, 60, 50.0, 16.7, True, -50]


def rate_cases():
    cases = []
    for smp_mod, smp_rate in RATES:
        for nominal in NOMINALS:
            item = {"smp_mod": smp_mod, "smp_rate": smp_rate}
            name = f"{smp_mod}/{smp_rate}/{nominal}"
            cases.append(case(f"expected_rate/{name}", sv.expected_rate, item, nominal))
            cases.append(case(f"samples_per_cycle/{name}", sv.samples_per_cycle, item, nominal))
    return cases


def watch_cases():
    many = [stream(number, *RATES[number % len(RATES)][:2]) for number in range(20)]
    ieds = [{"name": "MU1", "role": "simulated", "sv": {"publishes": many[:5]}},
            {"name": "MU2", "role": "off", "sv": {"publishes": many[5:8]}},
            {"name": "MU3", "sv": None}, {"name": "MU4"},
            {"name": "MU5", "role": "real", "sv": {"publishes": many[8:]}},
            {"name": "MU6", "sv": {"publishes": [stream(30, key="old", state="x", wrap=5, extra=[1])]}}]
    cases = []
    for nominal in (50, 60.0):
        for count in (0, 1, 3, 5, 6):
            cases.append(case(f"watch_list/{count}/{nominal}", lambda items, hz: list(sv.watch_list(items, hz)),
                              ieds[:count], nominal))
    real = [{"name": ied["name"], "role": ied.get("role"), "sv": ied.get("sv")} for ied in project("model")["ieds"]]
    cases.append(case("watch_list/model", lambda items, hz: list(sv.watch_list(items, hz)), real, 50))
    return cases


def native_cases():
    cases = []
    for layout in ("9-2LE", "raw", None):
        for per_cycle in (0, 80, -1, 80.0, 0.5, True):
            watch = {"key": "K", "app_id": 0x4000, "mac": "01-0C-CD-04-00-01", "layout": layout,
                     "per_cycle": per_cycle, "wrap": 4000, "channels": ["Ia"], "scale": [0.01]}
            cases.append(case(f"native_stream/{layout}/{per_cycle}", sv.native_stream, watch))
    return cases


EVENTS = [
    {}, {"state": "receiving"}, {"state": "receiving", "asdus": 0, "conf_rev": 9},
    {"state": "receiving", "asdus": 10, "conf_rev": 1, "data_size": 64, "rate": 4000, "smp_synch": 1},
    {"state": "receiving", "asdus": 10, "conf_rev": 2, "data_size": 32, "rate": 4100, "smp_synch": 0,
     "lost": 3, "invalid_quality": 2, "simulate": True},
    {"state": "receiving", "asdus": 1, "conf_rev": 1.0, "data_size": 64.0, "rate": 4080, "smp_synch": False},
    {"state": "receiving", "asdus": 1, "conf_rev": 1, "data_size": None, "rate": 4080.5, "smp_synch": 2},
    {"state": "receiving", "asdus": 1, "conf_rev": True, "rate": 3919.9, "lost": 1, "invalid_quality": 0},
    {"state": "receiving", "asdus": 1, "conf_rev": 1, "rate": 0, "lost": None, "simulate": 0},
    {"state": "timeout", "asdus": 5, "conf_rev": 7, "rate": 1, "lost": 10},
    {"state": None, "asdus": [1], "conf_rev": 1, "rate": 4000.0},
    {"asdus": 2, "conf_rev": 1, "rate": 50, "smp_synch": 0.0, "invalid_quality": 5},
]
WATCHES = [
    {"conf_rev": 1, "layout": "9-2LE", "expected_rate": 4000},
    {"conf_rev": 1, "layout": "raw", "expected_rate": 4000.0},
    {"conf_rev": 1, "layout": "9-2LE", "expected_rate": None},
    {"conf_rev": 1, "layout": "9-2LE", "expected_rate": 0},
    {"conf_rev": 1, "layout": "9-2LE"},
    {"conf_rev": 2, "layout": "9-2LE", "expected_rate": 4096},
]
PREVIOUS = [None, {}, {"lost": 3, "invalid_quality": 2}, {"lost": 0.5, "invalid_quality": None}]


def assess_cases():
    cases = []
    for event_number, event in enumerate(EVENTS):
        for watch_number, watch in enumerate(WATCHES):
            for previous_number, previous in enumerate(PREVIOUS):
                name = f"assess/{event_number}/{watch_number}/{previous_number}"
                cases.append(case(name, sv.assess, watch, event, previous))
    return cases


if __name__ == "__main__":
    record("sv_supervision.json", rate_cases() + watch_cases() + native_cases() + assess_cases())
