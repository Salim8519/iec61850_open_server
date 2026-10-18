"""Signal-layer endpoints: GOOSE link map, MMS clients and remembered switches (LAYER-07..09)."""
import io
import json
from pathlib import Path

from studio_app.app_factory import create_app
from studio_app.settings import Settings

ROOT = Path(__file__).resolve().parents[3]
SOURCE = (ROOT / "scd/open_substation.scd").read_text(encoding="utf-8").replace(
    'srcCBName="GoCB"', 'srcCBName="gcbEvents"').encode("utf-8")


def client_for(tmp_path, names=("IED1_XCBR",)):
    app = create_app(Settings(tmp_path, tmp_path / "unused.exe", tmp_path), "layer-test")
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "layer-test"})
    if names:
        client.post("/api/v1/scl/import", data={"file": (io.BytesIO(SOURCE), "station.scd"),
                                                "selected_names": json.dumps(list(names))})
    return app, client


def test_goose_links_name_publisher_block_and_subscriber(tmp_path):
    app, client = client_for(tmp_path)
    try:
        result = client.get("/api/v1/goose/links").json
        link = next(item for item in result["links"] if item["subscriber"] == "IED1_XCBR")
        assert link["publisher"] == "IED2_PTOC" and link["gocb_ref"] == "IED2_PTOCGenericIO/LLN0$GO$gcbEvents"
        assert link["status"] == "declared" and link["live"] == {"state": "stopped"} and result["running"] is False
        publisher = next(ied for ied in result["ieds"] if ied["name"] == "IED2_PTOC")
        assert publisher["in_project"] is False and publisher["remote"] is False
        block = next(item for item in result["blocks"] if item["gocb_ref"] == link["gocb_ref"])
        assert "IED2_PTOCGenericIO/PTRC1.Tr.general" in block["member_refs"]
        assert result["summary"]["IED1_XCBR"]["gets_from"] == ["IED2_PTOC"]
    finally:
        app.extensions["runtime"].close()


def test_empty_project_has_an_empty_map_and_no_clients(tmp_path):
    app, client = client_for(tmp_path, names=())
    try:
        assert client.get("/api/v1/goose/links").json["links"] == []
        assert client.get("/api/v1/mms/access").json == {"running": False, "ieds": {}}
    finally:
        app.extensions["runtime"].close()


def test_layer_switches_are_remembered_across_restarts(tmp_path):
    app, client = client_for(tmp_path, names=())
    try:
        assert client.get("/api/v1/preferences").json["sld_goose_layer"] == 0
        saved = client.put("/api/v1/preferences", json={"sld_goose_layer": 1, "sld_readers_layer": 1})
        assert saved.status_code == 200
        assert client.put("/api/v1/preferences", json={"sld_goose_layer": 2}).status_code == 400
    finally:
        app.extensions["runtime"].close()
    app, client = client_for(tmp_path, names=())
    try:
        prefs = client.get("/api/v1/preferences").json
        assert prefs["sld_goose_layer"] == 1 and prefs["sld_readers_layer"] == 1
    finally:
        app.extensions["runtime"].close()
