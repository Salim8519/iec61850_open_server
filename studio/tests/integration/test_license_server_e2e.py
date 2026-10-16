"""Python client against the real Go license server: trial, activate, move, revoke check."""
import json
import os
from pathlib import Path
import shutil
import socket
import subprocess
import time
from urllib.request import urlopen
import pytest
from studio_app.licensing.client import LicenseServerError
from studio_app.licensing.release_key import key_id
from studio_app.licensing.service import LicenseService

ROOT = Path(__file__).resolve().parents[3]
SERVER_SOURCE = ROOT / "license-server"
pytestmark = pytest.mark.skipif(shutil.which("go") is None, reason="Go toolchain not installed")


def free_port():
    with socket.socket() as listener:
        listener.bind(("127.0.0.1", 0))
        return listener.getsockname()[1]


@pytest.fixture(scope="module")
def server(tmp_path_factory):
    work = tmp_path_factory.mktemp("license-server")
    binary = work / "license-server.exe"
    subprocess.run(["go", "build", "-o", str(binary), "./cmd/license-server"], cwd=SERVER_SOURCE, check=True)
    data = work / "data"
    port = free_port()
    env = {**os.environ, "LICENSE_DATA_DIR": str(data), "LICENSE_LISTEN_ADDR": f"127.0.0.1:{port}",
           "LICENSE_POW_DIFFICULTY": "10", "LICENSE_RATE_SENSITIVE": "100"}
    created = subprocess.run([str(binary), "keygen"], env=env, capture_output=True, text=True, check=True)
    public_hex = created.stdout.strip().rsplit(" ", 1)[-1]
    process = subprocess.Popen([str(binary), "serve"], env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    url = f"http://127.0.0.1:{port}"
    deadline = time.monotonic() + 15
    while time.monotonic() < deadline:
        try:
            with urlopen(url + "/api/v1/health", timeout=1) as response:
                assert json.loads(response.read())["ok"]
                break
        except OSError:
            time.sleep(0.2)
    else:
        process.kill()
        pytest.fail("license server did not start")
    public = bytes.fromhex(public_hex)
    yield {"url": url, "binary": binary, "env": env, "keys": {key_id(public): public}}
    process.terminate()
    process.wait(timeout=10)


def machine(tmp_path, server, name, hwid):
    studio = LicenseService(tmp_path / name, tmp_path / (name + "-data"), True, hwid_provider=lambda: hwid,
                            keys=server["keys"])
    studio.set_server_url(server["url"])
    return studio


def issue_key(server):
    result = subprocess.run([str(server["binary"]), "issue", "--email", "buyer@example.com"], env=server["env"],
                            capture_output=True, text=True, check=True)
    return json.loads(result.stdout)["license"]["key"]


def test_trial_is_granted_once_per_machine(tmp_path, server):
    first = machine(tmp_path, server, "pc1", "11" * 32)
    status = first.start_trial()
    assert status["licensed"] and status["mode"] == "trial" and status["days_remaining"] == 7
    reinstall = machine(tmp_path, server, "pc1-reinstalled", "11" * 32)
    assert reinstall.start_trial()["expires_at"] == status["expires_at"]


def test_lifetime_key_locks_to_one_pc_and_can_move(tmp_path, server):
    key = issue_key(server)
    first = machine(tmp_path, server, "pcA", "aa" * 32)
    assert first.activate(key.lower())["mode"] == "lifetime"
    second = machine(tmp_path, server, "pcB", "bb" * 32)
    with pytest.raises(LicenseServerError) as refused:
        second.activate(key)
    assert refused.value.code == "bound_elsewhere"
    assert second.activate(key, move=True)["licensed"]
    first.state_file.write_text(json.dumps({"last_verify": 0}))
    assert first.verify_online_if_due() == "moved"
    assert not first.is_licensed() and "moved" in first.status()["reason"]


def test_unknown_key_is_refused(tmp_path, server):
    studio = machine(tmp_path, server, "pcC", "cc" * 32)
    with pytest.raises(LicenseServerError) as refused:
        studio.activate("SUBS-2345-6789-ABCD-EFGH")
    assert refused.value.code == "unknown_key"
