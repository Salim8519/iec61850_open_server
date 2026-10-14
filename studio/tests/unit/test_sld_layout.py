"""SLD import: equipment mapping, wiring, layout and limits (spec SLD)."""
from collections import Counter
from pathlib import Path

import pytest
from studio_app.domain import sld_layout
from studio_app.domain.sld_layout import build_diagram
from studio_app.projects.sld_import import _merge
from studio_app.projects.validation import validate_diagram
from studio_app.scl.substation import read_substation
from sld_fixtures import equipment, feeder, scl

ROOT = Path(__file__).resolve().parents[3]


def build(text, source="src"):
    return build_diagram(read_substation(text), source)


def kinds(result):
    return {n["data"]["label"]: n["data"]["kind"] for n in result["diagram"]["nodes"]
            if n["data"]["kind"] != "JUNCTION"}


def test_equipment_kind_mapping():
    assert kinds(build(feeder())) == {"QB1": "DS", "QA1": "CB", "BI1": "CT", "BU1": "VT", "QC1": "ES"}


def test_unsupported_equipment_is_listed():
    skipped = {(item["path"], item["type"]) for item in build(feeder())["skipped"]}
    assert skipped == {("S/V/F1/IFL1", "IFL"), ("S/T1", "PowerTransformer")}
    assert all(item["reason"] for item in build(feeder())["skipped"])


def test_connectivity_nodes_become_wires_or_junctions():
    result = build(feeder())
    labels = Counter(n["data"]["label"] for n in result["diagram"]["nodes"]
                     if n["data"]["kind"] == "JUNCTION")
    # L1 and L2 join exactly two drawn items: plain wires, no junction.
    assert "F1/L1" not in labels and "F1/L2" not in labels
    # BB/L has one drawn terminal; F1/L3 joins CT, VT and ES: a junction row.
    assert labels["BB/L"] == 1 and labels["F1/L3"] >= 2
    # The grounded terminal of the earth switch never becomes a junction.
    assert not any("grounded" in label for label in labels)


def test_generated_diagrams_validate():
    for name in ("open_substation.scd", "model_substation.scd"):
        result = build((ROOT / "scd" / name).read_bytes())
        validate_diagram(result["diagram"], [])
    validate_diagram(build(feeder())["diagram"], [])


def test_esld_coordinates_are_used_without_overlap():
    result = build(feeder(esld=True))
    assert result["layout"] == "esld"
    cards = [n["position"] for n in result["diagram"]["nodes"] if n["data"]["kind"] != "JUNCTION"]
    for index, a in enumerate(cards):
        assert a["x"] % 20 == 0 and a["y"] % 20 == 0
        for b in cards[index + 1:]:
            assert abs(a["x"] - b["x"]) >= sld_layout.CARD_W or abs(a["y"] - b["y"]) >= sld_layout.CARD_H
    model = build((ROOT / "scd/model_substation.scd").read_bytes())
    assert model["layout"] == "esld"


def test_partial_coordinates_use_automatic_layout():
    text = scl([("F1", equipment("A", "CBR", ["S/V/F1/L1"], esld=(1, 1))
                 + equipment("B", "CBR", ["S/V/F1/L1"]), ["L1"])])
    result = build(text)
    assert result["layout"] == "automatic"
    assert any("Only some equipment" in item for item in result["diagnostics"])


def test_layout_is_deterministic():
    for text in (feeder(), feeder(esld=True), (ROOT / "scd/model_substation.scd").read_bytes()):
        assert build(text) == build(text)
    ids = {n["id"] for n in build(feeder())["diagram"]["nodes"]}
    assert ids.isdisjoint({n["id"] for n in build(feeder(), "other")["diagram"]["nodes"]})


def test_add_beside_skips_existing_ids():
    built = build(feeder())["diagram"]
    first, present = _merge({"nodes": [], "edges": []}, build(feeder())["diagram"], "beside")
    assert present == []
    again, present = _merge(first, built, "beside")
    assert len(again["nodes"]) == len(first["nodes"]) and len(present) == len(first["nodes"])
    assert len({n["id"] for n in again["nodes"]}) == len(again["nodes"])
    assert len({e["id"] for e in again["edges"]}) == len(again["edges"])


def test_oversized_station_is_refused(monkeypatch):
    monkeypatch.setattr(sld_layout, "MAX_NODES", 3)
    with pytest.raises(ValueError, match="diagram limit"):
        build(feeder())
