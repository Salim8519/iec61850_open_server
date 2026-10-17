"""Station health endpoints: live network and real IEDs compared with the SCD."""
from flask import Blueprint, Response, jsonify
from studio_app.acquisition import health_snapshot
from studio_app.domain.health_export import csv_text, html_report
from studio_app.version import __version__
from .context import projects, runtime

blueprint = Blueprint("health", __name__)


def _health():
    return health_snapshot.station_health(projects(), runtime())


@blueprint.get("/health")
def health():
    return jsonify({**_health(), "app_version": __version__})


@blueprint.post("/health/refresh")
def refresh():
    failures = health_snapshot.refresh(runtime())
    return jsonify(failures=failures, **_health())


@blueprint.get("/health.csv")
def health_csv():
    return Response(csv_text(_health()), mimetype="text/csv",
                    headers={"Content-Disposition": "attachment; filename=station-health.csv"})


@blueprint.get("/health/report.html")
def health_report():
    service = projects()
    page = html_report(_health(), service.project["name"], f"{service.session_id}-{service.revision}")
    return Response(page, mimetype="text/html",
                    headers={"Content-Disposition": "attachment; filename=station-health.html"})
