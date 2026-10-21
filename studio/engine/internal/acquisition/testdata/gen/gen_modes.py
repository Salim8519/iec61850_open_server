"""Golden cases for Go package acquisition/modes (health_reads, test_mode, mode_badges).

Run from this folder with the studio venv: python gen_modes.py
Each want is json.dumps(result) (Python insertion order) or "error:<ValueError text>".
"""
import json

from common import project, record
from studio_app.domain import health_reads, mode_badges, test_mode


def roundtrip(value):
    return json.loads(json.dumps(value))


def case(name, function, *args):
    """Call function on a JSON copy of args, exactly as the Go test decodes them."""
    args = roundtrip(list(args))
    try:
        want = json.dumps(function(*roundtrip(args)))
    except ValueError as error:
        want = "error:" + str(error)
    return name, args, want


def sig(ref, fc="ST"):
    return {"ref": ref, "fc": fc}


def public_ieds():
    """The public IEDs with their full point list reduced to ref/fc (what the rules read)."""
    ieds = []
    for name in ("open", "model"):
        for ied in project(name)["ieds"]:
            ieds.append({"id": ied["id"], "name": ied["name"], "role": ied["role"], "goose": ied["goose"],
                         "signals": [sig(item["ref"], item["fc"]) for item in ied["signals"]]})
    return ieds


SYN = {"id": "syn", "name": "SYN", "role": "simulated", "signals": [
    sig("LD1/LLN0.Mod.stVal"), sig("LD1/LLN0.Beh.stVal"), sig("LD1/XCBR1.Mod.stVal"), sig("LD1/XCBR1.Beh.stVal"),
    sig("LD1/PTOC1.Beh.stVal"), sig("LD1/LPHD1.Sim.stVal"), sig("LD1/LPHD.Sim.stVal"), sig("LD1/LLN0.Mod.q"),
    sig("LD2/LLN0.Beh.stVal"), sig("LD2/GGIO1.Beh.stVal"), sig("LD2/LPHD12.Sim.stVal"),
    sig("LD3/LLN0.Mod.stVal"), sig("LD3/LLN0.Beh.stVal", "MX"), sig("LD9/LLN0.Mod.stVal", "CF"),
    sig("LD4/LPHD1.Sim.stVal\n"), sig("LD5/X.Y.Beh.stVal"), sig("LD6/LPHD٣.Sim.stVal"), sig("LD6/LPHDx.Sim.stVal"),
    sig("LD7/A/B.Beh.stVal"), sig("/LLN0.Beh.stVal"), sig("LD8/LLN0.Beh.stVal\n"), sig("LD8/LLN0.Mod.stVal"),
    sig("LDx/.Beh.stVal"), sig("LD1/LLN0.Beh.stVal.extra"), sig("NOSLASH.Beh.stVal"), sig("LD1/XCBR1.Pos.stVal"),
    sig("LDé/LLN0.Beh.stVal"), sig("LDé/LLN0.Mod.stVal"), sig("LD1/LLN0.Beh.stVal"), sig("LD1/LPHD2.Sim.stVal", "MX")],
    "goose": {"publishes": [
        {"gocb_ref": "LD1/LLN0$GO$gcb1", "go_id": "G1", "dataset_ref": "LD1/LLN0$DS1", "conf_rev": 3},
        {"gocb_ref": "LD2/LLN0$GO$x$GO$y", "go_id": "G2", "dataset_ref": "LD2/LLN0$DS2", "conf_rev": 1},
        {"gocb_ref": "plain", "go_id": None, "dataset_ref": None, "conf_rev": None}]}}


def reads_cases():
    cases = [case("gocb_object/" + ref, health_reads.gocb_object, ref)
             for ref in ("LD/LLN0$GO$gcb", "LD/LLN0$GO$a$GO$b", "plain", "", "$GO$", "é$GO$é")]
    ieds = public_ieds() + [SYN, {"id": "none"}, {"goose": None}, {"goose": {}}, {"goose": {"publishes": []}}]
    cases += [case(f"gocb_items/{index}", health_reads.gocb_items, ied) for index, ied in enumerate(ieds)]
    signal_ieds = public_ieds() + [SYN, {"signals": []}, {"signals": [sig("A"), sig("B/LLN0.Mod.stVal", "CO")]}]
    cases += [case(f"mode_items/{index}", health_reads.mode_items, ied) for index, ied in enumerate(signal_ieds)]
    answers = [
        {"values": [{"ref": "a", "value": 1}, {"ref": "b", "structure": {"b.x": 2, "a": 9}},
                    {"ref": "c", "error": "denied"}, {"ref": "d"}, {"ref": "a", "value": None},
                    {"ref": "e", "value": [1, {"k": 2.5}], "error": "ignored"}, {"ref": "c", "error": None}]},
        {}, {"values": []}, {"values": [{"ref": "x", "structure": {}}]}]
    cases += [case(f"parse_answer/{index}", health_reads.parse_answer, answer) for index, answer in enumerate(answers)]
    for value in (0, 1, 2, 3, 4, 5, 6, -1, True, False, 3.0, "3", None, 2 ** 70, [1], {"a": 1}):
        cases.append(case(f"mode_name/{json.dumps(value)}", health_reads.mode_name, value))
    return cases


def test_mode_cases():
    cases = [case("mode_signals/feed1-full", test_mode.mode_signals, project("model")["ieds"][0]["signals"]),
             case("mode_signals/syn", test_mode.mode_signals, SYN["signals"]),
             case("mode_signals/empty", test_mode.mode_signals, [])]
    values = [1, 2, 3, 4, 5, 3.0, True, None, "x", 0]
    cases += [case(f"derive_beh/{json.dumps(a)}-{json.dumps(b)}", test_mode.derive_beh, a, b) for a in values for b in values]
    ieds = public_ieds() + [SYN, {"signals": []}]
    cases += [case(f"mode_points/{index}", test_mode.mode_points, ied) for index, ied in enumerate(ieds)]
    many = {f"LD{index}": 1 for index in range(101)}
    settings = [None, {}, [], 0, "", {"devices": None}, {"devices": []}, {"devices": [1]}, {"devices": "x"},
                {"devices": many}, {"devices": dict(list(many.items())[:100])}, {"devices": {"LD1": True}},
                {"devices": {"LD1": False}}, {"devices": {"LD1": 0}}, {"devices": {"LD1": 6}}, {"devices": {"LD1": 3.0}},
                {"devices": {"LD1": 3.5}}, {"devices": {"LD1": "3"}}, {"devices": {"LD1": None}}, {"devices": {"LD1": 2 ** 70}},
                {"devices": {"LD1": 2, "LD2": 5}, "simulation": True}, {"simulation": None}, {"simulation": "yes"},
                {"simulation": 1}, {"simulation": False, "extra": 1}, {"devices": {"LD1": 1, "LD2": "x"}, "simulation": True},
                {"devices": {"LD1": 4}, "simulation": 0}]
    cases += [case(f"validate_test_mode/{index}", test_mode.validate_test_mode, value) for index, value in enumerate(settings)]
    current = {"LD1/XCBR1.Mod.stVal": 2, "LD1/PTOC1.Mod.stVal": 7, "LD2/GGIO1.Mod.stVal": True,
               "LD7/A/B.Mod.stVal": 3.0, "LDé/LLN0.Mod.stVal": 4, "LD2/LLN0.Mod.stVal": "x"}
    writes = [
        ({"devices": {"LD1": 3, "LD2": 5, "LD8": 2, "LD7": 4, "LDé": 1, "UNKNOWN": 3}, "simulation": True}, current),
        ({"devices": {"LD1": 4.0, "LD3": 2}, "simulation": False}, current),
        ({"devices": {}, "simulation": True}, {}),
        (test_mode.validate_test_mode(None), {}),
        ({"devices": {"LD1": 1, "LD2": 1}, "simulation": False}, {"LD1/XCBR1.Mod.stVal": 5, "LD1/PTOC1.Mod.stVal": 3}),
    ]
    cases += [case(f"mode_writes/syn{index}", test_mode.mode_writes, SYN, s, c) for index, (s, c) in enumerate(writes)]
    feed = public_ieds()[2]
    cases.append(case("mode_writes/feed1", test_mode.mode_writes, feed, {"devices": {"FEED1LD1": 3}, "simulation": True}, {}))
    summaries = [{}, {"LD1/LLN0.Beh.stVal": 3, "LD2/LLN0.Mod.stVal": 2, "LD3/LLN0.Beh.stVal": None, "LD3/LLN0.Mod.stVal": 5,
                      "LD1/LPHD1.Sim.stVal": False, "LD2/LPHD12.Sim.stVal": True, "LD1/LPHD.Sim.stVal": 1},
                 {"LD1/LLN0.Beh.stVal": 1, "LD2/LLN0.Beh.stVal": 1, "LD1/LPHD1.Sim.stVal": False, "LD6/LPHD٣.Sim.stVal": False},
                 {"LD1/LLN0.Beh.stVal": True, "LD2/LLN0.Beh.stVal": 2.0, "LD7/LLN0.Beh.stVal": 9, "LD8/LLN0.Beh.stVal": "x",
                  "LDé/LLN0.Mod.stVal": 4, "LD1/LPHD1.Sim.stVal": "true"},
                 {"LD1/LLN0.Beh.stVal": 5, "LD2/LLN0.Beh.stVal": 4, "LD8/LLN0.Mod.stVal": 5, "LDé/LLN0.Beh.stVal": 2}]
    cases += [case(f"mode_summary/syn{index}", test_mode.mode_summary, SYN, values) for index, values in enumerate(summaries)]
    cases.append(case("mode_summary/feed1", test_mode.mode_summary, feed, {"FEED1LD1/LLN0.Beh.stVal": 3}))
    return cases


def small(ied_id, role, refs, **extra):
    return {"id": ied_id, "name": ied_id.upper(), "role": role, "signals": [sig(ref) for ref in refs], **extra}


def badge_ieds():
    syn = dict(SYN, test_mode={"devices": {"LD1": 3, "LD2": 1, "LD7": 4.0}, "simulation": True})
    real = ["R1/LLN0.Beh.stVal", "R1/XCBR1.Beh.stVal", "R1/LPHD1.Sim.stVal"]
    return [syn, small("sim2", "simulated", ["S2/LLN0.Beh.stVal", "S2/LLN0.Mod.stVal"]),
            small("sim3", "simulated", ["S3/LLN0.Beh.stVal", "S3b/LLN0.Beh.stVal"], test_mode={"devices": {"S3": True}}),
            small("sim4", "simulated", ["S4/LLN0.Beh.stVal"], test_mode={"devices": {"S4": 5}, "simulation": 1}),
            small("sim5", "simulated", ["S5/LLN0.Beh.stVal", "S5b/X.Beh.stVal"], test_mode={}),
            small("real1", "real", real), small("real2", "real", real), small("real3", "real", real),
            small("real4", "real", ["R4/LLN0.Beh.stVal"]), small("off1", "off", ["O/LLN0.Beh.stVal"]),
            {"id": "norole", "name": "NOROLE", "signals": [sig("N/LLN0.Beh.stVal")]}]


SESSIONS = {"real1": {"state": "live", "health": {"mode": {"values": {"R1/LLN0.Beh.stVal": 2, "R1/LPHD1.Sim.stVal": True}}}},
            "real2": {"state": "connecting", "health": None}, "real4": {"health": {"mode": {"values": {}}}},
            "norole": {"health": {"mode": {"values": {"N/LLN0.Beh.stVal": 3}}}}, "x": None}


def node(node_id, **data):
    return {"id": node_id, "data": data}


NODES = [node("n1", control_binding={"ied_id": "syn", "ref": "LD1/XCBR1.Pos"}),
         node("n2", state_mapping={"source": {"ied_id": "syn", "ref": "LD2/GGIO1.Ind"}}),
         node("n3", state_mapping={"source": {}, "open": {"ied_id": "real1", "ref": "R1/XCBR1.Pos"}}),
         node("n4", control_binding={"ied_id": "", "ref": "LD1/x"}, state_mapping={"source": {"ied_id": "syn", "ref": "LD7/A"}}),
         node("n5", control_binding={"ied_id": "syn", "ref": "noslash"}),
         node("n6", control_binding={"ied_id": "real4", "ref": "R4/XCBR1.Pos"}),
         node("n7", control_binding={"ied_id": "ghost", "ref": "G/XCBR1.Pos"}),
         node("n8", control_binding={"ied_id": "sim2", "ref": "S2/XCBR1.Pos"}),
         node("n9", control_binding=None, state_mapping=None), node("n10", control_binding={"ied_id": "syn", "ref": "LDx/a"}),
         node(5, control_binding={"ied_id": "sim4", "ref": "S4/a/b"}), node("n12", control_binding={"ied_id": 7, "ref": "R1/a"}),
         node("n13", control_binding={"ied_id": "sim3", "ref": "S3/a"}), node("n14", control_binding={"ied_id": "sim3", "ref": "S3b/a"}),
         node("n15", control_binding={"ied_id": "norole", "ref": "N/a"}), node("n16", control_binding={"ied_id": "syn", "ref": "/a"}),
         node("n17", control_binding={"ied_id": "sim5", "ref": "S5b/a"}), node("n1", control_binding={"ied_id": "real1", "ref": "R1/q"}),
         node("n18", state_mapping={"open": {"ied_id": "syn", "ref": "LDé/x"}}), node("n19", control_binding={})]


def badge_cases():
    cases = [case("mode_badges/running", mode_badges.mode_badges, badge_ieds(), SESSIONS, {"nodes": NODES}, True),
             case("mode_badges/stopped", mode_badges.mode_badges, badge_ieds(), SESSIONS, {"nodes": NODES}, False),
             case("mode_badges/no-diagram", mode_badges.mode_badges, badge_ieds(), {}, {}, True),
             case("mode_badges/public", mode_badges.mode_badges, public_ieds(), {}, {"nodes": []}, True),
             case("mode_badges/empty", mode_badges.mode_badges, [], {}, {}, False)]
    goose = {"watches": [
        {"gocb_ref": "A/LLN0$GO$g1", "state": "receiving", "received_test": True, "publisher": {"name": "IED_A"}},
        {"gocb_ref": "B/LLN0$GO$g2", "state": "problem", "received_test": 1, "publisher": None},
        {"gocb_ref": "C/LLN0$GO$g3", "state": "waiting", "received_test": True, "publisher": {"name": "C"}},
        {"gocb_ref": "D/LLN0$GO$g4", "state": "receiving", "received_test": False, "publisher": {"name": "D"}},
        {"gocb_ref": "E/LLN0$GO$g5", "state": "timeout", "received_test": True},
        {"gocb_ref": 12, "state": "receiving", "received_test": "yes", "publisher": {"name": ""}},
        {"gocb_ref": "F", "state": "problem", "received_test": True, "publisher": {"name": 3.5}}],
        "unknown": [{"gocb_ref": "X/LLN0$GO$u", "test": True}, {"gocb_ref": "Y", "test": False}, {"test": 1}, {}]}
    sv = {"watches": [
        {"key": "k1", "sv_id": "SV1", "publisher": "MU1", "problems": ["simulation", "timeout"]},
        {"key": "k2", "sv_id": "", "publisher": {"name": "MU2", "ip": None}, "problems": ["simulation"]},
        {"key": "k3", "sv_id": "SV3", "publisher": None, "problems": ["timeout"]},
        {"key": "k4", "sv_id": "SV4", "publisher": "MU4"}, {"key": "k5", "problems": "a simulation b"},
        {"key": "k6", "sv_id": None, "problems": {"simulation": True}}],
        "unknown": [{"sv_id": "U1", "simulate": True}, {"sv_id": "", "app_id": 16384, "simulate": True},
                    {"sv_id": "U3", "simulate": False}, {"app_id": None, "simulate": 1}]}
    cases += [case("simulation_sources/all", mode_badges.simulation_sources, goose, sv),
              case("simulation_sources/empty", mode_badges.simulation_sources, {}, {}),
              case("simulation_sources/goose-only", mode_badges.simulation_sources, goose, {"watches": []})]
    return cases


if __name__ == "__main__":
    record("modes_reads.json", reads_cases())
    record("modes_test_mode.json", test_mode_cases())
    record("modes_badges.json", badge_cases())
