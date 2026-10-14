"""Station health endpoints (spec HEALTH)."""
import csv
import io
import json
from pathlib import Path

import pytest
from studio_app.app_factory import create_app
from studio_app.settings import Settings

ROOT = Path(__file__).resolve().parents[3]
SOURCE = (ROOT / "scd/open_substation.scd").read_bytes()


@pytest.fixture
def client(tmp_path):
    app = create_app(Settings(tmp_path, tmp_path / "unused.exe", tmp_path), "health-test")
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "health-test"})
    client.post("/api/v1/scl/import", data={"file": (io.BytesIO(SOURCE), "station.scd"),
                                            "selected_names": json.dumps(["IED1_XCBR", "IED2_PTOC"])})
    yield client
    app.extensions["runtime"].close()


def test_csv_matches_results(client):
    health = client.get("/api/v1/health").json
    assert health["running"] is False and health["verdict"] == "not_checked"
    shown = [(entry["name"], check["area"], check["item"], check["result"], check["message"])
             for entry in health["ieds"] for check in entry["checks"]]
    response = client.get("/api/v1/health.csv")
    rows = list(csv.DictReader(io.StringIO(response.get_data(as_text=True))))
    assert [(row["ied"], row["area"], row["item"], row["result"], row["message"]) for row in rows] == shown
    assert set(rows[0]) >= {"observed_at", "source", "stale"}


def test_snapshot_report_is_escaped_html(client):
    page = client.get("/api/v1/health/report.html").get_data(as_text=True)
    assert page.startswith("<!doctype html>") and "Station health" in page and "<script" not in page
    assert client.post("/api/v1/health/refresh").json["failures"] == {}
