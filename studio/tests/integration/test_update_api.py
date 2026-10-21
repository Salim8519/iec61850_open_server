"""Update check API (ADR-0071): read, check now, dismiss and switch off, behind the session."""
from pathlib import Path
import sys
from studio_app.app_factory import create_app
from studio_app.settings import Settings
from studio_app.updates.service import UpdateService

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / "studio/tests/unit"))
from license_signer import keys  # noqa: E402
from test_update_check import Opener, answer  # noqa: E402


def test_update_endpoints_need_the_session_and_store_choices(tmp_path):
    settings = Settings(tmp_path / "data", ROOT / "studio/native/build/Release/studio_ied.exe", tmp_path)
    update = UpdateService(settings.directory, lambda: "https://license.scadabench.com", current="0.1.13",
                           keys=keys(), opener=Opener(answer()), clock=lambda: 1_000_000)
    application = create_app(settings, "update-token", update_service=update)
    client = application.test_client()
    try:
        assert client.get("/api/v1/app/update").status_code in (401, 403)
        assert client.post("/api/v1/session", json={"token": "update-token"}).status_code == 200
        assert client.get("/api/v1/app/update").json["available"] is False
        checked = client.post("/api/v1/app/update/check").json
        assert checked["available"] and checked["latest"]["page"] == "https://scadabench.com/download"
        assert client.put("/api/v1/app/update", json={"dismissed": "0.1.14"}).json["dismissed"] is True
        assert client.put("/api/v1/app/update", json={"auto": False}).json["auto"] is False
        assert client.put("/api/v1/app/update", json={"server": "x"}).status_code == 400
    finally:
        application.extensions["runtime"].close()
