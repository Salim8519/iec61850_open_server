"""Load-test identities cover executable/source changes and exclude customer XML."""
from pathlib import Path


def test_provenance_identifies_changed_files_without_customer_data(tmp_path, monkeypatch):
    monkeypatch.syspath_prepend(str(Path(__file__).resolve().parents[2] / "tools"))
    from benchmark_provenance import capture, changed_files
    backend = tmp_path / "backend"
    backend.mkdir()
    source = backend / "clock.py"
    source.write_text("PERIOD = 0.1\n")
    (backend / "customer.cid").write_text("private engineering data")
    binary = tmp_path / "native/build/Release/studio_ied.exe"
    binary.parent.mkdir(parents=True)
    binary.write_bytes(b"test binary")
    before = capture(tmp_path)
    assert set(before["files"]) == {"backend/clock.py", "native/build/Release/studio_ied.exe"}
    assert capture(tmp_path)["sha256"] == before["sha256"]
    source.write_text("PERIOD = 0.2\n")
    after = capture(tmp_path)
    assert after["sha256"] != before["sha256"]
    assert changed_files(before, after) == ["backend/clock.py"]
    binary.write_bytes(b"changed test binary")
    assert changed_files(after, capture(tmp_path)) == ["native/build/Release/studio_ied.exe"]
