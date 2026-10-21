"""Health reads only Mod/Beh/Sim points, so polls never copy the full model (change 0110)."""
from studio_app.acquisition.health_snapshot import health_view
from studio_app.domain.health_reads import mode_items
from studio_app.domain.mode_badges import mode_badges
from studio_app.domain.test_mode import mode_points, mode_signals, mode_summary

REFS = [("P1LD0/LLN0.Mod.stVal", "ST"), ("P1LD0/LLN0.Beh.stVal", "ST"), ("P1LD0/LPHD1.Sim.stVal", "ST"),
        ("P1LD0/XCBR1.Beh.stVal", "ST"), ("P1LD0/XCBR1.Mod.stVal", "ST"), ("P1LD0/XCBR1.Pos.stVal", "ST"),
        ("P1LD0/MMXU1.TotW.mag.f", "MX"), ("PROT/LLN0.Mod.ctlModel", "CF"), ("PROT/PTOC1.Beh.stVal", "ST"),
        ("PROT/PTOC1.Str.general", "ST"), ("CTRL/LLN0.Mod.stVal", "ST"), ("CTRL/CSWI1.Pos.stVal", "ST")]


def ied(signals=None):
    signals = signals or [{"ref": ref, "fc": fc, "value": 0, "live_state": "fresh"} for ref, fc in REFS]
    return {"id": "i1", "name": "P1", "role": "simulated", "source": "<SCL/>", "cfg": {"big": 1},
            "signals": signals, "test_mode": {"devices": {"P1LD0": 3}, "simulation": True}, "reports": [],
            "goose": {"publishes": []}}


def test_mode_signals_give_the_same_mode_answers_as_the_full_model():
    full = ied()
    small = {**full, "signals": mode_signals(full["signals"])}
    assert len(small["signals"]) < len(full["signals"])
    assert mode_points(small) == mode_points(full)
    assert mode_items(small) == mode_items(full)
    values = {"P1LD0/LLN0.Beh.stVal": 3, "CTRL/LLN0.Mod.stVal": 5, "P1LD0/LPHD1.Sim.stVal": True}
    assert mode_summary(small, values) == mode_summary(full, values)
    diagram = {"nodes": [{"id": "n1", "data": {"control_binding": {"ied_id": "i1", "ref": "P1LD0/XCBR1"}}}]}
    assert mode_badges([small], {}, diagram, True) == mode_badges([full], {}, diagram, True)


def test_health_view_leaves_out_the_model_and_follows_a_new_model():
    original = ied()
    view = health_view(original)
    assert "source" not in view and "cfg" not in view
    assert all(set(signal) == {"ref", "fc"} for signal in view["signals"])
    assert view["test_mode"] == original["test_mode"] and view["test_mode"] is not original["test_mode"]
    assert health_view(original)["signals"] is view["signals"], "same model: the small list is reused"
    replaced = {**original, "signals": [{"ref": "NEW/LLN0.Mod.stVal", "fc": "ST"}]}
    assert [signal["ref"] for signal in health_view(replaced)["signals"]] == ["NEW/LLN0.Mod.stVal"]
