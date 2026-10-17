"""High-quality screen capture for the tutorial videos.

Playwright's built-in recorder writes low-bitrate VP8, which blurs small UI text. Instead the
page is captured through the Chrome DevTools screencast: a full-quality JPEG frame each time
the screen changes, at 2x device pixels (3200x1800 for a 1600x900 view). The frames are
assembled at their real timestamps into a near-lossless 30 fps H.264 file, so Remotion can
zoom in and still render sharp 1080p.
"""
from __future__ import annotations

import base64
import subprocess
from pathlib import Path


class Screencast:
    def __init__(self, page, folder: Path, width: int, height: int):
        self.page, self.folder, self.size = page, folder, (width, height)
        self.frames: list[tuple[Path, float]] = []
        self.session = None

    def start(self) -> None:
        self.folder.mkdir(parents=True, exist_ok=True)
        self.session = self.page.context.new_cdp_session(self.page)
        self.session.on("Page.screencastFrame", self._frame)
        self.session.send("Page.startScreencast", {"format": "jpeg", "quality": 95, "everyNthFrame": 1,
                                                    "maxWidth": self.size[0], "maxHeight": self.size[1]})

    def _frame(self, event: dict) -> None:
        path = self.folder / f"{len(self.frames):06d}.jpg"
        path.write_bytes(base64.b64decode(event["data"]))
        self.frames.append((path, float(event["metadata"]["timestamp"])))
        try:
            self.session.send("Page.screencastFrameAck", {"sessionId": event["sessionId"]})
        except Exception:
            pass  # the page is closing

    def stop(self) -> None:
        try:
            self.session.send("Page.stopScreencast")
        except Exception:
            pass

    @property
    def first_timestamp(self) -> float:
        return self.frames[0][1]

    def assemble(self, target: Path, tail: float = 0.5) -> Path:
        """Write the frames with their real durations, then resample to 30 fps."""
        if len(self.frames) < 2:
            raise RuntimeError("the screencast captured no frames")
        lines = ["ffconcat version 1.0"]
        for (path, stamp), (_, following) in zip(self.frames, self.frames[1:] + [(None, self.frames[-1][1] + tail)]):
            lines += [f"file '{path.name}'", f"duration {max(following - stamp, 0.001):.4f}"]
        lines.append(f"file '{self.frames[-1][0].name}'")
        listing = self.folder / "frames.ffconcat"
        listing.write_text("\n".join(lines) + "\n", encoding="utf-8")
        subprocess.run(["ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-f", "concat", "-safe", "0",
                        "-i", listing.name, "-vf", f"fps=30,scale={self.size[0]}:{self.size[1]}:flags=lanczos,"
                        "format=yuv420p", "-c:v", "libx264", "-preset", "medium", "-crf", "12",
                        "-movflags", "+faststart", str(target.resolve())],
                       check=True, cwd=self.folder)
        return target
