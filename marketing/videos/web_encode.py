"""Make light web copies of the rendered tutorial videos for scadabench.com.

The 1080p masters in out/ are 7 to 20 MB each, too heavy for phones on mobile data.
This writes 720p H.264 copies with the index at the front (+faststart, so playback starts
before the download ends) to out/web/. website/deploy.cmd ships out/web/ when it exists.
Needs ffmpeg on PATH. Run: python marketing/videos/web_encode.py
"""
from __future__ import annotations

import shutil
import subprocess
import sys
from pathlib import Path

OUT = Path(__file__).resolve().parent / "out"
WEB = OUT / "web"
SKIP_SUFFIXES = (".capture.mp4",)


def masters() -> list[Path]:
    return sorted(p for p in OUT.glob("*.mp4") if not p.name.endswith(SKIP_SUFFIXES))


def encode(source: Path, target: Path) -> None:
    subprocess.run([
        "ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-i", str(source),
        "-vf", "scale=-2:720", "-c:v", "libx264", "-preset", "slow", "-crf", "28",
        "-pix_fmt", "yuv420p", "-c:a", "aac", "-b:a", "96k", "-movflags", "+faststart",
        str(target),
    ], check=True)


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
