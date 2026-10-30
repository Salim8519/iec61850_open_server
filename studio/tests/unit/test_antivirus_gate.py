"""ADR-0130: the release gate reads antivirus reports and the lab program is built unstripped."""
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / "tools"))

import antivirus_gate  # noqa: E402
import build_labnet  # noqa: E402

FOLDER = Path(r"C:\Users\x\AppData\Local\Temp\scadabench-av-1\files")
REPORT = (
    "; Processed objects:\t3\n"
    "2026-10-31 00:31:28\tC:\\Users\\x\\AppData\\Local\\Temp\\scadabench-av-1\\files\\ScadaBench__a.exe\tok\n"
    "2026-10-31 00:31:28 \tC:\\Users\\x\\AppData\\Local\\Temp\\scadabench-av-1\\files\\"
    "ScadaBench-Setup-0.1.30.exe//data0028\tsuspicion\tVHO:Trojan.Win64.Crypt.gen\n"
    "2026-10-31 00:31:28\tC:\\Users\\x\\AppData\\Local\\Temp\\scadabench-av-1\\files\\"
    "ScadaBench-Setup-0.1.30.exe//data0028\tskipped\n"
    "2026-10-31 00:31:29\tC:\\Users\\x\\AppData\\Local\\Temp\\scadabench-av-1\\files\\b.exe\tdetected\tHEUR:Trojan.Win32.Generic\n"
)


def test_every_kaspersky_detection_is_reported_with_its_name():
    assert antivirus_gate.kaspersky_findings(REPORT, FOLDER) == [
        ("ScadaBench-Setup-0.1.30.exe//data0028", "Kaspersky suspicion: VHO:Trojan.Win64.Crypt.gen"),
        ("b.exe", "Kaspersky detected: HEUR:Trojan.Win32.Generic"),
    ]


def test_a_clean_report_has_no_findings():
    clean = "\n".join(line for line in REPORT.splitlines() if "\tok" in line or line.startswith(";"))
    assert antivirus_gate.kaspersky_findings(clean, FOLDER) == []


def test_no_scanner_fails_unless_allowed(monkeypatch):
    monkeypatch.setattr(antivirus_gate, "kaspersky", lambda: None)
    monkeypatch.setattr(antivirus_gate, "defender_active", lambda: False)
    assert antivirus_gate.check([FOLDER], allow_no_scanner=True) == []
    try:
        antivirus_gate.check([FOLDER])
    except SystemExit as stop:
        assert "No antivirus scanner found" in str(stop)
    else:
        raise AssertionError("the gate passed without a scanner")


def test_the_lab_program_keeps_its_symbols():
    command = build_labnet.command(Path("x.exe"))
    flags = command[command.index("-ldflags") + 1]
    assert flags == "-H windowsgui"
