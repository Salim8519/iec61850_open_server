"""Licenses bound to the pre-v2 machine ID keep working after an update (ADR-0051)."""
import hashlib
import json
from license_signer import HWID, bundle, keys, verify_answer
from studio_app.domain import license_policy as policy
from studio_app.licensing import hwid
from studio_app.licensing.service import LicenseService

LEGACY = "cd" * 32
NOW = 1_760_000_000
DAY = 86400


def service(tmp_path, now=NOW + 60, **options):
    options.setdefault("legacy_hwid_provider", lambda: LEGACY)
    return LicenseService(tmp_path / "home", tmp_path / "data", True, hwid_provider=lambda: HWID,
                          clock=lambda: now, keys=keys(), **options)


def install(tmp_path, text):
    home = tmp_path / "home"
    home.mkdir(parents=True, exist_ok=True)
    (home / "license.cert").write_text(text, encoding="utf-8")


class RecordingClient:
    calls = []

    def __init__(self, url):
        self.url = url

    def verify(self, key, machine, nonce):
        RecordingClient.calls.append(machine)
        return {"status": "valid", "answer": verify_answer(nonce, key, machine, "valid"),
                "certificate": json.loads(bundle(hwid=machine, issued_at=NOW + 30 * DAY))}


def test_legacy_composition_matches_v1_format():
    assert hwid.compose_legacy(" G ", 0x1F) == hashlib.sha256(b"substation-studio-hwid-v1|g|0000001F").hexdigest()
    assert hwid.compose_legacy("g", 0x1F) != hwid.compose("g", 0x1F, "")


def test_bound_hwid_allows_legacy_only_before_cutoff():
    before, after = hwid.LEGACY_ISSUED_BEFORE - 1, hwid.LEGACY_ISSUED_BEFORE
    assert hwid.bound_hwid(LEGACY, before, HWID, LEGACY) == LEGACY
    assert hwid.bound_hwid(LEGACY, after, HWID, LEGACY) == HWID
    assert hwid.bound_hwid(HWID, before, HWID, LEGACY) == HWID
    assert hwid.bound_hwid(LEGACY, before, HWID, None) == HWID


def test_license_from_older_build_stays_valid_on_same_pc(tmp_path):
    install(tmp_path, bundle(hwid=LEGACY))
    status = service(tmp_path).status()
    assert status["licensed"] and status["mode"] == "lifetime" and not status["transfer_available"]


def test_legacy_binding_is_refused_after_cutoff_and_on_other_pcs(tmp_path):
    install(tmp_path, bundle(hwid=LEGACY, issued_at=hwid.LEGACY_ISSUED_BEFORE))
    late = service(tmp_path, now=hwid.LEGACY_ISSUED_BEFORE + 60).status()
    assert not late["licensed"] and late["reason"] == policy.WRONG_MACHINE
    install(tmp_path, bundle(hwid=LEGACY))
    other = service(tmp_path, legacy_hwid_provider=lambda: "ef" * 32).status()
    assert not other["licensed"] and other["reason"] == policy.WRONG_MACHINE


def test_online_check_asks_about_bound_legacy_id_and_keeps_old_certificate(tmp_path):
    original = bundle(hwid=LEGACY)
    install(tmp_path, original)
    RecordingClient.calls = []
    studio = service(tmp_path, now=NOW + 30 * DAY, client_factory=RecordingClient)
    assert studio.verify_online_if_due() == "valid"
    assert RecordingClient.calls == [LEGACY]
    assert (tmp_path / "home/license.cert").read_text(encoding="utf-8") == original
    assert studio.is_licensed()


def test_unreadable_legacy_id_falls_back_to_current_only(tmp_path):
    def broken():
        raise OSError("no registry")
    install(tmp_path, bundle(hwid=LEGACY))
    assert not service(tmp_path, legacy_hwid_provider=broken).is_licensed()
    install(tmp_path, bundle())
    assert service(tmp_path, legacy_hwid_provider=broken).is_licensed()
