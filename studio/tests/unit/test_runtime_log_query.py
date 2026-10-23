from argparse import Namespace

from studio.tools.query_runtime_logs import query
from studio_app.observability.log_writer import RunLog


def arguments(**values):
    defaults = {"run": None, "kind": None, "ied": None, "ref": None,
                "text": None, "limit": 50, "lifecycle": False}
    return Namespace(**{**defaults, **values})


def test_query_filters_events_and_lifecycle_without_mutation(tmp_path):
    log = RunLog(tmp_path)
    log.write({"kind": "control_applied", "ied_id": "ied-1", "ref": "LD/X.Pos"})
    log.write({"kind": "signal", "ied_id": "ied-1", "ref": "LD/X.Pos.stVal"})
    log.close()

    events = query(tmp_path, arguments(run=log.run_id, kind="control_applied"))
    assert len(events) == 1 and events[0]["ref"] == "LD/X.Pos"
    lifecycle = query(tmp_path, arguments(run=log.run_id, lifecycle=True))
    assert [item["kind"] for item in lifecycle] == [
        "run_started", "segment_opened", "run_closed"]


def test_ref_also_finds_generator_summaries(tmp_path):
    """ADR-0115: a signal_summary holding the point matches --ref like its samples did."""
    run = tmp_path / "run1"
    run.mkdir()
    (run / "events-0000.jsonl").write_text(
        '{"run_id":"run1","kind":"signal_summary","ied_id":"a","source":"generator","second":1,"samples":10,'
        '"from_id":1,"to_id":10,"values":{"LD/MMXU1.A.mag.f":1.5}}\n'
        '{"run_id":"run1","kind":"signal","ied_id":"a","ref":"LD/GGIO1.Ind1.stVal","value":true}\n', encoding="utf-8")
    found = query(tmp_path, arguments(run="run1", ref="LD/MMXU1.A.mag.f"))
    assert [item["kind"] for item in found] == ["signal_summary"]
