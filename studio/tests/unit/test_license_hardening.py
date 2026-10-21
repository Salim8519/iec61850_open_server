"""Cancelled stays cancelled (ADR-0097): a renamed certificate and deleted copies never re-license a PC."""
import json
import pytest
from license_fakes import DAY, NOW, FakeRegistry, Server, install_trial, studio
from license_signer import bundle
from studio_app.domain import license_policy as policy
from studio_app.licensing.service import LicenseActionError

LICENSE_ID = "SUBS-TEST-TEST-TEST-TEST"
LIFETIME = bundle()


def install(tmp_path, text=LIFETIME):
    home = tmp_path / "home"
    home.mkdir(parents=True, exist_ok=True)
    (home / "license.cert").write_text(text, encoding="utf-8")


def cancelled(tmp_path, registry=None, verdict="revoked"):
    """A lifetime license that the server just cancelled; returns the service and the clock."""
    install(tmp_path)
    at = [NOW + 30 * DAY]
    app = studio(tmp_path, at, Server(time=at[0], verdict=verdict), registry)
    assert app.verify_online_if_due() == verdict
    return app, at


def stays_cancelled(tmp_path, at, registry=None):
    fresh = studio(tmp_path, at, Server(online=False), registry)
    status = fresh.status()
    assert not status["licensed"] and status["reason"] == policy.REVOKED and status["plan"] == "lifetime"
    return fresh


@pytest.mark.parametrize("verdict", ["revoked", "moved"])
def test_renaming_a_cancelled_certificate_back_does_not_license_the_pc_again(tmp_path, verdict):
    registry = FakeRegistry()
    app, at = cancelled(tmp_path, registry, verdict)
    home = tmp_path / "home"
    assert not (home / "license.cert").exists()
    (aside,) = home.glob(f"license.cert.{verdict}-*")
    aside.rename(home / "license.cert")  # the cheat: put the file back
    assert (home / "license.cert").read_text(encoding="utf-8") == LIFETIME
    stays_cancelled(tmp_path, at, registry)
    assert not app.status()["licensed"] and not app.is_licensed()
    assert "cancelled" in app.status()["reason"]


def test_a_cancelled_id_is_kept_in_the_file_the_registry_and_the_state_file(tmp_path):
    registry = FakeRegistry()
    cancelled(tmp_path, registry)
    home = tmp_path / "home"
    assert (home / ".revoked_ids").read_text(encoding="utf-8") == LICENSE_ID
    assert registry.values["RevokedLicenses"] == LICENSE_ID
    assert json.loads((home / "license_state.json").read_text(encoding="utf-8"))["revoked_ids"] == LICENSE_ID


def delete_file(tmp_path, registry):
    (tmp_path / "home/.revoked_ids").unlink()


def delete_registry(tmp_path, registry):
    del registry.values["RevokedLicenses"]


def delete_state(tmp_path, registry):
    state = tmp_path / "home/license_state.json"
    data = json.loads(state.read_text(encoding="utf-8"))
    del data["revoked_ids"]
    state.write_text(json.dumps(data), encoding="utf-8")


def delete_two(tmp_path, registry):
    delete_file(tmp_path, registry)
    delete_registry(tmp_path, registry)


@pytest.mark.parametrize("delete", [delete_file, delete_registry, delete_state, delete_two])
def test_a_deleted_copy_is_restored_from_the_others_and_never_frees_the_license(tmp_path, delete):
    registry = FakeRegistry()
    app, at = cancelled(tmp_path, registry)
    install(tmp_path)  # certificate renamed back as well
    delete(tmp_path, registry)
    fresh = stays_cancelled(tmp_path, at, registry)
    home = tmp_path / "home"
    assert (home / ".revoked_ids").read_text(encoding="utf-8") == LICENSE_ID
    assert registry.values["RevokedLicenses"] == LICENSE_ID
    assert json.loads((home / "license_state.json").read_text(encoding="utf-8"))["revoked_ids"] == LICENSE_ID
    assert not fresh.is_licensed()


def test_without_a_registry_two_copies_are_enough(tmp_path):
    app, at = cancelled(tmp_path)
    install(tmp_path)
    delete_file(tmp_path, None)
    stays_cancelled(tmp_path, at)
    assert (tmp_path / "home/.revoked_ids").exists()


def test_other_licenses_are_not_affected_by_a_cancelled_one(tmp_path):
    registry = FakeRegistry()
    cancelled(tmp_path, registry)
    install(tmp_path, bundle(license_id="SUBS-OTHR-OTHR-OTHR-OTHR"))
    assert studio(tmp_path, [NOW + 30 * DAY], Server(online=False), registry).is_licensed()


def test_a_cancelled_license_cannot_be_installed_again(tmp_path):
    registry = FakeRegistry()
    app, at = cancelled(tmp_path, registry)
    with pytest.raises(LicenseActionError) as refused:
        app.activate_offline(LIFETIME)
    assert str(refused.value) == policy.REVOKED and not (tmp_path / "home/license.cert").exists()


def test_the_cancelled_ids_only_ever_grow(tmp_path):
    registry = FakeRegistry()
    cancelled(tmp_path, registry)
    install(tmp_path, bundle(license_id="SUBS-OTHR-OTHR-OTHR-OTHR"))
    at = [NOW + 60 * DAY]
    other = studio(tmp_path, at, Server(time=at[0], verdict="revoked"), registry)
    assert other.verify_online_if_due() == "revoked"
    assert registry.values["RevokedLicenses"] == f"SUBS-OTHR-OTHR-OTHR-OTHR,{LICENSE_ID}"  # sorted, nothing dropped


def test_a_cancelled_trial_stays_cancelled_too(tmp_path):
    registry = FakeRegistry()
    install_trial(tmp_path)
    at = [NOW + DAY]
    app = studio(tmp_path, at, Server(time=at[0], verdict="revoked"), registry)
    assert app.check_online_now()["reason"] == policy.REVOKED
    (aside,) = (tmp_path / "home").glob("license.cert.revoked-*")
    aside.rename(tmp_path / "home/license.cert")
    assert not studio(tmp_path, at, Server(online=False), registry).is_licensed()
