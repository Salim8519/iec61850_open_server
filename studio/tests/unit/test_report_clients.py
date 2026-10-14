"""Report clients come only from RptEnabled/ClientLN (spec FLOW-03)."""
from studio_app.domain.signal_flow import build_graph
from studio_app.scl.document import parse_document
from studio_app.scl.flow_sources import read_flow
from studio_app.scl.report_clients import report_blocks
from test_signal_flow import CLIENTS, SCD

TEXT = SCD.replace('<RptEnabled max="1" />', CLIENTS).encode("utf-8")


def test_client_ln_is_read_with_dataset_members():
    block = next(item for item in report_blocks(parse_document(TEXT)) if item["name"] == "EventsRCB")
    assert block["ref"] == "IED1_XCBRGenericIO/LLN0.BR.EventsRCB" and block["buffered"]
    assert [client["iedName"] for client in block["clients"]] == ["IED2_PTOC", "SCADA1"]
    assert block["dataset_found"] and block["members"]


def test_unknown_client_is_diagnosed():
    result = build_graph(read_flow(TEXT), {"IED1_XCBR"})
    reports = [edge for edge in result["edges"] if edge["type"] == "report"]
    assert [(e["publisher"], e["subscriber"]) for e in reports] == [("IED1_XCBR", "IED2_PTOC")]
    outside = [d for d in result["diagnostics"] if "SCADA1" in d["message"]]
    assert len(outside) == 1 and outside[0]["level"] == "info" and "outside this file" in outside[0]["message"]


def test_report_without_client_creates_no_edge():
    result = build_graph(read_flow(SCD.encode("utf-8")), {"IED1_XCBR"})
    assert not [edge for edge in result["edges"] if edge["type"] == "report"]
    assert any(block["type"] == "report" for block in result["blocks"])
