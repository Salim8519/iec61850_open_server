"""IED Explorer connect/disconnect against a stand-in live worker that keeps one association, like studio_live."""
import threading
import time

import pytest
from studio_app.tools import explorer
from studio_app.tools.explorer import ExplorerService


class Alive:
    def poll(self):
        return None


class FakeLiveWorker:
    """Mimics MmsSession: a second connect while associated answers "Already connected"."""
    instances = []

    def __init__(self, executable, on_event):
        self.process = Alive()
        self.associated = None
        self.refuse = set()
        self.ops = []
        FakeLiveWorker.instances.append(self)

    def request(self, op, timeout=30, **values):
        self.ops.append(op)
        if op == "hello":
            return {"capabilities": ["explore"]}
        if op == "disconnect":
            self.associated = None
            return True
        if op == "connect":
            time.sleep(0.05)
            if self.associated:
                raise RuntimeError("Already connected")
            if values["ip"] in self.refuse:
                raise RuntimeError(f"Could not connect to {values['ip']}:{values['port']} (MMS error 1)")
            self.associated = (values["ip"], values["port"])
            return {"connected": True}
        raise AssertionError(op)

    def close(self):
        pass


@pytest.fixture
def service(tmp_path, monkeypatch):
    FakeLiveWorker.instances = []
    monkeypatch.setattr(explorer, "WorkerProcess", FakeLiveWorker)
    executable = tmp_path / "studio_live.exe"
    executable.write_bytes(b"")
    return ExplorerService(executable)


def test_connect_after_a_failed_connect_resets_the_worker(service):
    service.connect("10.0.0.5")
    worker = FakeLiveWorker.instances[0]
    service.state = "failed"  # what an overlapping connect used to leave behind while the worker stayed associated
    assert service.connect("10.0.0.6")["state"] == "connected"
    assert worker.associated == ("10.0.0.6", 102)
    assert len(FakeLiveWorker.instances) == 1


def test_refused_device_then_retry_connects(service):
    service._worker().refuse.add("10.0.0.7")
    with pytest.raises(RuntimeError, match="Could not open an IEC 61850 association"):
        service.connect("10.0.0.7")
    assert service.status()["state"] == "failed"
    service.process.refuse.clear()
    assert service.connect("10.0.0.7")["state"] == "connected"


def test_overlapping_connects_to_one_device_both_succeed(service):
    errors, threads = [], []
    for _ in range(2):  # React StrictMode in development runs the "open from Find IEDs" effect twice
        thread = threading.Thread(target=lambda: _try(service, errors))
        threads.append(thread)
        thread.start()
    for thread in threads:
        thread.join(5)
    assert errors == []
    assert service.status()["state"] == "connected"
    assert service.process.ops.count("connect") == 1


def test_disconnect_always_clears_the_worker_association(service):
    service.connect("10.0.0.5")
    service.state = "failed"
    assert service.disconnect()["state"] == "disconnected"
    assert service.process.associated is None


def _try(service, errors):
    try:
        service.connect("10.0.0.5")
    except Exception as error:
        errors.append(error)
