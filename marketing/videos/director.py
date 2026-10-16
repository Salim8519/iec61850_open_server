"""Record the real ScadaBench app for tutorial videos.

A fresh app copy (temporary data, development build, real native workers) runs in this
process; Microsoft Edge, driven by Playwright, records the screen. A visible cursor and
click ripple are drawn into the page, and captions are timed against the recording so
compose.py can burn them in. Nothing here changes how the app behaves.
"""
from __future__ import annotations

import hashlib
import json
import re
import shutil
import subprocess
import time
import wave
from dataclasses import dataclass, field
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
STUDIO = ROOT / "studio"
OUT = Path(__file__).resolve().parent / "out"
VIEW = {"width": 1600, "height": 900}
VOICE = OUT / "voice"


def voice_id(text: str) -> str:
    return hashlib.sha1(text.encode("utf-8")).hexdigest()[:12]


def voice_seconds(text: str) -> float:
    """Length of the narration for a caption (estimated until the voice file exists)."""
    path = VOICE / f"{voice_id(text)}.wav"
    if not path.exists():
        return len(text) / 16.0
    with wave.open(str(path)) as audio:
        return audio.getnframes() / float(audio.getframerate())

CURSOR_JS = """
(() => {
  if (window.__sbCursor) return; window.__sbCursor = true;
  const install = () => {
    const css = document.createElement('style');
    css.textContent = `#sb-cursor{position:fixed;left:0;top:0;width:22px;height:22px;z-index:2147483647;
      pointer-events:none;transform:translate(-3px,-2px);transition:transform .05s linear}
      .sb-ripple{position:fixed;width:40px;height:40px;margin:-20px 0 0 -20px;border-radius:50%;
      border:3px solid #2be4b0;z-index:2147483646;pointer-events:none;animation:sbr .55s ease-out forwards}
      @keyframes sbr{from{transform:scale(.3);opacity:1}to{transform:scale(1.5);opacity:0}}`;
    document.head.appendChild(css);
    const c = document.createElement('div'); c.id = 'sb-cursor';
    c.innerHTML = '<svg viewBox="0 0 24 24" width="22" height="22"><path d="M3 2l7 19 2.6-7.4L20 11z" fill="#0b1324" stroke="#fff" stroke-width="1.6" stroke-linejoin="round"/></svg>';
    document.body.appendChild(c);
    addEventListener('mousemove', e => { c.style.left = e.clientX + 'px'; c.style.top = e.clientY + 'px'; }, true);
    addEventListener('mousedown', e => { const r = document.createElement('div'); r.className = 'sb-ripple';
      r.style.left = e.clientX + 'px'; r.style.top = e.clientY + 'px'; document.body.appendChild(r);
      setTimeout(() => r.remove(), 600); }, true);
  };
  if (document.body) install(); else addEventListener('DOMContentLoaded', install);
})();
"""


def start_app() -> tuple[subprocess.Popen, str, Path]:
    """Start a fresh ScadaBench copy (serve_copy.py) and return it, its URL and data folder."""
    python = STUDIO / ".venv/Scripts/python.exe"
    process = subprocess.Popen([str(python), str(Path(__file__).with_name("serve_copy.py"))],
                               stdout=subprocess.PIPE, text=True)
    url = process.stdout.readline().strip()
    data = Path(process.stdout.readline().strip())
    if not url.startswith("http"):
        process.kill()
        raise RuntimeError("the app copy did not start")
    return process, url, data


@dataclass
class Director:
    page: object
    started: float = field(default_factory=time.monotonic)
    captions: list[dict] = field(default_factory=list)
    clicks: list[dict] = field(default_factory=list)
    _open: dict | None = None
    _voice_until: float = 0.0

    def now(self) -> float:
        return time.monotonic() - self.started

    def say(self, text: str, hold: float = 0.0) -> None:
        """Start a caption and its narration once the previous line has been spoken.
        Actions after say() run while the voice plays, so the video never stalls."""
        self.settle()
        if self._open:
            self._open["end"] = self.now()
        self._open = {"start": self.now(), "end": None, "text": text, "voice": voice_id(text)}
        self.captions.append(self._open)
        self._voice_until = self.now() + voice_seconds(text) + 0.15
        if hold:
            self.wait(hold)

    def settle(self) -> None:
        """Wait until the current narration line has finished."""
        remaining = self._voice_until - self.now()
        if remaining > 0:
            self.wait(remaining)

    def quiet(self) -> None:
        """End the current caption (after its narration) without starting another."""
        if self._open:
            self._open["end"] = max(self.now(), self._voice_until)
            self._open = None

    def wait(self, seconds: float) -> None:
        self.page.wait_for_timeout(int(seconds * 1000))

    def point(self, locator, pause: float = 0.2) -> tuple[float, float]:
        locator.first.wait_for(state="visible", timeout=15000)
        locator.first.scroll_into_view_if_needed()
        box = locator.first.bounding_box()
        x, y = box["x"] + box["width"] / 2, box["y"] + box["height"] / 2
        self.page.mouse.move(x, y, steps=14)
        self.wait(pause)
        return x, y

    def click(self, locator, after: float = 0.45) -> None:
        x, y = self.point(locator)
        self.clicks.append({"t": round(self.now(), 3), "x": round(x), "y": round(y)})
        self.page.mouse.click(x, y)
        self.wait(after)

    def type(self, locator, text: str, after: float = 0.4) -> None:
        self.click(locator, after=0.15)
        locator.first.fill("")
        self.page.keyboard.type(text, delay=32)
        self.wait(after)

    def finish(self, name: str) -> Path:
        self.settle()
        if self._open:
            self._open["end"] = self.now()
        OUT.mkdir(parents=True, exist_ok=True)
        path = OUT / f"{name}.captions.json"
        path.write_text(json.dumps(self.captions, indent=1), encoding="utf-8")
        (OUT / f"{name}.clicks.json").write_text(json.dumps(self.clicks), encoding="utf-8")
        return path


def record(name: str, scenario, prepare=None) -> Path:
    """Run one scenario against a fresh app and keep the recording as out/<name>.webm."""
    process, url, data = start_app()
    raw = OUT / "raw" / name
    shutil.rmtree(raw, ignore_errors=True)
    try:
        return _record(name, scenario, url, data, raw, prepare)
    finally:
        process.kill()
        process.wait()
        shutil.rmtree(data.parent, ignore_errors=True)


def _record(name, scenario, url, data, raw, prepare) -> Path:
    from playwright.sync_api import sync_playwright

    with sync_playwright() as play:
        browser = play.chromium.launch(channel="msedge", headless=True)
        context = browser.new_context(viewport=VIEW, record_video_dir=str(raw), record_video_size=VIEW,
                                      device_scale_factor=1, color_scheme="light")
        context.add_init_script(CURSOR_JS)
        page = context.new_page()
        director = Director(page)
        if prepare:
            prepare(page)
        page.goto(url)
        page.get_by_text(re.compile("Engine connected|Activate ScadaBench")).first.wait_for(timeout=30000)
        page.mouse.move(VIEW["width"] / 2, VIEW["height"] / 2)
        scenario(director, page, data)
        director.settle()
        director.wait(0.8)
        director.finish(name)
        video = page.video
        context.close()
        browser.close()
        target = OUT / f"{name}.webm"
        shutil.move(video.path(), target)
    return target
