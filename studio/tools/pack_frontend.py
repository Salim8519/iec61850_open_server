"""Pack frontend/dist into one masked frontend.pak for protected builds.

    python studio/tools/pack_frontend.py [--dist studio/frontend/dist] [--out studio/build/protected/frontend.pak]
Source maps are refused so the shipped UI never contains original TypeScript.
"""
import argparse
from pathlib import Path
import sys

STUDIO = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(STUDIO / "backend"))
from studio_app.launcher.frontend_assets import pack, unpack  # noqa: E402


def collect(dist):
    files = {}
    for path in sorted(dist.rglob("*")):
        if not path.is_file():
            continue
        name = path.relative_to(dist).as_posix()
        if name.endswith(".map"):
            raise SystemExit(f"Source map found ({name}). Build the frontend without sourcemaps.")
        files[name] = path.read_bytes()
    if "index.html" not in files:
        raise SystemExit(f"{dist} has no index.html; run `npm run build` in studio/frontend first.")
    return files


def main(argv=None):
    parser = argparse.ArgumentParser()
    parser.add_argument("--dist", type=Path, default=STUDIO / "frontend/dist")
    parser.add_argument("--out", type=Path, default=STUDIO / "build/protected/frontend.pak")
    args = parser.parse_args(argv)
    files = collect(args.dist)
    blob = pack(files)
    if unpack(blob) != files:
        raise SystemExit("Pack round-trip failed")
    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_bytes(blob)
    print(f"frontend.pak: {len(files)} files, {len(blob) // 1024} KiB -> {args.out}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
