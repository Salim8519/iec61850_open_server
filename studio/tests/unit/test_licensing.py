"""License rules: development bypass, trial, rollback, machine lock and tampering."""
import hashlib
import json
from pathlib import Path
import pytest
from license_signer import HWID, bundle, keys, public_key, verify_answer
from studio_app.domain import license_policy as policy
from studio_app.licensing import certificate, client, hwid
from studio_app.licensing import release_key
from studio_app.licensing.release_key import key_id
from studio_app.licensing.service import LicenseActionError, LicenseService

FIXTURES = Path(__file__).resolve().parents[1] / "fixtures/licensing"
NOW = 1_760_000_000
DAY = 86400


def service(tmp_path, enforce=True, now=NOW + 60, **options):
    clock = options.pop("clock", lambda: now)
    provider = options.pop("hwid_provider", lambda: HWID)
    return LicenseService(tmp_path / "home", tmp_path / "data", enforce, hwid_provider=provider,
                          clock=clock, keys=keys(), **options)


def install(tmp_path, text):
    home = tmp_path / "home"
    home.mkdir(parents=True, exist_ok=True)
    (home / "license.cert").write_text(text, encoding="utf-8")


def test_development_mode_is_always_licensed_without_reading_files(tmp_path):
    studio = service(tmp_path, enforce=False, hwid_provider=None)
    status = studio.status()
    assert status["licensed"] and status["mode"] == "development" and status["hwid"] is None
    assert not (tmp_path / "home").exists()
    assert policy.evaluate_certificate(None, HWID, NOW, 0, True).mode == "development"


def test_packaged_mode_without_certificate_is_unlicensed(tmp_path):
    status = service(tmp_path).status()
    assert not status["licensed"] and status["reason"] == policy.NO_LICENSE
    assert status["hwid"] == HWID


def test_seven_day_trial_counts_down_then_expires(tmp_path):
    install(tmp_path, bundle(type="trial", expires_at=NOW + 7 * DAY, license_id="TRIAL-x"))
    status = service(tmp_path, now=NOW + DAY + 5).status()
    assert status["licensed"] and status["mode"] == "trial" and status["days_remaining"] == 6
    ended = service(tmp_path, now=NOW + 7 * DAY).status()
    assert not ended["licensed"] and ended["reason"] == policy.TRIAL_ENDED


def test_clock_rollback_is_detected_from_anchor_and_issue_time(tmp_path):
    install(tmp_path, bundle(type="trial", expires_at=NOW + 7 * DAY))
    assert service(tmp_path, now=NOW + 4 * DAY).is_licensed()
    rolled_back = service(tmp_path, now=NOW + DAY).status()  # trials allow 48 h of drift (0106)
    assert not rolled_back["licensed"] and "rollback" in rolled_back["reason"]
    assert policy.evaluate_certificate({"version": 1, "type": "trial", "hwid": HWID, "issued_at": NOW,
                                        "expires_at": NOW + DAY}, HWID, NOW - 3600, 0, False).reason \
        == policy.CLOCK_ROLLBACK


def test_clock_rollback_uses_latest_run_log_time(tmp_path):
    install(tmp_path, bundle())
    logs = tmp_path / "data/logs"
    logs.mkdir(parents=True)
    (logs / "runs.jsonl").write_text('{"kind":"run_started","at":"2026-01-01T00:00:00+00:00"}\n')
    assert not service(tmp_path, now=NOW).is_licensed()


def test_lifetime_license_never_expires(tmp_path):
    install(tmp_path, bundle(email="buyer@example.com"))
    status = service(tmp_path, now=NOW + 3650 * DAY).status()
    assert status["licensed"] and status["mode"] == "lifetime" and status["days_remaining"] is None
    assert status["masked_key"] == "SUBS-****-****-****-TEST"


def test_certificate_for_another_machine_is_rejected(tmp_path):
    install(tmp_path, bundle(hwid="cd" * 32))
    status = service(tmp_path).status()
    assert not status["licensed"] and status["reason"] == policy.WRONG_MACHINE
    assert status["reason"].endswith("buy a new license for this computer, or start a trial.")
    assert not status["transfer_available"]  # licenses never move (ADR-0097); the field stays for old screens


def test_tampered_or_foreign_signature_is_rejected(tmp_path):
    original = json.loads(bundle(type="trial", expires_at=NOW + DAY))
    original["payload"] = original["payload"].replace(str(NOW + DAY), str(NOW + 999 * DAY))
    install(tmp_path, json.dumps(original))
    assert "changed" in service(tmp_path).status()["reason"]
    install(tmp_path, bundle(seed=bytes(32)))
    assert not service(tmp_path).is_licensed()
    with pytest.raises(certificate.CertificateError):
        certificate.verify_bundle(certificate.parse_bundle(bundle()), {})


def test_go_signed_certificate_verifies_in_python():
    vector = json.loads((FIXTURES / "go_vector.json").read_text())
    public = bytes.fromhex(vector["public_key"])
    assert public == public_key() and key_id(public) == vector["kid"]
    payload = certificate.verify_bundle(vector["lifetime"], {vector["kid"]: public})
    assert payload["type"] == "lifetime" and payload["hwid"] == vector["hwid"]
    assert certificate.verify_bundle(vector["trial"], {vector["kid"]: public})["expires_at"] == NOW + 7 * DAY


def test_hwid_composition_is_stable_and_matches_native_format():
    value = hwid.compose(" 1A2B3C4D-AAAA-BBBB-CCCC-0123456789EF ", 0xA1B2C3D4, "aabb")
    assert value == hwid.compose("1a2b3c4d-aaaa-bbbb-cccc-0123456789ef", 0xA1B2C3D4, "aabb")
    text = "substation-studio-hwid-v2|1a2b3c4d-aaaa-bbbb-cccc-0123456789ef|A1B2C3D4|aabb"
    assert value == hashlib.sha256(text.encode()).hexdigest()
    assert hwid.compose("g", 0x1F, "") == hashlib.sha256(b"substation-studio-hwid-v2|g|0000001F|").hexdigest()


def test_proof_of_work_solver_meets_difficulty():
    counter = client.solve("00ff", 12)
    digest = hashlib.sha256(f"00ff:{counter}".encode()).digest()
    assert int.from_bytes(digest[:2], "big") >> 4 == 0
    with pytest.raises(client.LicenseServerError):
        client.solve("x", 40)


class FakeClient:
    def __init__(self, url):
        self.url = url

    def start_trial(self, machine, parts=None, email=None, code=None):
        return json.loads(bundle(type="trial", expires_at=NOW + 7 * DAY, hwid=machine))

    def activate(self, key, machine):
        return json.loads(bundle(license_id=key, hwid=machine))

    def verify(self, key, machine, nonce):
        return {"status": "revoked", "answer": verify_answer(nonce, key, machine, "revoked")}


def test_trial_activation_and_offline_install_save_certificate(tmp_path):
    studio = service(tmp_path, client_factory=FakeClient)
    assert studio.start_trial()["mode"] == "trial"
    assert studio.activate("subs-2345-6789-abcd-efgh")["mode"] == "lifetime"
    assert (tmp_path / "home/license.cert").exists()
    with pytest.raises(LicenseActionError):
        studio.activate("SUBS-0000-1111-OOOO-IIII")
    with pytest.raises(LicenseActionError):
        studio.activate_offline(bundle(hwid="cd" * 32))
    with pytest.raises(LicenseActionError):
        studio.activate_offline("not json")


def test_online_check_removes_refunded_license_but_keeps_file(tmp_path):
    install(tmp_path, bundle())
    studio = service(tmp_path, now=NOW + 30 * DAY, client_factory=FakeClient)
    assert studio.verify_online_if_due() == "revoked"
    assert not studio.is_licensed() and "cancelled" in studio.status()["reason"]
    assert list((tmp_path / "home").glob("license.cert.revoked-*"))


def test_online_check_keeps_a_license_the_server_does_not_know(tmp_path):
    class UnknownToServer(FakeClient):
        def verify(self, key, machine, nonce):
            return {"status": "revoked", "answer": verify_answer(nonce, key, machine, "unknown")}
    install(tmp_path, bundle())
    studio = service(tmp_path, now=NOW + 30 * DAY, client_factory=UnknownToServer)
    assert studio.verify_online_if_due() == "unknown"
    assert studio.is_licensed() and (tmp_path / "home/license.cert").exists()


def test_swapped_native_worker_or_debugger_blocks_packaged_mode(tmp_path, monkeypatch):
    from studio_app.licensing import integrity
    install(tmp_path, bundle())
    native = tmp_path / "native"
    native.mkdir()
    (native / "studio_ied.exe").write_bytes(b"original")
    expected = {"studio_ied.exe": integrity.file_digest(native / "studio_ied.exe")}
    assert service(tmp_path, native_folder=native, expected_files=expected).is_licensed()
    (native / "studio_ied.exe").write_bytes(b"patched")
    assert service(tmp_path, native_folder=native, expected_files=expected).status()["reason"] == integrity.TAMPERED
    monkeypatch.setattr(integrity, "debugger_attached", lambda: True)
    assert service(tmp_path).status()["reason"] == integrity.DEBUGGER
    assert service(tmp_path, enforce=False).is_licensed()


def test_server_url_must_be_https(tmp_path):
    studio = service(tmp_path, allow_test_server=True)  # the test-only seam keeps the old address rule
    with pytest.raises(LicenseActionError):
        studio.set_server_url("http://evil.example.com")
    assert studio.set_server_url("https://license.example.com/")["server_url"] == "https://license.example.com"


def test_license_client_verifies_https_with_the_windows_certificate_check(monkeypatch):
    """ADR-0056: OpenSSL alone rejected the 2026 Let's Encrypt chain on PCs without ISRG Root X2."""
    import ssl
    import truststore
    context = client.https_context()
    assert isinstance(context, truststore.SSLContext)
    assert context.verify_mode == ssl.CERT_REQUIRED and context.check_hostname
    seen = {}

    class Answer:
        def __enter__(self):
            return self

        def __exit__(self, *error):
            return False

        def read(self, size):
            return b'{"ok": true}'

    def fake_urlopen(request, timeout, context):
        seen.update(url=request.full_url, context=context)
        return Answer()
    monkeypatch.setattr(client, "urlopen", fake_urlopen)
    built_in = release_key.DEFAULT_SERVER_URL
    assert client.LicenseClient(built_in)._call("GET", "/api/v1/health") == {"ok": True}
    assert seen == {"url": built_in + "/api/v1/health", "context": context}
