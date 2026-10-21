"""Only a signed server answer counts (ADR-0097): unsigned, forged or replayed answers are ignored."""
import json
import re
import pytest
from license_fakes import DAY, NOW, Server, install_trial, studio
from license_signer import HWID, TEST_SEED, bundle, keys, verify_answer
from studio_app.domain import license_policy as policy
from studio_app.licensing import client
from studio_app.licensing.client import LicenseServerError
from studio_app.licensing.service import LicenseService

LIFETIME = bundle()


def install(tmp_path, text=LIFETIME):
    home = tmp_path / "home"
    home.mkdir(parents=True, exist_ok=True)
    (home / "license.cert").write_text(text, encoding="utf-8")


def forged(changes):
    """A client factory whose verify answer is altered as `changes` says."""
    class Forger:
        def __init__(self, url):
            self.url = url

        def verify(self, key, machine, nonce):
            fields = {"nonce": nonce, "license_id": key, "machine": machine, "status": "revoked"}
            fields.update(changes.get("fields", {}))
            extra = changes.get("payload", {})
            signed = verify_answer(fields["nonce"], fields["license_id"], fields["machine"], fields["status"],
                                   seed=changes.get("seed", TEST_SEED), **extra)
            return {"status": "revoked", "answer": changes.get("answer", signed)}
    return Forger


BAD_ANSWERS = {
    "missing": {"answer": None},
    "not an object": {"answer": "revoked"},
    "payload not text": {"answer": {"payload": {"status": "revoked"}, "signature": "AAAA"}},
    "other signer": {"seed": bytes(32)},
    "wrong nonce": {"fields": {"nonce": "f" * 32}},
    "wrong license": {"fields": {"license_id": "SUBS-OTHR-OTHR-OTHR-OTHR"}},
    "wrong machine": {"fields": {"machine": "cd" * 32}},
    "wrong purpose": {"payload": {"purpose": "server-time"}},
    "wrong version": {"payload": {"version": 2}},
    "version as decimal": {"payload": {"version": 1.0}},
    "version as text": {"payload": {"version": "1"}},
    "unknown status": {"fields": {"status": "banned"}},
    "status not text": {"fields": {"status": ["revoked"]}},
    "answer with no signature": {"answer": {"payload": "{}"}},
}


@pytest.mark.parametrize("name", sorted(BAD_ANSWERS))
def test_an_unsigned_or_forged_answer_is_treated_like_an_unreachable_server(tmp_path, name):
    install(tmp_path)
    service = LicenseService(tmp_path / "home", tmp_path / "data", True, hwid_provider=lambda: HWID,
                             clock=lambda: NOW + 30 * DAY, keys=keys(), parts_provider=lambda: {},
                             client_factory=forged(BAD_ANSWERS[name]))
    assert service.verify_online_if_due() is None
    assert service.is_licensed() and (tmp_path / "home/license.cert").read_text(encoding="utf-8") == LIFETIME
    assert not list((tmp_path / "home").glob("license.cert.*")) and not (tmp_path / "home/.revoked_ids").exists()
    state = json.loads((tmp_path / "home/license_state.json").read_text(encoding="utf-8"))
    assert "last_verify" not in state  # the check did not count as done


def test_the_unsigned_top_level_status_is_never_trusted(tmp_path):
    install(tmp_path)

    class Proxy:
        def __init__(self, url):
            pass

        def verify(self, key, machine, nonce):  # a hosts-file redirect shouting "revoked"
            return {"status": "revoked", "certificate": {}}
    service = LicenseService(tmp_path / "home", tmp_path / "data", True, hwid_provider=lambda: HWID,
                             clock=lambda: NOW + 30 * DAY, keys=keys(), client_factory=Proxy)
    assert service.verify_online_if_due() is None and service.is_licensed()


def test_the_signed_status_wins_over_the_unsigned_one(tmp_path):
    install(tmp_path)

    class Honest:
        def __init__(self, url):
            pass

        def verify(self, key, machine, nonce):
            return {"status": "valid", "answer": verify_answer(nonce, key, machine, "revoked")}
    service = LicenseService(tmp_path / "home", tmp_path / "data", True, hwid_provider=lambda: HWID,
                             clock=lambda: NOW + 30 * DAY, keys=keys(), client_factory=Honest)
    assert service.verify_online_if_due() == "revoked" and not service.is_licensed()


def test_a_signed_valid_answer_is_remembered_as_the_last_check(tmp_path):
    install(tmp_path)
    at = [NOW + 30 * DAY]
    app = studio(tmp_path, at, Server(time=at[0]))
    assert app.verify_online_if_due() == "valid"
    state = json.loads((tmp_path / "home/license_state.json").read_text(encoding="utf-8"))
    assert state["last_verify"] == at[0] and app.is_licensed()


def test_every_check_sends_a_fresh_random_nonce(tmp_path):
    seen = []

    class Recorder:
        def __init__(self, url):
            pass

        def verify(self, key, machine, nonce):
            seen.append(nonce)
            return {"answer": verify_answer(nonce, key, machine, "valid")}
    install(tmp_path)
    for step in range(3):
        service = LicenseService(tmp_path / "home", tmp_path / "data", True, hwid_provider=lambda: HWID,
                                 clock=lambda step=step: NOW + (30 + 10 * step) * DAY, keys=keys(),
                                 client_factory=Recorder)
        service.verify_online_if_due()
    assert len(set(seen)) == 3 and all(re.fullmatch(r"[0-9a-f]{32}", nonce) for nonce in seen)


def test_the_client_puts_the_nonce_in_the_verify_body():
    sent = {}

    def opener(request, timeout):
        sent.update(json.loads(request.data), path=request.full_url.rsplit("/api/v1/", 1)[-1])
        import io
        return io.BytesIO(b"{}")
    client.LicenseClient("https://license.example.com", opener, test_server=True).verify("SUBS-X", "ab" * 32, "0" * 32)
    assert sent == {"license_key": "SUBS-X", "hwid": "ab" * 32, "nonce": "0" * 32, "path": "verify"}


def test_a_trial_needs_a_signed_answer_to_confirm_its_date(tmp_path):
    install_trial(tmp_path)
    at = [NOW + DAY]
    app = studio(tmp_path, at, Server(time=at[0] + 5, signs_answers=False))
    assert app.verify_online_if_due() is None
    with pytest.raises(LicenseServerError) as refused:
        app.check_online_now()
    assert refused.value.code == "bad_answer" and str(refused.value) == policy.BAD_ANSWER
    assert app.trial_clock.confirmed() == 0  # nothing from an unsigned answer is remembered
    assert app.is_licensed()


def test_a_trial_confirms_its_date_with_a_signed_answer(tmp_path):
    install_trial(tmp_path)
    at = [NOW + DAY]
    app = studio(tmp_path, at, Server(time=at[0] + 5))
    assert app.verify_online_if_due() == "confirmed" and app.trial_clock.confirmed() == at[0] + 5
