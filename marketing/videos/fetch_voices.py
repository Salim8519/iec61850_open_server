"""Save fal.ai TTS results (a saved wait_for_jobs JSON) as out/voice/<id>.wav.

  python marketing/videos/fetch_voices.py RESULTS.json [MISSING.json]
The job index in the results is the line's position in MISSING.json (voices.py output).
"""
import json
import sys
import urllib.request
from pathlib import Path

from director import VOICE, voice_id

results = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
missing = json.loads(Path(sys.argv[2] if len(sys.argv) > 2 else VOICE / "missing.json").read_text(encoding="utf-8"))
saved, failed = 0, []
for job in results.get("jobs", []):
    url = ((job.get("result") or {}).get("audio") or {}).get("url")
    if job.get("status") != "completed" or not url:
        failed.append(job.get("index"))
        continue
    text = missing[job["index"]]["text"]
    urllib.request.urlretrieve(url, VOICE / f"{voice_id(text)}.wav")
    saved += 1
print(f"saved {saved}, not ready: {failed}")
