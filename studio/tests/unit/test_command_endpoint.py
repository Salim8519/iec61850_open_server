"""Internal commands go to the address the IED worker listens on, in lab mode too (ADR-0069)."""
import json
import subprocess
from types import SimpleNamespace
import pytest
from studio_app.gateway.commands import endpoint as gateway_endpoint
from studio_app.runtime import mms_client, supervisor as supervisor_module
from studio_app.runtime.supervisor import Supervisor
from studio_app.runtime.worker_settings import REQUIRED_CAPABILITIES

LAB = {"mode": "lab", "adapter_guid": "{lab}", "adapter_label": "Ethernet0"}


class FakeWorker:
    def __init__(self, executable, on_event):
        self.requests = []

    def request(self, op, **values):
        self.requests.append((op, values))
        if op == "start":
            return {"running": True, "ip": values["ip"], "port": values["port"],
                    "capabilities": sorted(REQUIRED_CAPABILITIES)}
        return {}

    def close(self):
        pass


def lab_ied():
    return {"id": "c02", "name": "C02_BCU", "ip": "127.50.0.9", "port": 102, "lab_ip": "192.168.230.20",
            "cfg": "MODEL(C02_BCU){}", "controls": [{"ref": "C02_BCUSystem/Bkr0XCBR1.Pos"}],
            "signals": [], "edition": 2, "state": "stopped"}


@pytest.fixture
def supervisor(tmp_path, monkeypatch):
    executable = tmp_path / "studio_ied.exe"
    executable.write_bytes(b"")
    monkeypatch.setattr(supervisor_module, "WorkerProcess", FakeWorker)
    monkeypatch.setattr(supervisor_module, "preflight", lambda ieds, network, allowed: None)
    return Supervisor(executable, tmp_path / "runtime", lambda key, event: None,
                      lab_addresses=lambda guid: ["192.168.230.20"])


def test_supervisor_remembers_the_lab_endpoint_the_worker_confirmed(supervisor):
    ied = lab_ied()
    supervisor.start([ied], LAB)
    assert supervisor.endpoint("c02") == ("192.168.230.20", 102)
    start = supervisor.workers["c02"].requests[0][1]
    assert "compatibility" not in json.dumps(start["controls"])
    supervisor.stop([ied])
    with pytest.raises(ValueError, match="not running"):
        supervisor.endpoint("c02")


def test_internal_operate_dials_the_listening_endpoint_not_the_loopback_address(supervisor, monkeypatch):
    ied = lab_ied()
    supervisor.start([ied], LAB)
    (supervisor.executable.with_name("studio_probe.exe")).write_bytes(b"")
    calls = []

    def run(arguments, **options):
        calls.append(arguments)
        return subprocess.CompletedProcess(arguments, 0, stdout='{"ok": true}', stderr="")

    monkeypatch.setattr(mms_client.subprocess, "run", run)
    result = mms_client.operate(supervisor.executable, ied, "C02_BCUSystem/Bkr0XCBR1.Pos", True,
                                supervisor.endpoint("c02"))
    assert result == {"ok": True}
    assert calls[0][1:3] == ["192.168.230.20", "102"]
    assert "127.50.0.9" not in calls[0]


def test_failed_internal_operate_names_the_address_it_tried(supervisor, monkeypatch):
    ied = lab_ied()
    supervisor.start([ied], LAB)
    (supervisor.executable.with_name("studio_probe.exe")).write_bytes(b"")
    monkeypatch.setattr(mms_client.subprocess, "run", lambda arguments, **options:
                        subprocess.CompletedProcess(arguments, 1, stdout='{"ok": false}', stderr=""))
    with pytest.raises(RuntimeError, match="192.168.230.20:102"):
        mms_client.operate(supervisor.executable, ied, "C02_BCUSystem/Bkr0XCBR1.Pos", False,
                           supervisor.endpoint("c02"))


def test_gateway_commands_a_simulated_ied_where_it_listens(supervisor):
    ied = lab_ied()
    supervisor.start([ied], LAB)
    coordinator = SimpleNamespace(supervisor=supervisor)
    assert gateway_endpoint({**ied, "role": "simulated"}, coordinator) == ("192.168.230.20", 102)
    real = {**ied, "role": "real", "live": {"ip": "10.0.0.5", "port": 102}}
    assert gateway_endpoint(real) == ("10.0.0.5", 102)
