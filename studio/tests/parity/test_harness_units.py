"""Harness building blocks without an engine process: wire codec, live views, client encoding."""
from pathlib import Path

import pytest

from parity import wire
from parity.errors import InProcessOnly
from parity.http_client import _body
from parity.live_view import LiveRoot, wrap


def test_wire_round_trips_python_only_types():
    value = {"b": b"\x00\xff", "s": {"x", "y"}, "t": (1, 2.5), "p": Path("C:/data"), "k": {1: "int key"}}
    assert wire.loads(wire.dumps(value)) == value
    error = wire.decode_exception(wire.encode_exception(PermissionError("injected")))
    assert isinstance(error, PermissionError) and str(error) == "injected"


def test_handles_delegate_to_the_original():
    handle = wire.decode(wire.encode(object()))
    with pytest.raises(wire.DelegateToOriginal):
        handle.directory


class FakeEngine:
    def __init__(self, project):
        self.project, self.calls = project, []

    def parity(self, method, operation, body=None, **query):
        self.calls.append((method, operation, body))
        if operation == "project/raw":
            return self.project
        if operation == "project/set":
            target = self.project
            for step in body["path"]:
                target = next(item for item in target if item.get("id") == step["id"]) if isinstance(step, dict) \
                    else target[step]
            target[body["key"]] = body["value"]
        if operation == "project/update":
            next(item for item in self.project["ieds"] if item["id"] == body["path"][1]["id"]).update(body["values"])


def test_live_views_read_fresh_and_write_through():
    engine = FakeEngine({"name": "A", "ieds": [{"id": "x1", "ip": "127.50.0.2", "signals": [1, 2]}]})
    project = wrap({}, LiveRoot(engine), [])
    ied = project["ieds"][0]
    engine.project["ieds"][0]["ip"] = "127.50.0.9"  # an engine-side change after the read
    assert ied["ip"] == "127.50.0.9"
    ied.update(port=18113)
    assert engine.project["ieds"][0]["port"] == 18113
    project["name"] = "Changed"
    assert engine.project["name"] == "Changed" and project == engine.project
    assert [item["id"] for item in project["ieds"]] == ["x1"] and project["ieds"][0]["signals"].index(2) == 1
    with pytest.raises(InProcessOnly):
        project["ieds"].append({})


def test_json_and_multipart_bodies_follow_the_flask_test_client():
    data, kind = _body({"b": 1, "a": "\u00e9"}, None, None)
    assert data == b'{"a": "\\u00e9", "b": 1}' and kind == "application/json"
    import io
    data, kind = _body(None, {"file": (io.BytesIO(b"<SCL/>"), "x.scd"), "selected_names": "[]"}, None)
    assert kind.startswith("multipart/form-data; boundary=") and b'filename="x.scd"' in data
