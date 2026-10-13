"""Independent launchers must not own the same data directory simultaneously."""
import os
from pathlib import Path
import subprocess
import sys
from studio_app.launcher.directory_lease import DirectoryLease


def test_os_lock_blocks_another_process_and_releases(tmp_path):
    code = "from pathlib import Path; from studio_app.launcher.directory_lease import DirectoryLease; " \
           "lease = DirectoryLease(Path(__import__('sys').argv[1])); lease.__enter__(); lease.__exit__()"
    environment = {**os.environ, "PYTHONPATH": str(Path(__file__).resolve().parents[2] / "backend")}
    command = [sys.executable, "-c", code, str(tmp_path)]
    with DirectoryLease(tmp_path):
        result = subprocess.run(command, capture_output=True, text=True, timeout=10, env=environment)
        assert result.returncode != 0
        assert "already open" in result.stderr
    result = subprocess.run(command, capture_output=True, text=True, timeout=10, env=environment)
    assert result.returncode == 0, result.stderr


def test_abrupt_process_exit_releases_lock_without_deleting_file(tmp_path):
    code = "import os, sys; from pathlib import Path; " \
           "from studio_app.launcher.directory_lease import DirectoryLease; " \
           "lease = DirectoryLease(Path(sys.argv[1])); lease.__enter__(); os._exit(17)"
    environment = {**os.environ, "PYTHONPATH": str(Path(__file__).resolve().parents[2] / "backend")}
    result = subprocess.run([sys.executable, "-c", code, str(tmp_path)],
                            capture_output=True, timeout=10, env=environment)
    assert result.returncode == 17
    assert (tmp_path / "studio.lock").exists()
    with DirectoryLease(tmp_path):
        pass
