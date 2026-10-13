import json

from studio_app.observability import log_writer
from studio_app.observability.log_writer import RunLog


def lines(path):
    return [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines()]


def test_run_log_uses_compact_append_only_lifecycle_journals(tmp_path):
    log = RunLog(tmp_path)
    log.write({"kind": "signal", "id": 1, "value": 12})
    log.close()

    record_text = (log.path / "events-0000.jsonl").read_text(encoding="utf-8")
    assert '"run_id":"' in record_text and '"value":12' in record_text
    assert lines(log.path / "manifest.jsonl")[0]["kind"] == "run_started"
    assert [item["kind"] for item in lines(log.path / "manifest.jsonl")] == [
        "run_started", "segment_opened", "run_closed"]
    assert [item["kind"] for item in lines(tmp_path / "runs.jsonl")] == [
        "run_started", "run_closed"]
    assert not (log.path / "manifest.json").exists()


def test_rotation_retains_every_segment_and_appends_index(monkeypatch, tmp_path):
    monkeypatch.setattr(log_writer, "SEGMENT_BYTES", 100)
    log = RunLog(tmp_path)
    log.write({"kind": "signal", "id": 1, "value": "a" * 40})
    log.write({"kind": "signal", "id": 2, "value": "b" * 40})
    log.close()

    assert (log.path / "events-0000.jsonl").exists()
    assert (log.path / "events-0001.jsonl").exists()
    opened = [item["file"] for item in lines(log.path / "manifest.jsonl")
              if item["kind"] == "segment_opened"]
    assert opened == ["events-0000.jsonl", "events-0001.jsonl"]
