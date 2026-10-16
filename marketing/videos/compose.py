"""Edit a recording into the published tutorial with Remotion (marketing/videos/remotion).

The real-app recording becomes footage; Remotion adds the animated title and end cards,
a camera that eases toward each click, and caption pills. Output: out/<slug>.mp4
(1920x1080 H.264) and out/<slug>.jpg (poster for the website's <video> element).
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
    footage = REMOTION / "public" / "footage" / f"{slug}.webm"
    footage.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(OUT / f"{slug}.webm", footage)
    lesson = {"slug": slug, "number": number, "title": title, "seconds": round(seconds(footage), 2),
              "captions": json.loads((OUT / f"{slug}.captions.json").read_text(encoding="utf-8")),
              "clicks": json.loads((OUT / f"{slug}.clicks.json").read_text(encoding="utf-8"))}
    lessons = json.loads(MANIFEST.read_text(encoding="utf-8")) if MANIFEST.exists() else []
    lessons = [item for item in lessons if item["slug"] != slug] + [lesson]
    MANIFEST.write_text(json.dumps(sorted(lessons, key=lambda item: item["number"]), indent=1), encoding="utf-8")


def remotion(*args: str) -> None:
    subprocess.run(["npx", "remotion", *args], cwd=REMOTION, check=True, shell=True)


def compose(slug: str, number: int, title: str) -> Path:
    update_manifest(slug, number, title)
    final = OUT / f"{slug}.mp4"
    remotion("render", "src/index.ts", slug, str(final), "--codec=h264", "--crf=22", "--log=error")
    remotion("still", "src/index.ts", slug, str(OUT / f"{slug}.jpg"), "--frame=60", "--image-format=jpeg",
             "--scale=0.6667", "--log=error")
    return final
