"""List the narration lines of every lesson and which voice files are still missing.

  python marketing/videos/voices.py            print missing lines as JSON (id + text)
  python marketing/videos/voices.py URL ID     download a generated WAV as out/voice/<ID>.wav

Voices are generated with fal.ai "google/gemini-3.8-flash-lite-tts" (voice Charon,
brisk tutorial delivery). A file is named after the hash of its exact text, so editing
a line in a lesson simply asks for a new voice file.
"""
import json
import re
import sys
import urllib.request
from pathlib import Path

from director import VOICE, voice_id

HERE = Path(__file__).resolve().parent
SAY = re.compile(r'(?:say\(|^\s*)"([^"\n]{8,})"', re.MULTILINE)
VOICE_NAME = "Charon"
STYLE = "Confident, friendly product tutorial narrator. Brisk, clear pace, like a short Apple demo."


def lines() -> list[str]:
    found: list[str] = []
    for name in ("lessons_basics.py", "lessons_more.py", "lessons_tools.py", "lessons_license.py"):
        source = (HERE / name).read_text(encoding="utf-8")
        for match in re.finditer(r'd\.say\("([^"]+)"', source):
            found.append(match.group(1))
        loop = re.search(r'for text in \[(.*?)\]:', source, re.DOTALL)
        if loop:
            found.extend(re.findall(r'"([^"]+)"', loop.group(1)))
    return list(dict.fromkeys(found))


def missing() -> list[dict]:
    return [{"id": voice_id(text), "text": text} for text in lines() if not (VOICE / f"{voice_id(text)}.wav").exists()]


if __name__ == "__main__":
    VOICE.mkdir(parents=True, exist_ok=True)
    if len(sys.argv) == 3:
        urllib.request.urlretrieve(sys.argv[1], VOICE / f"{sys.argv[2]}.wav")
    else:
        print(json.dumps(missing(), indent=1))
