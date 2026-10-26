"""Edit a recording into the published tutorial with Remotion (marketing/videos/remotion).

The real-app recording becomes footage; Remotion adds the animated title and end cards,
a camera that eases toward each click, and caption pills. Output: out/<slug>.mp4
(2560x1440 H.264, index at the front) and out/<slug>.jpg (poster for the website's <video> element).
The 1080p composition is rendered at 2x (3840x2160, from the 3200x1800 capture) and downscaled with
Lanczos, so small UI text stays sharp; out/<slug>.2x.mp4 is only an intermediate.
"""
from __future__ import annotations

import json
import shutil
import subprocess
from pathlib import Path

HERE = Path(__file__).resolve().parent
OUT = HERE / "out"
REMOTION = HERE / "remotion"
MANIFEST = REMOTION / "src" / "lessons.json"


def seconds(video: Path) -> float:
    result = subprocess.run(["ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0",
                             str(video)], capture_output=True, text=True, check=True)
    return float(result.stdout.strip())


def update_manifest(slug: str, number: int, title: str) -> None:
    footage = REMOTION / "public" / "footage" / f"{slug}.mp4"
    footage.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(OUT / f"{slug}.capture.mp4", footage)
    lesson = {"slug": slug, "number": number, "title": title, "seconds": round(seconds(footage), 2),
              "captions": json.loads((OUT / f"{slug}.captions.json").read_text(encoding="utf-8")),
              "clicks": json.loads((OUT / f"{slug}.clicks.json").read_text(encoding="utf-8"))}
    voices = REMOTION / "public" / "voice"
    voices.mkdir(parents=True, exist_ok=True)
    for caption in lesson["captions"]:
        clip = OUT / "voice" / f"{caption.get('voice')}.wav"
        if not clip.exists():
            raise FileNotFoundError(f"no voice for: {caption['text']} (run voices.py)")
        shutil.copyfile(clip, voices / clip.name)
    lessons = json.loads(MANIFEST.read_text(encoding="utf-8")) if MANIFEST.exists() else []
    lessons = [item for item in lessons if item["slug"] != slug] + [lesson]
    MANIFEST.write_text(json.dumps(sorted(lessons, key=lambda item: item["number"]), indent=1), encoding="utf-8")


def remotion(*args: str) -> None:
    subprocess.run(["npx", "remotion", *args], cwd=REMOTION, check=True, shell=True)


def downscale(source: Path, target: Path, crf: int = 18) -> None:
    """1440p H.264 for the website: sharp text, playback starts before the download ends."""
    subprocess.run(["ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-i", str(source),
                    "-vf", "scale=2560:1440:flags=lanczos", "-c:v", "libx264", "-preset", "slow", "-crf", str(crf),
                    "-pix_fmt", "yuv420p", "-c:a", "copy", "-movflags", "+faststart", str(target)], check=True)


def compose(slug: str, number: int, title: str) -> Path:
    update_manifest(slug, number, title)
    final = OUT / f"{slug}.mp4"
    sharp = OUT / f"{slug}.2x.mp4"
    remotion("render", "src/index.ts", slug, str(sharp), "--codec=h264", "--scale=2", "--crf=12",
             "--audio-codec=aac", "--audio-bitrate=192k", "--log=error")
    downscale(sharp, final)
    sharp.unlink()
    remotion("still", "src/index.ts", slug, str(OUT / f"{slug}.jpg"), "--frame=60", "--image-format=jpeg",
             "--log=error")
    return final
