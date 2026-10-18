"""Make the portable ScadaBench from the protected app folder (run after compile_protected.py).

    python studio/tools/make_portable.py --version 0.1.6

Two forms of the same sealed build (never the unprotected preview from build_portable.py):
- ScadaBench-<version>-Portable.zip: ScadaBench/ with an empty data/ folder beside the exe;
- ScadaBench-Portable-<version>.exe: a self-extractor (packaging/portable.iss) that unpacks the
  same folder where the user chooses. It is smaller (LZMA) and fits the website file limit.
A data folder next to the exe keeps projects in the folder (Settings.packaged_data_directory).
"""
import argparse
from pathlib import Path
import shutil
import subprocess
import tempfile
import time
import zipfile

STUDIO = Path(__file__).resolve().parents[1]
PAGES_LIMIT = 25 * 1024 * 1024  # Cloudflare Pages largest file
README = (STUDIO / "packaging" / "PORTABLE.txt").read_text(encoding="utf-8")
ISCC = Path(r"C:\Program Files (x86)\Inno Setup 6\ISCC.exe")
TRIES = 8  # antivirus scanners briefly lock the new setup file; Inno Setup does not retry by itself
FORBIDDEN = (".py", ".pyc", ".map", ".pdb")


def check(folder):
    """Refuse a folder that is not a clean protected build."""
    if not (folder / "ScadaBench.exe").is_file():
        raise SystemExit(f"No ScadaBench.exe in {folder}; run compile_protected.py first.")
    data = folder / "data"
    if data.exists() and any(data.iterdir()):
        raise SystemExit(f"{data} holds user data; a portable zip must start empty.")
    leaks = [path for path in folder.rglob("*") if path.suffix in FORBIDDEN and "studio_app" in path.as_posix()]
    if leaks:
        raise SystemExit(f"Source or symbols found, refusing to pack: {leaks[:3]}")


def make(folder, target):
    check(folder)
    with zipfile.ZipFile(target, "w", zipfile.ZIP_DEFLATED, compresslevel=9) as archive:
        for path in sorted(folder.rglob("*")):
            if path.is_file() and path.parent.name != "data":
                archive.write(path, Path("ScadaBench") / path.relative_to(folder))
        archive.writestr("ScadaBench/data/", "")
        archive.writestr("ScadaBench/PORTABLE.txt", README)
    return target.stat().st_size


def extractor(folder, dist, version):
    """Build the self-extracting portable exe, retrying while an antivirus holds the file."""
    check(folder)
    # Built outside the scanned release folder, then moved in (antivirus locks, change 0090).
    staging = Path(tempfile.mkdtemp(prefix="scadabench-portable-"))
    command = [str(ISCC), "/Q", f"/DSourceDir={folder}", f"/DOutputDir={staging}", f"/DAppVersion={version}",
               str(STUDIO / "packaging" / "portable.iss")]
    for attempt in range(1, TRIES + 1):
        if subprocess.run(command, capture_output=True).returncode == 0:
            target = dist / f"ScadaBench-Portable-{version}.exe"
            shutil.move(str(staging / target.name), str(target))
            shutil.rmtree(staging, ignore_errors=True)
            return target
        print(f"  Inno Setup try {attempt} failed (often an antivirus scan); retrying in 10 s")
        time.sleep(10)
    raise SystemExit("The portable exe could not be built; see the Inno Setup message by running ISCC by hand.")


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--version", required=True)
    parser.add_argument("--dist", type=Path, default=STUDIO / "dist-protected")
    args = parser.parse_args(argv)
    target = args.dist / f"ScadaBench-{args.version}-Portable.zip"
    size = make(args.dist / "ScadaBench", target)
    print(f"Portable zip: {target} ({size / 1024 / 1024:.1f} MiB)")
    if size > PAGES_LIMIT:
        print("  Larger than the 25 MiB Cloudflare Pages file limit: the website ships the portable exe instead.")
    if ISCC.is_file():
        exe = extractor(args.dist / "ScadaBench", args.dist, args.version)
        print(f"Portable exe: {exe} ({exe.stat().st_size / 1024 / 1024:.1f} MiB)")
    else:
        print("  Inno Setup not found: only the zip was made.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
