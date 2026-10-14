"""Serve the declared signal-flow graph of a stored SCL document."""
import csv
import io
from studio_app.contracts.errors import StudioError
from studio_app.domain.signal_flow import build_graph
from studio_app.scl.flow_sources import read_flow
from .scl_sources import require_source, stored_sources


def flow_sources(service):
    with service.lock:
        return [{"id": entry["id"], "filename": entry["filename"], "ieds": entry["ieds"]}
                for entry in stored_sources(service.project).values()]


def graph(service, source, ied=None):
    with service.lock:
        entry = require_source(service.project, source)
        loaded = {item["name"] for item in service.project["ieds"]}
    result = build_graph(read_flow(entry["text"]), loaded, ied)
    if ied is not None and not any(item["name"] == ied for item in result["ieds"]):
        result["ieds"] = [{"name": ied, "desc": "", "manufacturer": "", "loaded": ied in loaded}]
    return {"source": entry["id"], "filename": entry["filename"], **result}


def edge_csv(service, source, edge_id):
    """The signal rows of one edge, exactly as the view lists them."""
    result = graph(service, source)
    edge = next((item for item in result["edges"] if item["id"] == edge_id), None)
    if edge is None:
        raise StudioError("Unknown signal-flow link", code="not_found")
    block = next(item for item in result["blocks"] if item["id"] == edge["block"])
    buffer = io.StringIO()
    writer = csv.writer(buffer, lineterminator="\n")
    writer.writerow(["type", "publisher", "block", "subscriber", "source", "input", "description"])
    for signal in edge["signals"]:
        writer.writerow([edge["type"], edge["publisher"], block["ref"], edge["subscriber"],
                         signal["source"], signal["int_addr"], signal["desc"]])
    return buffer.getvalue(), f"{edge['publisher']}-to-{edge['subscriber']}-{edge['type']}.csv"
