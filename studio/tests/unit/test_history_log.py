from argparse import Namespace
from datetime import datetime, timezone

from studio.tools.history_log import append_record, check, read_records


def values(identifier, previous_file=""):
    files = [f"{previous_file}=Updated contract"] if previous_file else []
    return Namespace(id=identifier, intent="Keep future work traceable",
                     reason="Agents need a compact immutable record", file=files,
                     expected=["A query finds the change"], evidence=["unit test"],
                     tag=["docs", "traceability"])


def test_append_builds_a_valid_hash_chain_and_rotates(monkeypatch, tmp_path):
    import studio.tools.history_log as history
    monkeypatch.setattr(history, "MAX_SEGMENT_LINES", 1)
    first = append_record(tmp_path, values("first"), datetime(2026, 1, 1, tzinfo=timezone.utc))
    second = append_record(tmp_path, values("second", "docs/INDEX.md"),
                           datetime(2026, 1, 2, tzinfo=timezone.utc))
    assert second["prev"] == first["hash"]
    assert [path.name for path in tmp_path.glob("ledger-*.jsonl")] == [
        "ledger-0001.jsonl", "ledger-0002.jsonl"]
    assert check(tmp_path, enforce_git=False) == []


def test_editing_a_prior_record_breaks_verification(tmp_path):
    append_record(tmp_path, values("first"))
    path = tmp_path / "ledger-0001.jsonl"
    path.write_text(path.read_text(encoding="utf-8").replace("future", "past"), encoding="utf-8")
    assert any("hash does not match" in error for error in check(tmp_path, enforce_git=False))
    assert len(list(read_records(tmp_path))) == 1
