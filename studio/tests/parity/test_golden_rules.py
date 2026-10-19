"""Golden replay rules: privacy guard, normaliser and diff classes (harness self-tests)."""
import json

import pytest

from parity.replay.diff import compare
from parity.replay.jsonlex import loads
from parity.replay.manifest import MANIFEST, Manifest, PrivacyError, build
from parity.replay.normalise import Normaliser
from parity.replay.record import Recorder


def test_manifest_lists_only_public_inputs_with_current_hashes():
    document = build()
    assert {entry["alias"] for entry in document["inputs"]} >= {"demo", "model_substation"}
    assert not any("test_cids" in (entry["path"] or "") for entry in document["inputs"])
    assert document == json.loads(MANIFEST.read_text(encoding="utf-8"))


def test_recorder_refuses_an_unlisted_upload():
    manifest = Manifest()
    with pytest.raises(PrivacyError):
        manifest.check_upload(b"<SCL>private customer substation</SCL>", "upload")
    with pytest.raises(PrivacyError):
        manifest.load("customer")
    with pytest.raises(PrivacyError):
        manifest.check_names(["FEED1", "CustomerRelay7"])
    manifest.check_names(["FEED1", "Feeder01"], allowed_extra=["Feeder01"])


def test_refused_recording_leaves_no_transcript(tmp_path):
    output = tmp_path / "refused.jsonl"
    output.write_text("partial", encoding="utf-8")
    scenario = {"name": "refused", "steps": [
        {"req": "POST /api/v1/session", "json": {"token": "golden-session-token"}},
        {"req": "POST /api/v1/scl/preview", "multipart": {"file": "@customer"}}]}
    with pytest.raises(PrivacyError):
        Recorder("python-http").run(scenario, output)
    assert not output.exists()


def test_normaliser_keeps_everything_but_ids_times_and_paths():
    context = {"data": "C:\\tmp\\run\\data", "port": 51234}
    normaliser = Normaliser(context)
    tree = normaliser.tree(loads(json.dumps({
        "id": "0123456789abcdef0123456789abcdef", "timestamp": 1760000000.5, "elapsed_ms": 12,
        "path": "C:\\tmp\\run\\data\\runtime\\0123456789abcdef0123456789abcdef.cfg",
        "url": "http://127.0.0.1:51234/x", "value": 1.0, "text": "ok"})))
    assert tree == {"id": "<uuid:1>", "timestamp": "<ts>", "elapsed_ms": "<dur:int>",
                    "path": "<data>\\runtime\\<uuid:1>.cfg", "url": "http://127.0.0.1:<port>/x",
                    "value": "1.0", "text": "ok"}


def test_diff_classes():
    def classes(left, right):
        return [item[0] for item in compare(loads(left), loads(right))]
    assert classes('{"a": 1}', '{"a": 1.0}') == ["numtype"]
    assert classes('{"a": 1, "b": 2}', '{"b": 2, "a": 1}') == ["order"]
    assert classes('{"a": 1}', '{"a": 1, "b": 2}') == ["extra"]
    assert classes('[1, 2]', '[1]') == ["missing"]
    assert classes('{"a": true}', '{"a": 1}') == ["value"]
    assert classes('{"a": "x"}', '{"a": "x"}') == []
