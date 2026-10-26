"""Copy the rendered tutorial videos to out/web/ for scadabench.com.

The 1440p masters from compose.py already have the index at the front (+faststart), so they
are shipped as they are: sharp text matters more than a few megabytes. Only a master over the
Cloudflare Pages limit (25 MiB per file) is encoded again, lighter but still 1440p.
website/deploy.cmd ships out/web/ when it exists. Needs ffmpeg on PATH.
Run: python marketing/videos/web_encode.py
"""
from __future__ import annotations

import shutil
import sys
from pathlib import Path

from compose import downscale

OUT = Path(__file__).resolve().parent / "out"
WEB = OUT / "web"
SKIP_SUFFIXES = (".capture.mp4", ".2x.mp4")
PAGES_LIMIT = 25 * 1024 * 1024


def masters() -> list[Path]:
    return sorted(p for p in OUT.glob("*.mp4") if not p.name.endswith(SKIP_SUFFIXES))


def encode(source: Path, target: Path) -> None:
    shutil.copy2(source, target)
    crf = 20
    while target.stat().st_size >= PAGES_LIMIT and crf <= 30:
        downscale(source, target, crf)
        crf += 2


def main() -> int:
    if shutil.which("ffmpeg") is None:
        print("ffmpeg was not found on PATH. Install it, then run this again.")
        return 1
    sources = masters()
    if not sources:
        print(f"No videos in {OUT}. Run marketing/videos/make.py first.")
        return 1
    WEB.mkdir(exist_ok=True)
    for source in sources:
        target = WEB / source.name
        encode(source, target)
        poster = source.with_suffix(".jpg")
        if poster.exists():
            shutil.copy2(poster, WEB / poster.name)
        before, after = source.stat().st_size / 1e6, target.stat().st_size / 1e6
        print(f"{source.name}: {before:.1f} MB -> {after:.1f} MB")
    return 0


if __name__ == "__main__":
    sys.exit(main())
