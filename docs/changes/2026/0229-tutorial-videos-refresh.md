# 0229 Tutorial videos: a result-first opening video, commands pop-up, Lab address in Devices, new Test lesson

Date: 2026-11-20. Owner: "Getting started video is not motivating i don't like, anyhow check
what's missing or outdated, don't do video for small feature like save project", then "for videos
u do manually". Layer: `marketing/videos` and `website/src/data/tutorials.ts`; no app code.

## What changed

- **getting-started** ("A whole substation on one PC", `lessons_start.py`): replaces the menu tour.
  The demo SCD becomes six running IEDs, the SLD draws itself, a breaker is operated, GOOSE is
  shown, and the Devices addresses close it. 78 s.
- **real-devices**: Allow commands per IED, the Send commands pop-up (confirm, device answer),
  then Compare with SCD and Health check. 77 s.
- **lab-network**: after Ping, Devices shows the same address under "Clients connect to" and says
  it can be changed there too (0.1.33). 86 s.
- **test-station** (new, `lessons_test.py`): a Smooth wave generator, a timed sequence that opens
  the breaker after 5 s, Start automatically with Run, the live value. 59 s.
- `lessons_basics.nav()`: main-menu buttons are found by the start of their name (a green dot
  joins the name while IEDs run).
- Website: titles, steps and measured lengths in `tutorials.ts`; docs Generators and Sequences
  pages show the new video. The phone menu and footer get "Offline activation" (header unchanged:
  no room); "Docs" was added by 0228.

## Decided against

- A "station check" video (Health check and Signal flow): on the public demo station every IED is
  simulated, so Health check only shows "Not checked". Health check stays in real-devices, Signal
  flow in goose.
- Videos for small actions (save, undo), as the owner asked.

## Evidence

- Trial recordings found and fixed: the installed app still held 127.50.0.x:102 (stopped it),
  nav buttons renamed while running, wrapped form labels, number boxes keeping a leading 0.
- Final `make.py getting-started real-devices lab-network test-station`: 4 recorded and composed,
  no failures; contact sheets checked frame by frame; 24 narration lines with
  google/gemini-3.8-flash-lite-tts (Charon); `web_encode.py` keeps every file under 25 MiB.
- Public material only: OpenSCD-based demo station and the built-in sample IED.

## Limitations

- Other lessons (commands, goose, sld, ...) were not re-recorded; their UI still matches.
- Not deployed: videos and site go live with the next `website/deploy.cmd`.
