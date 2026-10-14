"""Reading the SCL Substation section (spec SLD)."""
from pathlib import Path

from studio_app.scl.substation import has_substation, read_substation
from sld_fixtures import equipment, feeder, scl

ROOT = Path(__file__).resolve().parents[3]


def test_unresolved_terminal_is_diagnosed():
    text = scl([("F1", equipment("QA1", "CBR", ["S/V/F1/L1", "S/V/F1/NOPE"]), ["L1"])])
    result = read_substation(text)
    item = result["equipment"][0]
    assert item["terminals"] == ["S/V/F1/L1", None]
    assert any("S/V/F1/NOPE" in message and "not declared" in message for message in result["diagnostics"])


def test_grounded_terminal_and_lnodes_are_read():
    lnode = '<LNode iedName="P1" ldInst="LD0" lnClass="XSWI" lnInst="1"/>'
    phase = '<SubEquipment name="A" phase="A"><LNode iedName="P1" ldInst="LD0" lnClass="TCTR" lnInst="1"/></SubEquipment>'
    text = scl([("F1", equipment("QC1", "DIS", ["S/V/F1/L1", "grounded"], lnodes=lnode + phase), ["L1"])])
    item = read_substation(text)["equipment"][0]
    assert item["terminals"] == ["S/V/F1/L1", "grounded"]
    assert [entry["lnClass"] for entry in item["lnodes"]] == ["XSWI"]


def test_fixture_documents_are_read():
    model = read_substation((ROOT / "scd/model_substation.scd").read_bytes())
    assert model["substations"] == ["SS1"] and len(model["equipment"]) == 56
    assert model["nodes"]["SS1/VL_50kV/BB1/L"]["bus_y"] == 13.5
    assert has_substation(feeder())
    assert not has_substation('<SCL xmlns="http://www.iec.ch/61850/2003/SCL"/>')
