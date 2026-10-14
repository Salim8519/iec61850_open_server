"""Signal-flow endpoints: who sends GOOSE and reports to whom, as declared in the SCD."""
from flask import Blueprint, Response, jsonify, request
from studio_app.projects import signal_flow
from .context import projects

blueprint = Blueprint("signal_flow", __name__)


def _source():
    source = request.args.get("source", "")
    if not source:
        sources = signal_flow.flow_sources(projects())
        if not sources:
            raise ValueError("Import an SCL file first")
        source = sources[0]["id"]
    return source


@blueprint.get("/signal-flow/sources")
def sources():
    return jsonify(sources=signal_flow.flow_sources(projects()))


@blueprint.get("/signal-flow")
def graph():
    return jsonify(signal_flow.graph(projects(), _source(), request.args.get("ied") or None))


@blueprint.get("/signal-flow/edges/<edge_id>.csv")
def edge_csv(edge_id):
    text, filename = signal_flow.edge_csv(projects(), _source(), edge_id)
    return Response(text, mimetype="text/csv",
                    headers={"Content-Disposition": f'attachment; filename="{filename}"'})
