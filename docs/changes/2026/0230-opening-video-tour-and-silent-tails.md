# 0230 Opening video redone as a live tour; silent tails removed from the tutorial videos

Date: 2026-11-20. Owner on 0229's opening video: "he talks and screen does something else
especially at goose, at end of video silence happened ... show user all app features quickly
and correctly", then "Show when goose happens ... Also gateways and ied explorer and more", and
"the test station video has silence issue". Layer: `marketing/videos`, `website/src/data/tutorials.ts`.

## Causes found

- **Voice over the wrong screen.** The 0229 script started a line, then ran the next action while
  it played, so "Switch on GOOSE" was heard over the Devices page. The GOOSE layer also showed
  nothing: the demo diagram had no IED boxes, so there was no line to draw.
- **Silence at the end.** `capture.py` built the footage with the ffmpeg concat demuxer, which held
  the repeated last frame a second time. Any lesson ending on a still screen got that hold twice
  (getting-started +14.6 s; cid-scd, goose, lab-network, sld, point-panels, simulation and
  scada-gateway 5-13 s). Fixed with `-t` = the real length.
- **Test-station gap.** 6.4 s of pointing without narration after Run.

## What changed

- `lessons_start.py`: a tour that shows each feature working, one scene per line (`scene()`: act,
  then speak over that screen, or speak while the action it names runs). Import → SLD with a box
  per GOOSE IED → Run → operate a breaker → send GOOSE from FEED1 and watch it reach BUS1/BUS2 →
  Find IEDs → IED Explorer on CSWI1.Pos → SCADA gateway listening with its point list → SCADA
  master → the remaining tools → end on the SLD.
- `voices.py` finds `scene(d, ...)` lines and lines split over adjacent string literals.
- `open_tool()` uses `nav()` (menu names gain a dot while IEDs run).
- Test-station: a narrated line covers the sequence running; no gap over 0.3 s.
- The seven affected captures were trimmed to their last line + 1 s and composed again (no
  re-recording; their pictures and voices are unchanged).

## Evidence

Trial recordings checked frame by frame at every caption; gap list (voice end to next line)
computed from captions and voice lengths; 17 new narration lines (Gemini TTS, Charon).

## Limitations

Short gaps (2-4 s) remain where the app is working (import, opening a tool); the screen is moving
there.
