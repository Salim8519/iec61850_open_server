"""Declared signal flow between IEDs (spec FLOW)."""
from pathlib import Path

from studio_app.domain.signal_flow import build_graph
from studio_app.scl.flow_sources import read_flow

SCD = (Path(__file__).resolve().parents[3] / "scd/open_substation.scd").read_text(encoding="utf-8")
NAMED = SCD.replace('srcCBName="GoCB"', 'srcCBName="gcbEvents"')
BY_DATA = SCD.replace(' srcLDInst="GenericIO" srcLNClass="LLN0" srcCBName="GoCB"', "")
CLIENTS = ('<RptEnabled max="2"><ClientLN iedName="IED2_PTOC" ldInst="GenericIO" lnClass="IHMI" lnInst="1"/>'
           '<ClientLN iedName="SCADA1" ldInst="LD0" lnClass="ITCI" lnInst="1"/></RptEnabled>')


def graph(text, loaded=("IED1_XCBR",), ied=None):
    return build_graph(read_flow(text.encode("utf-8")), set(loaded), ied)


def test_goose_edges_from_extref():
    result = graph(NAMED)
    goose = [edge for edge in result["edges"] if edge["type"] == "goose"]
    assert [(e["publisher"], e["subscriber"]) for e in goose] == [("IED2_PTOC", "IED1_XCBR")]
    assert goose[0]["signals"] == [{"source": "IED2_PTOC/GenericIO/PTRC1.Tr.general", "int_addr": "Tr",
                                    "desc": "Trip"}]
    block = next(b for b in result["blocks"] if b["id"] == goose[0]["block"])
    assert block["ref"] == "IED2_PTOCGenericIO/LLN0$GO$gcbEvents" and block["type"] == "goose"


def test_match_method_is_recorded():
    assert graph(NAMED)["edges"][0]["match"] == "named"
    by_data = [e for e in graph(BY_DATA)["edges"] if e["type"] == "goose"]
    assert len(by_data) == 1 and by_data[0]["match"] == "data"


def test_unloaded_ieds_are_marked():
    ieds = {item["name"]: item["loaded"] for item in graph(NAMED)["ieds"]}
    assert ieds == {"IED1_XCBR": True, "IED2_PTOC": False, "IED3_SMV": False, "IED4_SMV": False}


def test_unresolved_extref_creates_no_edge():
    result = graph(SCD)
    assert not [e for e in result["edges"] if e["type"] == "goose"]
    problems = [d for d in result["diagnostics"] if d["level"] == "problem"]
    assert any("IED2_PTOC/GenericIO/GoCB" in d["message"] and d["ied"] == "IED1_XCBR" for d in problems)
    infos = [d["message"] for d in result["diagnostics"] if d["level"] == "info"]
    assert any("Nobody in this file subscribes" in message for message in infos)


def test_graph_is_deterministic():
    assert graph(NAMED) == graph(NAMED)
    assert graph(NAMED.replace('<RptEnabled max="1" />', CLIENTS)) == graph(NAMED.replace('<RptEnabled max="1" />', CLIENTS))


def test_single_ied_view_keeps_only_its_links():
    text = NAMED.replace('<RptEnabled max="1" />', CLIENTS)
    result = graph(text, ied="IED2_PTOC")
    assert result["edges"] and all("IED2_PTOC" in (e["publisher"], e["subscriber"]) for e in result["edges"])
    assert {item["name"] for item in result["ieds"]} == {"IED1_XCBR", "IED2_PTOC"}
    assert all(item["ied"] == "IED2_PTOC" or "IED2_PTOC" in item["message"] for item in result["diagnostics"])
    broken = graph(SCD, ied="IED2_PTOC")["diagnostics"]
    assert any(item["ied"] == "IED1_XCBR" and item["level"] == "problem" for item in broken)
    assert not graph(SCD, ied="IED2_PTO")["diagnostics"]
