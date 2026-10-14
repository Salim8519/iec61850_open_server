"""Signal-flow endpoints (spec FLOW)."""
import csv
import io
import json
from pathlib import Path

import pytest
from studio_app.app_factory import create_app
from studio_app.settings import Settings

ROOT = Path(__file__).resolve().parents[3]
SOURCE = (ROOT / "scd/open_substation.scd").read_text(encoding="utf-8").replace(
    'srcCBName="GoCB"', 'srcCBName="gcbEvents"').encode("utf-8")


@pytest.fixture
def client(tmp_path):
    app = create_app(Settings(tmp_path, tmp_path / "unused.exe", tmp_path), "flow-test")
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "flow-test"})
    client.post("/api/v1/scl/import", data={"file": (io.BytesIO(SOURCE), "station.scd"),
                                            "selected_names": json.dumps(["IED1_XCBR"])})
    yield client
    app.extensions["runtime"].close()


def test_edge_csv_matches_rows(client):
    result = client.get("/api/v1/signal-flow").json
    edge = next(item for item in result["edges"] if item["type"] == "goose")
    response = client.get(f"/api/v1/signal-flow/edges/{edge['id']}.csv?source={result['source']}")
    assert response.status_code == 200 and response.mimetype == "text/csv"
    rows = list(csv.DictReader(io.StringIO(response.get_data(as_text=True))))
    assert [(row["source"], row["input"], row["description"]) for row in rows] == [
        (signal["source"], signal["int_addr"], signal["desc"]) for signal in edge["signals"]]
    assert rows[0]["publisher"] == "IED2_PTOC" and rows[0]["subscriber"] == "IED1_XCBR"


def test_ied_filter_and_unknown_edge(client):
    result = client.get("/api/v1/signal-flow?ied=IED1_XCBR").json
    assert all("IED1_XCBR" in (e["publisher"], e["subscriber"]) for e in result["edges"])
    assert client.get("/api/v1/signal-flow/edges/nope.csv").status_code == 400
    assert client.get("/api/v1/signal-flow/sources").json["sources"][0]["filename"] == "station.scd"
