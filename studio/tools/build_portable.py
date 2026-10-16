"""Build the offline Windows folder from already-built native and frontend assets."""
from pathlib import Path
import argparse
import os
import subprocess
import sys
from uuid import uuid4

STUDIO = Path(__file__).resolve().parents[1]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--fresh-work", action="store_true", help="Use a fresh intermediate folder without deleting old build files")
    parser.add_argument("--dist-dir", type=Path, default=STUDIO / "dist")
    parser.add_argument("--native-bin", type=Path, default=STUDIO / "native/build/Release")
    parser.add_argument("--frontend-dir", type=Path, default=STUDIO / "frontend/dist")
    args = parser.parse_args()
    if sys.platform != "win32":
        raise SystemExit("Build the Windows distribution on Windows x64")
    destination = args.dist_dir.resolve()
    if not destination.is_relative_to(STUDIO.parent.resolve()) or destination in {STUDIO.resolve(), STUDIO.parent.resolve()}:
        raise SystemExit("Build output must be a dedicated directory inside this workspace")
    if (destination / "ScadaBench/data").exists():
        raise SystemExit("Preview contains user data. Back it up and choose a clean build output first.")
    work = STUDIO / "build" / ("portable-" + uuid4().hex if args.fresh_work else "portable")
    command = [sys.executable, "-m", "PyInstaller", "--noconfirm",
               "--distpath", str(destination), "--workpath", str(work),
               str(STUDIO / "packaging/studio.spec")]
    environment = {**os.environ, "STUDIO_BUILD_NATIVE": str(args.native_bin.resolve()),
                   "STUDIO_BUILD_FRONTEND": str(args.frontend_dir.resolve())}
    subprocess.run(command, cwd=STUDIO.parent, check=True, env=environment)
    print(f"Preview folder: {destination / 'ScadaBench'}")


if __name__ == "__main__":
    main()
