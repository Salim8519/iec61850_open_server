"""ADR-0099 release rules: the VM host helper ships, is built without the test seam, and is stopped
and unregistered by the installer and by stop_all."""
import importlib.util
from pathlib import Path
import sys

from studio.tools.stop_all import is_scadabench

STUDIO = Path(__file__).resolve().parents[2]
HELPER = "scadabench_hostkey.exe"


def load_tool(name):
    sys.path.insert(0, str(STUDIO / "tools"))
    spec = importlib.util.spec_from_file_location(f"studio_tool_{name}", STUDIO / f"tools/{name}.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def test_the_helper_is_one_of_the_programs_the_go_release_lays_out_and_leak_checks():
    steps = load_tool("go_release_steps")
    assert HELPER in steps.WORKERS
    assert steps.WORKERS.count(HELPER) == 1
    # It is not a sealed worker: the seal names the native workers the engine starts for ScadaBench itself.
    from protected_source import SEALED
    assert HELPER not in SEALED


def test_the_helper_is_built_windowsgui_trimmed_and_never_with_the_test_seam():
    build = load_tool("build_hostkey_helper")
    command = build.command(Path("out") / HELPER)
    assert command[1:3] == ["build", "-trimpath"]
    assert "-tags" not in command and not any("hostkeytest" in part for part in command)
    assert "-H windowsgui -s -w" in command
    assert command[-1] == "./cmd/scadabench_hostkey"
    assert build.OUTPUT.name == HELPER and build.OUTPUT.parent.name == "Release"
    # The seam file exists only for the hostkeytest tag, so a build without tags cannot contain it.
    seam = (build.ENGINE / "cmd/scadabench_hostkey/seam_hostkeytest.go").read_text(encoding="utf-8")
    assert seam.splitlines()[0].strip() == "//go:build hostkeytest"


def test_staleness_follows_the_helper_sources_but_ignores_tests(tmp_path, monkeypatch):
    build = load_tool("build_hostkey_helper")
    names = {path.name for path in build.sources()}
    assert {"main.go", "helper.go", "protocol.go", "go.mod"} <= names
    assert not any(name.endswith("_test.go") for name in names)
    missing = tmp_path / HELPER
    assert build.stale(missing)


def test_the_go_release_builds_the_helper_before_the_engine_and_the_layout(capsys):
    release = load_tool("compile_go_release")
    assert callable(release.build_hostkey)
    text = (STUDIO / "tools/compile_go_release.py").read_text(encoding="utf-8")
    main = text.split("def main(", 1)[1]
    assert main.index("build_snmp(mode)") < main.index("build_hostkey(mode)") < main.index("steps.build_engine(")


def test_the_dev_launcher_keeps_the_helper_as_fresh_as_the_source():
    text = (STUDIO / "tools/run_dev_engine.py").read_text(encoding="utf-8")
    assert '"build_hostkey_helper.py"), "--if-stale"' in text


def test_the_installer_ends_the_helper_and_removes_its_run_value():
    text = (STUDIO / "packaging/installer.iss").read_text(encoding="utf-8")
    uninstall = text.split("[UninstallRun]", 1)[1].split("[Code]", 1)[0]
    assert "/IM scadabench_hostkey.exe" in uninstall
    assert "Run /v ScadaBenchVMHost /f" in uninstall
    assert uninstall.count("RunOnceId") == 2
    code = text.split("[Code]", 1)[1]
    assert "PrepareToInstall" in code and "scadabench_hostkey.exe" in code
    # The value the installer removes is the one the app writes.
    doc = (STUDIO / "engine/internal/licensing/hostkey/doc.go").read_text(encoding="utf-8")
    assert 'RunValueName = "ScadaBenchVMHost"' in doc
    assert r"Software\Microsoft\Windows\CurrentVersion\Run" in doc


def test_stop_all_knows_the_helper():
    row = {"ProcessId": 7, "ParentProcessId": 1, "Name": "ScadaBench_HostKey.EXE", "CommandLine": ""}
    assert is_scadabench(row)
