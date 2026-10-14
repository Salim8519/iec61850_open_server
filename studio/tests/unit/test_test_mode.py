"""Test mode, behaviour and simulation (spec TMODE)."""
import pytest
from studio_app.domain.health_model_checks import mode_rows
from studio_app.domain.mode_badges import mode_badges, simulation_sources
from studio_app.domain.test_mode import derive_beh, mode_summary, mode_writes, validate_test_mode

SIGNALS = [{"ref": ref, "fc": "ST"} for ref in (
    "P1LD0/LLN0.Mod.stVal", "P1LD0/LLN0.Beh.stVal", "P1LD0/PTOC1.Beh.stVal", "P1LD0/PTOC1.Mod.stVal",
    "P1LD0/XCBR1.Beh.stVal", "P1LD0/LPHD1.Sim.stVal")]
IED = {"id": "i1", "name": "P1", "role": "simulated", "signals": SIGNALS}


def test_mode_values_have_standard_names():
    rows = {row["item"]: row for row in mode_rows(IED, {"values": {
        "P1LD0/LLN0.Mod.stVal": 1, "P1LD0/LLN0.Beh.stVal": 4, "P1LD0/LPHD1.Sim.stVal": False},
        "errors": {}, "checked_at": "t"})}
    assert rows["P1LD0/LLN0.Mod.stVal"]["result"] == "ok"
    assert "test/blocked" in rows["P1LD0/LLN0.Beh.stVal"]["message"]
    odd = mode_rows(IED, {"values": {"P1LD0/LLN0.Mod.stVal": 9}, "errors": {}, "checked_at": "t"})[0]
    assert odd["result"] == "warning" and "9" in odd["message"] and "not a valid mode" in odd["message"]
    assert mode_summary(IED, {"P1LD0/LLN0.Beh.stVal": 3})["devices"] == {"P1LD0": "test"}


def test_missing_mode_is_not_on():
    bare = {"id": "i2", "name": "P2", "signals": [{"ref": "P2LD0/GGIO1.Ind1.stVal", "fc": "ST"}]}
    row = mode_rows(bare, {"values": {}, "errors": {}, "checked_at": "t"})[0]
    assert row["result"] == "not_checked" and "not available" in row["message"]
    assert mode_summary(IED, {})["devices"] == {"P1LD0": "not available"}
    assert mode_summary(IED, {})["worst"] is None and mode_summary(IED, {})["simulation"] is None


def test_beh_is_derived_from_mod():
    assert [derive_beh(mode) for mode in (1, 2, 3, 4, 5)] == [1, 2, 3, 4, 5]
    assert derive_beh(3, 2) == 4 and derive_beh(2, 3) == 4 and derive_beh(1, 5) == 5 and derive_beh(4, 1) == 4
    writes = dict(mode_writes(IED, validate_test_mode({"devices": {"P1LD0": 3}, "simulation": True}),
                              {"P1LD0/PTOC1.Mod.stVal": 2}))
    assert writes["P1LD0/LLN0.Mod.stVal"] == 3 and writes["P1LD0/LLN0.Beh.stVal"] == 3
    assert writes["P1LD0/PTOC1.Beh.stVal"] == 4 and writes["P1LD0/XCBR1.Beh.stVal"] == 3
    assert writes["P1LD0/LPHD1.Sim.stVal"] is True
    assert "P1LD0/PTOC1.Mod.stVal" not in writes
    for wrong in ({"devices": {"P1LD0": 0}}, {"devices": {"P1LD0": True}}, {"simulation": "yes"}):
        with pytest.raises(ValueError):
            validate_test_mode(wrong)


def test_simulation_traffic_warns():
    goose = {"watches": [{"gocb_ref": "P1LD0/LLN0$GO$g1", "received_test": True, "state": "receiving",
                          "publisher": {"name": "P1"}}],
             "unknown": [{"gocb_ref": "X/LLN0$GO$t", "test": True}, {"gocb_ref": "Y/LLN0$GO$n", "test": False}]}
    sv = {"watches": [{"key": "MU/LLN0.sv", "sv_id": "MU01", "publisher": "MU", "problems": ["simulation"]}],
          "unknown": [{"sv_id": "TEST", "simulate": True}]}
    sources = simulation_sources(goose, sv)
    assert len(sources) == 4 and any("P1" in item for item in sources) and any("MU01" in item for item in sources)
    assert not any("Y/LLN0" in item for item in sources)
    assert simulation_sources({}, {}) == []


def test_equipment_badges_follow_bound_logical_device():
    ied = {**IED, "test_mode": {"devices": {"P1LD0": 4}, "simulation": False}}
    node = {"id": "cb", "data": {"state_mapping": {"mode": "direct", "source": {"ied_id": "i1",
                                                                              "ref": "P1LD0/XCBR1.Pos.stVal"}}}}
    badges = mode_badges([ied], {}, {"nodes": [node]}, running=False)
    assert badges["equipment"] == {"cb": "test/blocked"} and badges["ieds"]["i1"]["worst"] == "test/blocked"
    normal = mode_badges([{**IED, "test_mode": {"devices": {}, "simulation": False}}], {}, {"nodes": [node]}, False)
    assert normal["equipment"] == {}
