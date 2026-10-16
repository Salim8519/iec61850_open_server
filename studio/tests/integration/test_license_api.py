"""License gate: packaged builds answer 402 until activated; development never checks."""
from pathlib import Path
import sys
import pytest
from studio_app.app_factory import create_app
from studio_app.licensing.service import LicenseService
from studio_app.settings import Settings

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / "studio/tests/unit"))
from license_signer import HWID, bundle, keys  # noqa: E402


def make_app(tmp_path, enforce):
    settings = Settings(tmp_path / "data", ROOT / "studio/native/build/Release/studio_ied.exe", tmp_path)
    service = LicenseService(tmp_path / "home", settings.directory, enforce, hwid_provider=lambda: HWID,
                             clock=lambda: 1_760_000_100, keys=keys())
    application = create_app(settings, "license-token", license_service=service)
    client = application.test_client()
    assert client.post("/api/v1/session", json={"token": "license-token"}).status_code == 200
    return application, client


@pytest.fixture
def packaged(tmp_path):
    application, client = make_app(tmp_path, True)
    yield client
    application.extensions["runtime"].close()


def test_development_backend_never_asks_for_a_license(tmp_path):
    application, client = make_app(tmp_path, False)
    try:
        assert client.get("/api/v1/project").status_code == 200
        status = client.get("/api/v1/license/status").json
        assert status["licensed"] and status["mode"] == "development" and not status["enforced"]
        assert client.post("/api/v1/license/trial").status_code == 400
    finally:
        application.extensions["runtime"].close()


def test_packaged_backend_blocks_engineering_api_until_licensed(packaged):
    blocked = packaged.get("/api/v1/project")
    assert blocked.status_code == 402 and blocked.json["code"] == "license_required"
    status = packaged.get("/api/v1/license/status")
    assert status.status_code == 200 and not status.json["licensed"] and status.json["hwid"] == HWID
    assert packaged.post("/api/v1/session", json={"token": "license-token"}).status_code == 200


def test_offline_activation_unlocks_the_api(packaged):
    wrong = packaged.post("/api/v1/license/activate-offline", json={"certificate": bundle(hwid="cd" * 32)})
    assert wrong.status_code == 400 and "different computer" in wrong.json["error"]
    good = packaged.post("/api/v1/license/activate-offline", json={"certificate": bundle()})
    assert good.status_code == 200 and good.json["licensed"] and good.json["mode"] == "lifetime"
    assert packaged.get("/api/v1/project").status_code == 200


def test_online_activation_reports_unreachable_server(packaged):
    assert packaged.put("/api/v1/license/config", json={"server_url": "http://127.0.0.1:9"}).status_code == 200
    response = packaged.post("/api/v1/license/activate", json={"license_key": "SUBS-2345-6789-ABCD-EFGH"})
    assert response.status_code == 503 and response.json["code"] == "offline"
    assert packaged.put("/api/v1/license/config", json={"server_url": "ftp://x"}).status_code == 400
