"""Infeed is drawing-only and never reaches an engine (spec energization, INFEED-01..04)."""
from pathlib import Path

import pytest
from studio_app.app_factory import create_app
from studio_app.contracts.errors import StudioError
from studio_app.domain.sld_layout import build_diagram
from studio_app.projects.archive import decode
from studio_app.projects.validation import validate_diagram
from studio_app.scl.substation import read_substation
from studio_app.settings import Settings

ROOT = Path(__file__).resolve().parents[3]


def node(node_id, kind, **extra):
    return {"id": node_id, "position": {"x": 0, "y": 0}, "data": {"label": node_id, "kind": kind, "rotation": 0, **extra}}


def test_infeed_rejects_bindings():
    binding = {"ied_id": "ied-a", "ref": "GGIO1.Ind1.stVal"}
    with pytest.raises(StudioError, match="infeed is a drawing point"):
        validate_diagram({"nodes": [node("f", "INFEED", state_mapping={"mode": "direct", "source": binding})]}, [])
    with pytest.raises(StudioError, match="infeed is a drawing point"):
        validate_diagram({"nodes": [node("f", "INFEED", control_binding=binding)]}, [])
    with pytest.raises(StudioError, match="Only an infeed"):
        validate_diagram({"nodes": [node("cb", "CB", infeed_state="live")]}, [])
    stored = validate_diagram({"nodes": [node("f", "INFEED")]}, [])["nodes"][0]["data"]
    assert (stored["infeed_type"], stored["infeed_state"]) == ("line", "live")


def test_existing_nodes_store_no_infeed_keys():
    kinds = ["CB", "DS", "ES", "CT", "VT", "SOURCE", "RELAY", "TRIP_RELAY", "PUSH_BUTTON", "JUNCTION"]
    stored = validate_diagram({"nodes": [node(kind.lower(), kind) for kind in kinds]}, [])
    for item in stored["nodes"]:
        assert "infeed_type" not in item["data"] and "infeed_state" not in item["data"], item["id"]
    assert validate_diagram(stored, []) == stored, "stored form is stable"


@pytest.fixture
def studio(tmp_path):
    app = create_app(Settings(tmp_path, tmp_path / "unused.exe", tmp_path), "infeed-test")
    client = app.test_client()
    client.post("/api/v1/session", json={"token": "infeed-test"})
    ied = app.extensions["projects"].add_template("ExportBay")[0]
    yield app, client, ied
    app.extensions["runtime"].close()


def test_scl_export_ignores_infeeds(studio):
    app, client, ied = studio
    service = app.extensions["projects"]
    exported = lambda: client.post(f"/api/v1/ieds/{ied['id']}/export-scl", json={"overrides": {}}).data
    before = exported()
    diagram = {"nodes": [node("feed", "INFEED", infeed_type="transformer"), node("j", "JUNCTION")],
               "edges": [{"id": "w", "source": "feed", "target": "j"}]}
    response = client.put("/api/v1/project/diagram", json={"diagram": diagram, "revision": service.revision, "tab_id": "t"})
    assert response.status_code == 200, response.json
    assert exported() == before
    assert client.get(f"/api/v1/ieds/{ied['id']}/source").data == ied["source"].encode()
    assert decode(service.export())["diagram"]["nodes"][0]["data"]["infeed_type"] == "transformer"


def test_sld_import_never_creates_infeeds():
    for name in ("model_substation", "open_substation"):
        topology = read_substation((ROOT / f"scd/{name}.scd").read_text(encoding="utf-8"))
        kinds = {item["data"]["kind"] for item in build_diagram(topology, "src")["diagram"]["nodes"]}
        assert "INFEED" not in kinds and kinds <= {"CB", "DS", "ES", "CT", "VT", "JUNCTION"}
