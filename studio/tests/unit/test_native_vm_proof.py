"""The native guard accepts a virtual machine's host proof (vm_proof.json), and only that proof.

docs/architecture/license-binding-contract.md, "VM host key": the Go backend of a VM without a
license of its own writes the host's certificate, the nonce, the VM's machine ID, the host key's
signature and the time; the native workers accept it only for their own machine ID and only while the
time is at most 24 hours old. The proofs here are built by an independent Python signer.
"""
import base64
import json
import os
from pathlib import Path
import subprocess
import time
import pytest
from license_signer import bundle, public_key, sign
import hwid_v3
from studio_app.runtime.worker_process import WorkerProcess

STUDIO = Path(__file__).resolve().parents[2]
ENFORCED = Path(os.environ.get("STUDIO_ENFORCED_NATIVE_BIN", STUDIO / "native/build/license-enforced/Release"))
CHECK = ENFORCED / "studio_license_check.exe"
pytestmark = pytest.mark.skipif(not CHECK.exists(), reason="enforced native build not present")
HOST_SEED = bytes.fromhex("17" * 32)
OTHER_SEED = bytes.fromhex("29" * 32)
NONCE = "0123456789abcdef0123456789abcdef"
VM = "ab" * 32


def b64url(data):
    return base64.urlsafe_b64encode(data).decode().rstrip("=")


def host_key(seed=HOST_SEED):
    return b64url(public_key(seed))


def proof_text(now, vm_hwid=VM, nonce=NONCE, seed=HOST_SEED, age=0, signed_for=None, signed_time=None,
               certificate=None, **fields):
    """A vm_proof.json the host signed `age` seconds ago (its "time" is part of the signed message);
    signed_for and signed_time override what the host key signed, so the file disagrees with it."""
    fields.setdefault("host_key", host_key(seed))
    certificate = certificate or bundle(**fields)
    at = int(now - age)
    message = ("scadabench-vm-proof-v2|" + nonce + "|" + (signed_for or vm_hwid) + "|" +
               str(at if signed_time is None else signed_time)).encode()
    return json.dumps({"certificate": json.loads(certificate), "nonce": nonce, "vm_hwid": vm_hwid,
                       "signature": b64url(sign(message, seed)), "time": at})


def edited_time(text, **changes):
    """The same proof file with fields of it changed afterwards (what someone with a text editor does)."""
    document = json.loads(text)
    document.update(changes)
    return json.dumps(document)


def verify(tmp_path, text, machine=VM, now=None, key=None):
    now = int(now or time.time())
    path = tmp_path / "vm_proof.json"
    path.write_text(text, encoding="utf-8")
    result = subprocess.run([str(CHECK), "verify-vm", str(path), (key or public_key()).hex(), machine, str(now), str(tmp_path)],
                            capture_output=True, text=True, timeout=10)
    return result.stdout.strip()


def test_a_fresh_proof_for_this_machine_is_accepted(tmp_path):
    now = time.time()
    assert verify(tmp_path, proof_text(now), now=now) == "ok"
    assert verify(tmp_path, proof_text(now, age=23 * 3600), now=now) == "ok"
    assert verify(tmp_path, proof_text(now, age=-200), now=now) == "ok"  # a host clock a little ahead is fine
    assert verify(tmp_path, proof_text(now, age=24 * 3600), now=now) == "ok"  # exactly a day is still fine


def test_a_proof_older_than_a_day_or_from_the_future_is_refused(tmp_path):
    now = time.time()
    assert "out of date" in verify(tmp_path, proof_text(now, age=24 * 3600 + 5), now=now)
    assert "out of date" in verify(tmp_path, proof_text(now, age=-601), now=now)  # more than 10 minutes ahead
    assert verify(tmp_path, proof_text(now, age=-600), now=now) == "ok"
    assert "out of date" in verify(tmp_path, proof_text(now, age=10 * 86400), now=now)


def test_editing_the_time_of_a_proof_breaks_the_signature_so_an_old_proof_cannot_be_made_young(tmp_path):
    now = time.time()
    expired = proof_text(now, age=25 * 3600)
    assert "out of date" in verify(tmp_path, expired, now=now)
    # The attack: change "time" to now. The host signed the old time, so the signature no longer fits.
    assert "did not confirm" in verify(tmp_path, edited_time(expired, time=int(now)), now=now)
    young = proof_text(now, age=3600)
    assert verify(tmp_path, young, now=now) == "ok"
    assert "did not confirm" in verify(tmp_path, edited_time(young, time=int(now)), now=now)
    assert "did not confirm" in verify(tmp_path, edited_time(young, time=int(now - 3600) + 1), now=now)
    # A host that signed another time than the file says is the same thing.
    assert "did not confirm" in verify(tmp_path, proof_text(now, age=3600, signed_time=int(now) - 7200), now=now)


def test_a_time_that_is_not_a_whole_number_is_a_damaged_file(tmp_path):
    now = time.time()
    young = proof_text(now, age=60)
    at = json.loads(young)["time"]
    for value in (float(at), str(at), None, [at], True, 0, -1):
        assert verify(tmp_path, edited_time(young, time=value), now=now).startswith("fail"), value
    assert "damaged" in verify(tmp_path, edited_time(young, time=float(at)), now=now)


def test_setting_the_clock_back_cannot_bring_an_old_proof_back(tmp_path):
    now = int(time.time())
    # The computer saw `now` (the license clock anchor); someone then sets its clock back a day, where the
    # proof (made at "now - 20 h") would look young again.
    proof = proof_text(now - 86400, age=20 * 3600)
    assert verify(tmp_path, proof, now=now - 86400) == "ok"  # the set-back clock alone would allow it
    (tmp_path / ".time_anchor").write_text(str(now), encoding="utf-8")
    assert "rollback" in verify(tmp_path, proof, now=now - 86400)
    # The rule of the proof itself: a proof a day old by the clock is refused when the anchor is 200 s
    # further on, although that is inside the certificate's own 300 s tolerance.
    exactly_a_day = proof_text(now, age=24 * 3600)
    (tmp_path / ".time_anchor").write_text(str(now), encoding="utf-8")
    assert verify(tmp_path, exactly_a_day, now=now) == "ok"
    (tmp_path / ".time_anchor").write_text(str(now + 200), encoding="utf-8")
    assert "rollback" in verify(tmp_path, exactly_a_day, now=now)
    # Nothing newer than the proof in the anchor: no objection.
    (tmp_path / ".time_anchor").write_text(str(now - 3 * 86400), encoding="utf-8")
    assert verify(tmp_path, proof_text(now, age=3600), now=now) == "ok"


def test_a_proof_for_another_machine_is_refused_even_when_the_host_signed_it(tmp_path):
    now = time.time()
    other = "cd" * 32
    assert "another computer" in verify(tmp_path, proof_text(now, vm_hwid=other), machine=VM, now=now)
    # Edited to this machine: the host's signature was for the other one.
    forged = json.loads(proof_text(now, vm_hwid=other))
    forged["vm_hwid"] = VM
    assert "did not confirm" in verify(tmp_path, json.dumps(forged), now=now)
    assert "did not confirm" in verify(tmp_path, proof_text(now, signed_for=other), now=now)


def test_a_proof_for_another_nonce_or_signed_by_another_key_is_refused(tmp_path):
    now = time.time()
    changed = json.loads(proof_text(now))
    changed["nonce"] = "f" * 32
    assert "did not confirm" in verify(tmp_path, json.dumps(changed), now=now)
    # The certificate names one host key, the proof was signed by another.
    swapped = json.loads(proof_text(now, seed=OTHER_SEED, host_key=host_key(HOST_SEED)))
    assert "did not confirm" in verify(tmp_path, json.dumps(swapped), now=now)


def test_only_a_genuine_lifetime_license_with_a_host_key_counts(tmp_path):
    now = time.time()
    assert "damaged or was changed" in verify(tmp_path, proof_text(now, certificate=bundle(seed=OTHER_SEED, host_key=host_key())), now=now)
    assert "lifetime" in verify(tmp_path, proof_text(now, type="trial", expires_at=int(now) + 86400), now=now)
    assert "no virtual machine key" in verify(tmp_path, proof_text(now, certificate=bundle()), now=now)
    assert "no virtual machine key" in verify(tmp_path, proof_text(now, host_key=host_key()[:40]), now=now)
    assert "no virtual machine key" in verify(tmp_path, proof_text(now, host_key=""), now=now)
    assert "expired" in verify(tmp_path, proof_text(now, expires_at=int(now) - 100), now=now)
    assert "rollback" in verify(tmp_path, proof_text(now, issued_at=int(now) + 7 * 86400), now=now)
    assert "version" in verify(tmp_path, proof_text(now, version=2), now=now)
    tampered = json.loads(proof_text(now))
    tampered["certificate"]["payload"] = tampered["certificate"]["payload"].replace("SUBS-TEST", "SUBS-EVIL")
    assert "damaged or was changed" in verify(tmp_path, json.dumps(tampered), now=now)


def test_a_cancelled_hosts_license_is_refused_in_any_of_the_three_places(tmp_path):
    now = time.time()
    text = proof_text(now)
    assert verify(tmp_path, text, now=now) == "ok"
    (tmp_path / ".revoked_ids").write_text("SUBS-TEST-TEST-TEST-TEST", encoding="utf-8")
    assert "cancelled" in verify(tmp_path, text, now=now)
    (tmp_path / ".revoked_ids").unlink()
    (tmp_path / "license_state.json").write_text('{"revoked_ids": "SUBS-OTHR-OTHR-OTHR-OTHR,SUBS-TEST-TEST-TEST-TEST"}', encoding="utf-8")
    assert "cancelled" in verify(tmp_path, text, now=now)


def test_damaged_proof_files_never_crash_and_never_license(tmp_path):
    now = time.time()
    for text in ["", "not json", "[]", "{}", '{"certificate": 1}', '{"certificate": {}, "nonce": 1, "vm_hwid": 2, "signature": 3, "time": 4}',
                 json.dumps({"certificate": {}, "nonce": NONCE, "vm_hwid": VM, "signature": "AAAA", "time": int(now)}),
                 json.dumps({"certificate": json.loads(bundle(host_key=host_key())), "nonce": "XYZ", "vm_hwid": VM,
                             "signature": "AAAA", "time": int(now)})]:
        assert verify(tmp_path, text, now=now).startswith("fail"), text


# The enforced workers use the same rules through the license folder (STUDIO_LICENSE_CERT_PATH).

@pytest.fixture
def folder(tmp_path, monkeypatch):
    monkeypatch.setenv("STUDIO_LICENSE_CERT_PATH", str(tmp_path / "license.cert"))
    return tmp_path


def start_error(executable):
    worker = WorkerProcess(executable)
    try:
        worker.request("start", timeout=10, ieds=[])
    except RuntimeError as error:
        return str(error)
    finally:
        worker.close()
    return ""


def licensed(error):
    return "icense" not in error


@pytest.mark.skipif(not (ENFORCED / "studio_ied.exe").exists(), reason="enforced workers not built")
def test_an_enforced_worker_serves_a_vm_that_has_a_fresh_host_proof_and_no_license_of_its_own(folder):
    machine = hwid_v3.machine_id()
    now = time.time()
    assert "No license" in start_error(ENFORCED / "studio_ied.exe")
    (folder / "vm_proof.json").write_text(proof_text(now, vm_hwid=machine), encoding="utf-8")
    assert licensed(start_error(ENFORCED / "studio_ied.exe"))


@pytest.mark.skipif(not (ENFORCED / "studio_ied.exe").exists(), reason="enforced workers not built")
def test_an_enforced_worker_refuses_a_stale_or_foreign_or_cancelled_host_proof(folder):
    machine = hwid_v3.machine_id()
    now = time.time()
    proof = folder / "vm_proof.json"
    proof.write_text(proof_text(now, vm_hwid=machine, age=25 * 3600), encoding="utf-8")
    assert "No license" in start_error(ENFORCED / "studio_ied.exe")
    proof.write_text(proof_text(now, vm_hwid="cd" * 32), encoding="utf-8")
    assert "No license" in start_error(ENFORCED / "studio_ied.exe")
    proof.write_text(proof_text(now, vm_hwid=machine), encoding="utf-8")
    assert licensed(start_error(ENFORCED / "studio_ied.exe"))
    (folder / ".revoked_ids").write_text("SUBS-TEST-TEST-TEST-TEST", encoding="utf-8")
    assert "No license" in start_error(ENFORCED / "studio_ied.exe")


@pytest.mark.skipif(not (ENFORCED / "studio_ied.exe").exists(), reason="enforced workers not built")
def test_a_certificate_for_another_computer_does_not_stop_a_valid_host_proof_and_a_bad_proof_does_not_stop_an_own_license(folder):
    machine = hwid_v3.machine_id()
    now = time.time()
    certificate = folder / "license.cert"
    certificate.write_text(bundle(hwid="cd" * 32), encoding="utf-8")  # a copied license of another PC
    assert "another computer" in start_error(ENFORCED / "studio_ied.exe")
    (folder / "vm_proof.json").write_text(proof_text(now, vm_hwid=machine), encoding="utf-8")
    assert licensed(start_error(ENFORCED / "studio_ied.exe"))
    certificate.write_text(bundle(hwid=machine), encoding="utf-8")  # an own license wins whatever the proof says
    (folder / "vm_proof.json").write_text("garbage", encoding="utf-8")
    assert licensed(start_error(ENFORCED / "studio_ied.exe"))
