"""Only the built-in server, no moves, no owner helper key (ADR-0097)."""
import re
from pathlib import Path
import pytest
from license_fakes import NOW, studio
from license_signer import HWID, bundle, keys
from studio_app.domain import license_policy as policy
from studio_app.licensing import client, release_key
from studio_app.licensing.client import LicenseServerError
from studio_app.licensing.service import LicenseActionError, LicenseService

STUDIO = Path(__file__).resolve().parents[2]


def install(tmp_path, text):
    home = tmp_path / "home"
    home.mkdir(parents=True, exist_ok=True)
    (home / "license.cert").write_text(text, encoding="utf-8")


def test_an_enforcing_build_ignores_the_configured_server_address(tmp_path):
    home = tmp_path / "home"
    home.mkdir()
    (home / "client_config.json").write_text('{"server_url": "http://127.0.0.1:9"}', encoding="utf-8")
    app = studio(tmp_path, [NOW])
    assert app.server_url() == release_key.DEFAULT_SERVER_URL
    assert app.status()["server_url"] == release_key.DEFAULT_SERVER_URL


@pytest.mark.parametrize("address", ["http://127.0.0.1:9", "http://localhost:8090", "https://evil.example.com", ""])
def test_the_server_address_cannot_be_changed_in_an_enforcing_build(tmp_path, address):
    app = studio(tmp_path, [NOW])
    with pytest.raises(LicenseActionError) as refused:
        app.set_server_url(address)
    assert refused.value.code == "bad_url" and str(refused.value) == policy.SERVER_FIXED
    assert not (tmp_path / "home/client_config.json").exists()


def test_the_real_client_refuses_every_address_but_the_built_in_one():
    for address in ("http://127.0.0.1:9", "http://localhost:8090", "https://evil.example.com", ""):
        with pytest.raises(LicenseServerError) as refused:
            client.LicenseClient(address)._call("GET", "/api/v1/health")
        assert refused.value.code == "no_server"
    assert client.server_allowed(release_key.DEFAULT_SERVER_URL)
    assert not client.server_allowed("https://license.scadabench.com.evil.example") and not client.server_allowed(
        release_key.DEFAULT_SERVER_URL + ".evil.example")


def test_the_service_builds_its_client_for_the_built_in_server_only(tmp_path):
    home = tmp_path / "home"
    home.mkdir()
    (home / "client_config.json").write_text('{"server_url": "http://127.0.0.1:9"}', encoding="utf-8")
    app = LicenseService(home, tmp_path / "data", True, hwid_provider=lambda: HWID, keys=keys())
    assert app._client().server_url == release_key.DEFAULT_SERVER_URL and not app._client().test_server


def test_the_test_seam_keeps_loopback_for_tests_and_the_sandbox(tmp_path):
    app = LicenseService(tmp_path / "home", tmp_path / "data", True, hwid_provider=lambda: HWID, keys=keys(),
                         allow_test_server=True)
    assert app.set_server_url("http://127.0.0.1:9")["server_url"] == "http://127.0.0.1:9"
    assert app._client().test_server and app._client().server_url == "http://127.0.0.1:9"
    with pytest.raises(LicenseActionError):
        app.set_server_url("ftp://x")


def test_no_part_of_the_product_turns_the_test_seam_on():
    needle = re.compile(r"allow_test_server\s*=\s*(True|1)|test_server\s*=\s*True")
    for folder in ("backend/studio_app", "tools"):
        for source in (STUDIO / folder).rglob("*.py"):
            if source.name == "license_sandbox.py":
                continue  # the sandbox is a developer tool that talks to a server on this PC
            assert not needle.search(source.read_text(encoding="utf-8")), source


# ----- no moves -----------------------------------------------------------------------------


def test_the_service_has_no_way_to_move_a_license():
    import inspect
    assert "move" not in inspect.signature(LicenseService.activate).parameters
    assert not hasattr(client.LicenseClient, "transfer") and not hasattr(LicenseService, "transfer")
    assert policy.WRONG_MACHINE == ("This license belongs to a different computer. Each license works on one "
                                    "computer only; buy a new license for this computer, or start a trial.")


# ----- the owner helper key is gone --------------------------------------------------------


def key_rows(path, pattern):
    return re.findall(pattern, path.read_text(encoding="utf-8"))


def test_only_the_license_servers_key_is_accepted_everywhere():
    assert list(release_key.public_keys()) == ["768dc76e"]
    python = key_rows(STUDIO / "backend/studio_app/licensing/release_key.py", r'"([0-9a-f]{64})"')
    go = key_rows(STUDIO / "engine/internal/licensing/keys.go", r'"([0-9a-f]{64})"')
    native = key_rows(STUDIO / "native/release/release_key.hpp", r"\{\{(0x[0-9a-f]{2}(?:, 0x[0-9a-f]{2}){31})\}\}")
    assert len(python) == len(go) == len(native) == 1
    assert bytes(int(item, 16) for item in native[0].split(", ")).hex() == python[0] == go[0]
    assert "kKeyCount = 1;" in (STUDIO / "native/release/release_key.hpp").read_text(encoding="utf-8")


def test_a_certificate_signed_by_the_retired_owner_key_is_refused(tmp_path):
    install(tmp_path, bundle(seed=bytes([7]) * 32, license_id="OWNER-0123456789ABCDEF", kid="16901336",
                             email="owner@scadabench.com"))
    status = LicenseService(tmp_path / "home", tmp_path / "data", True, hwid_provider=lambda: HWID,
                            clock=lambda: NOW + 60).status()  # the real release keys
    assert not status["licensed"] and "damaged or was changed" in status["reason"]
