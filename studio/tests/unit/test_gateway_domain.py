"""Gateway point proposal, addressing, validation, CSV and value sampling (spec gateway.md)."""
import pytest
from studio_app.domain.gateway_csv import export_csv, import_csv
from studio_app.domain.gateway_points import occupied, propose
from studio_app.domain.gateway_rules import normalize_gateway, validate_points, validate_settings
from studio_app.domain.gateway_values import display, sample
from studio_app.scl.importer import import_scl
from studio_app.scl.templates import make_template


@pytest.fixture(scope="module")
def ieds():
    simulated = import_scl(make_template("BayA"))[0]
    simulated.update(id="a", role="simulated", state="running")
    real = import_scl(make_template("BayB"))[0]
    real.update(id="b", role="real", state="real_live")
    return [simulated, real]


def test_proposal_maps_positions_statuses_measurements_and_commands(ieds):
    points, notes = propose(ieds, "iec104")
    assert not notes
    kinds = {(point["ied"], point["kind"]) for point in points}
    assert ("a", "double") in kinds and ("a", "float") in kinds and ("a", "command_double") in kinds
    assert ("b", "double") in kinds and ("b", "command_double") in kinds  # owner: real commands allowed
    position = next(point for point in points if point["ref"].endswith("XCBR1.Pos.stVal") and point["ied"] == "a")
    assert position["kind"] == "double" and 20001 <= position["address"] < 30001
    assert len({point["address"] for point in points}) == len(points)  # one IOA per object


def test_smart_selection_keeps_only_what_scada_needs():
    from studio_app.domain.gateway_selection import smart_kind
    leaf = lambda ref, kind, fc="ST": smart_kind({"ref": ref, "kind": kind, "fc": fc})
    assert leaf("LD0/QA1XCBR1.Pos.stVal", "Dbpos") == "double"
    assert leaf("LD0/XCBR1.Loc.stVal", "BOOLEAN") == "single"
    assert leaf("PROT/PTOC1.Op.general", "BOOLEAN") == "single"
    assert leaf("LD0/MMXU1.TotW.mag.f", "FLOAT32", "MX") == "float"
    assert leaf("LD0/LLN0.Health.stVal", "Enum") == "integer"
    for ref, kind, fc in (("LD0/LLN0.Mod.stVal", "BOOLEAN", "ST"), ("LD0/PTOC1.Health.stVal", "Enum", "ST"),
                          ("LD0/RDRE1.RcdMade.stVal", "BOOLEAN", "ST"), ("LD0/TCTR1.Amp.instMag.i", "INT32", "MX"),
                          ("LD0/XCBR1.Pos.Oper.ctlVal", "BOOLEAN", "CO"), ("LD0/PTOC1.StrVal.setMag.f", "FLOAT32", "SP")):
        assert leaf(ref, kind, fc) is None, ref


def test_spare_generic_inputs_stay_out_and_rank_last():
    from studio_app.domain.gateway_selection import dataset_refs, engineered, rank
    ied = {"datasets": [{"signal_refs": ["LD0/GGIO1.Ind1.stVal"]}]}
    used, spare = {"ref": "LD0/GGIO1.Ind1.stVal"}, {"ref": "LD0/GGIO1.Ind2.stVal", "description": ""}
    refs = dataset_refs(ied)
    assert engineered(used, refs) and not engineered(spare, refs)
    assert engineered(dict(spare, description="SF6 low"), refs) and engineered(spare, dataset_refs({}))
    assert engineered({"ref": "LD0/XCBR1.Loc.stVal"}, refs)  # only generic I/O needs evidence
    ordered = sorted([{"ref": "LD0/GGIO1.Ind1.stVal", "kind": "single"}, {"ref": "LD0/MMXU1.A.phsA.cVal.mag.f", "kind": "float"},
                      {"ref": "LD0/XCBR1.Pos.stVal", "kind": "double"}], key=rank)
    assert [item["kind"] for item in ordered] == ["double", "float", "single"]


def test_modbus_floats_take_two_registers_without_overlap(ieds):
    points, _ = propose(ieds, "modbus")
    cells = [cell for point in points for cell in occupied("modbus", point)]
    assert len(cells) == len(set(cells))
    floats = sorted(point["address"] for point in points if point["kind"] == "float")
    assert floats[0] == 1000 and floats[1] == 1002


def test_validation_explains_every_problem(ieds):
    points, _ = propose(ieds, "dnp3")
    clean, problems = validate_points("dnp3", points, ieds)
    assert len(clean) == len(points) and not problems
    real_command = {"kind": "command_single", "ied": "b", "ref": "BayBCTRL/XCBR1.Pos", "address": 99}
    clash = dict(points[0], address=points[1]["address"]) if points[0]["kind"] == points[1]["kind"] else None
    bad = [real_command, {"kind": "float", "ied": "a", "ref": "BayACTRL/XCBR1.Pos.stVal", "address": 7},
           {"kind": "single", "ied": "zz", "ref": "x", "address": 1}, {"kind": "single", "address": -1},
           {"kind": "float", "ied": "a", "ref": "BayACTRL/GGIO1.AnIn1.mag.f", "address": 3, "scale": 0}]
    clean, problems = validate_points("dnp3", bad + ([points[1], clash] if clash else []), ieds)
    text = " | ".join(problems)
    assert "cannot be sent as float" in text and not any("BayBCTRL/XCBR1.Pos" in problem for problem in problems)
    assert real_command["ref"] in {point["ref"] for point in clean}
    assert "IED is not in the project" in text and "from 0 to 65535" in text and "other than 0" in text
    if clash:
        assert "already used by point" in text


def test_settings_follow_the_protocol_rules():
    assert validate_settings("iec104", {"common_address": 5})["common_address"] == 5
    assert validate_settings("modbus", None)["port"] == 502
    for protocol, values, reason in (("iec104", {"t1": 5, "t2": 10}, "t2 must be shorter"),
                                     ("iec104", {"k": 2, "w": 8}, "w must not be larger"),
                                     ("modbus", {"unit_id": 300}, "from 0 to 255"),
                                     ("dnp3", {"port": "x"}, "whole number"), ("dnp3", {"foo": 1}, "Unknown")):
        with pytest.raises(ValueError, match=reason):
            validate_settings(protocol, values)


def test_csv_round_trip_and_line_diagnostics(ieds):
    points, _ = propose(ieds, "iec104")
    text = export_csv(points, ieds)
    assert text.startswith("type,address,ied,reference,scale,description")
    again, problems = import_csv(text, ieds)
    assert not problems and len(again) == len(points)
    key = lambda item: item["id"]
    assert sorted(validate_points("iec104", again, ieds)[0], key=key) == sorted(validate_points("iec104", points, ieds)[0], key=key)
    _, problems = import_csv(text + "bogus,1,BayA,x,1,\nsingle,2,Nobody,x,1,\nsingle,x,BayA,y,1,\n", ieds)
    assert [problem.split(":")[0] for problem in problems] == [f"Line {n}" for n in range(len(points) + 2, len(points) + 5)]
    with pytest.raises(ValueError, match="first row"):
        import_csv("a,b\n", ieds)


def test_sampling_marks_stopped_or_stale_sources_invalid(ieds):
    simulated = ieds[0]
    signal = next(item for item in simulated["signals"] if item["ref"].endswith("XCBR1.Pos.stVal"))
    index = {("a", item["ref"]): dict(item) for item in simulated["signals"]}
    index[("a", signal["ref"])]["value"] = 2
    point = {"id": "p", "kind": "double", "ied": "a", "ref": signal["ref"], "scale": 1.0}
    current = sample(point, index, simulated, "running")
    assert current["source_ok"] and current["value"] == 2
    assert display(point, current) == "ON (closed)"
    assert not sample(point, index, simulated, "stopped")["source_ok"]
    index[("a", signal["ref"])]["live_state"] = "stale"
    assert not sample(point, index, simulated, "running")["source_ok"]
    assert not sample(dict(point, ref="missing"), index, simulated, "running")["source_ok"]


def test_stored_gateway_keeps_points_of_removed_ieds():
    stored = {"protocol": "modbus", "listen": "network",
              "points": {"modbus": [{"kind": "single", "address": 3, "ied": "gone", "ref": "x"}]}}
    result = normalize_gateway(stored)
    assert result["points"]["modbus"][0]["ied"] == "gone" and result["settings"]["iec104"]["port"] == 2404
    assert normalize_gateway(None)["protocol"] == "iec104"
    with pytest.raises(ValueError, match="Unknown gateway protocol"):
        normalize_gateway({"protocol": "opc"})
