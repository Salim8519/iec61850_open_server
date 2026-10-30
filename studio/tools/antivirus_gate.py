"""Release gate: scan every built program with the PC's antivirus before anything is published (ADR-0130).

    python studio/tools/antivirus_gate.py <file or folder> [...]

Kaspersky quarantined the 0.1.30 installer for one heuristic verdict on scadabench_labnet.exe. A release
must never reach a customer like that, so compile_go_release.py runs this gate on the installed folder,
the portable exe and the installer. The files are copied to a fresh folder in %TEMP% first: the project
folder may be excluded from scanning on the build PC. The scan only reports (Kaspersky /i0); a file
that disappears from the copy was removed by real-time protection and counts as a detection.
Kaspersky (avp.com) and Microsoft Defender (MpCmdRun.exe, only while it is the active antivirus) are
used when present; with neither the gate fails unless --allow-no-scanner is given.
"""
from pathlib import Path
import argparse
import glob
import os
import re
import shutil
import subprocess
import sys
import tempfile

KASPERSKY_GLOBS = (r"C:\Program Files (x86)\Kaspersky Lab\*\avp.com", r"C:\Program Files\Kaspersky Lab\*\avp.com")
DEFENDER = Path(os.environ.get("ProgramFiles", r"C:\Program Files")) / "Windows Defender/MpCmdRun.exe"
# Kaspersky report lines: "<date> <time>\t<path>[//<inner object>]\t<verdict>\t<detection name>".
KASPERSKY_FINDING = re.compile(r"\t(?P<path>[^\t]+)\t(?P<verdict>detected|suspicion|infected)\t(?P<name>[^\t]+)$")


def kaspersky():
    found = sorted(path for pattern in KASPERSKY_GLOBS for path in glob.glob(pattern))
    return found[-1] if found else None


def defender_active():
    if not DEFENDER.is_file():
        return False
    command = ["powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "(Get-MpComputerStatus).AMRunningMode"]
    completed = subprocess.run(command, capture_output=True, text=True, timeout=60)
    return completed.stdout.strip() == "Normal"


def kaspersky_findings(report_text, folder):
    """(file name, verdict) for each detection in a Kaspersky report, with the copy folder left out."""
    findings = []
    for line in report_text.splitlines():
        match = KASPERSKY_FINDING.search(line.rstrip())
        if match:
            target = match["path"].replace(str(folder) + "\\", "")
            findings.append((target, f"Kaspersky {match['verdict']}: {match['name']}"))
    return findings


def scan_kaspersky(avp, folder):
    # avp.com writes no report when the report file name holds a hyphen.
    report = folder.parent / "report.txt"
    subprocess.run([avp, "SCAN", str(folder), "/i0", "/fa", f"/R:{report}"], capture_output=True, timeout=1800)
    if not report.is_file():
        raise SystemExit(f"Kaspersky wrote no scan report ({avp}); the release is not checked.")
    return kaspersky_findings(report.read_text(encoding="utf-8", errors="replace"), folder)


def scan_defender(folder):
    completed = subprocess.run([str(DEFENDER), "-Scan", "-ScanType", "3", "-File", str(folder), "-DisableRemediation"],
                               capture_output=True, text=True, timeout=1800)
    if completed.returncode == 0:
        return []
    return [(str(folder.name), "Microsoft Defender: " + (completed.stdout.strip() or f"exit {completed.returncode}"))]


def copy_targets(paths, folder):
    copies = []
    for path in paths:
        files = sorted(item for item in path.rglob("*") if item.is_file()) if path.is_dir() else [path]
        for item in files:
            if item.suffix.lower() in (".exe", ".dll"):
                name = item.relative_to(path).as_posix().replace("/", "__") if path.is_dir() else item.name
                copy = folder / (path.name + "__" + name if path.is_dir() else name)
                shutil.copyfile(item, copy)
                copies.append(copy)
    return copies


def check(paths, allow_no_scanner=False):
    """Scan copies of the programs in paths; return the findings (empty when every scanner agrees)."""
    avp, defender = kaspersky(), defender_active()
    if not avp and not defender:
        if allow_no_scanner:
            print("  antivirus check skipped: no Kaspersky or active Microsoft Defender on this PC.")
            return []
        raise SystemExit("No antivirus scanner found (Kaspersky avp.com or active Microsoft Defender); "
                         "the release is not checked. Pass --allow-no-scanner to build anyway.")
    work = Path(tempfile.mkdtemp(prefix="scadabench-av-"))
    folder = work / "files"
    folder.mkdir()
    try:
        copies = copy_targets([Path(path) for path in paths], folder)
        findings = scan_kaspersky(avp, folder) if avp else []
        findings += scan_defender(folder) if defender else []
        findings += [(copy.name, "removed by real-time protection") for copy in copies if not copy.exists()]
        scanners = " and ".join(name for name, used in (("Kaspersky", avp), ("Microsoft Defender", defender)) if used)
        print(f"  antivirus check: {len(copies)} programs scanned with {scanners}, {len(findings)} findings.")
        return findings
    finally:
        shutil.rmtree(work, ignore_errors=True)


def gate(paths, allow_no_scanner=False):
    findings = check(paths, allow_no_scanner)
    if findings:
        lines = "\n".join(f"  {target}: {verdict}" for target, verdict in findings)
        raise SystemExit(f"The antivirus reports release files; do not publish this build:\n{lines}")


def main(arguments):
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("paths", nargs="+")
    parser.add_argument("--allow-no-scanner", action="store_true")
    options = parser.parse_args(arguments)
    gate(options.paths, options.allow_no_scanner)
    print("No antivirus findings.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
