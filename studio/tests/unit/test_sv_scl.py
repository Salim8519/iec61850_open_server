"""Sampled Values streams come only from the SCD (SV-02, SV-07)."""
from pathlib import Path

from studio_app.scl.importer import import_scl

SCD = (Path(__file__).resolve().parents[3] / "scd/open_substation.scd").read_text(encoding="utf-8")


def load(text):
    return import_scl(text.encode("utf-8"), "test.scd", ["IED3_SMV"])[0]


def test_control_without_address_is_diagnosed():
    stream = load(SCD)["sv"]["publishes"][0]
    assert (stream["mac"], stream["app_id"], stream["sv_id"], stream["smp_rate"], stream["smp_mod"]) == (
        "01-0C-CD-01-00-03", 0x4000, "xxxxMUnn01", 80, 0)
    assert stream["layout"] == "9-2LE" and len(stream["channels"]) == 8
    assert stream["scale"] == [0.001] * 4 + [0.01] * 4
    renamed = load(SCD.replace('cbName="MSVCB01"', 'cbName="Other"', 1))
    assert renamed["sv"]["publishes"] == []
    assert any("MSVCB01" in warning and "no network address (SMV)" in warning for warning in renamed["warnings"])
