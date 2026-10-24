# 0188: Tutorial videos re-recorded on the Go engine (thirteen lessons)

Date: 2026-10-24. Owner request: the tutorial videos are outdated; remake them for the current app
and add lessons for the new features. Decision: [ADR-0049](../../decisions/ADR-0049-tutorial-videos.md)
with its 2026-10-24 amendment (record the released Go engine). App fix found on the way:
[0187](0187-sld-fit-view-clear-of-toolbar.md).

## Change

- `video_app.py` (new) builds a private copy of the app in `studio/build/video-app`: the Go
  `studio_server.exe`, the production frontend and native workers built from master in their own
  CMake folder. The owner's running dev app and `studio/native/build` are never touched.
- `serve_copy.py` starts that copy with `--no-browser` on a free port with a throw-away data
  folder and stops it with Ctrl+Break; `director.py` and the real-devices lesson use it.
- Lessons 1-8 and 13 follow the current app: search placeholders, the role names *Simulate /
  Monitor real device / Skip*, the ST/MX type badge, Settings › Display *Fit to window*, one-press
  Select before Operate on the SLD, IED Explorer's *Select… → Operate* and *More › Export CSV*,
  the gateway's *Add points* dialog (Paste from Excel), type filter and *Commands* tab, the SCADA
  master point list, and the license page (VM box, one key per computer, no key moves).
- New lessons (`lessons_sld.py`, `lessons_tools.py`):
  - 9 *Send GOOSE from the SLD* (ADR-0112, change 0183): Add all GOOSE IEDs, the GOOSE
    messages layer, sending Closed/Open from the sheet and the "IEDs got it" chips.
  - 10 *Point panels and the smart point picker* (ADR-0102, ADR-0113, change 0184).
  - 11 *Select before operate, typed commands and tap changers* (ADR-0108, ADR-0109): uses the
    ScadaBench-made `studio/tests/fixtures/typed_controls/SampleTyped.scd` next to the demo station.
  - 12 *SNMP devices: switches and clocks*.
- Narration: Gemini 3.8 Flash Lite TTS (voice Charon) through the fal.ai MCP, files named by
  `voice_id(text)` in `out/voice/`; `voices.py` reports no missing line.
- Website: `tutorials.astro` / `faqs.ts` say thirteen; `tutorials.ts` has the four new entries,
  updated steps and the measured lengths.

## Videos

Masters in `marketing/videos/out/<slug>.mp4` (2560×1440), web copies in `out/web/` and on
scadabench.com at `/videos/<slug>.mp4`. Total 17:17.

| Slug | Lesson | Length |
|---|---|---|
| getting-started | IEC 61850 and ScadaBench in one minute | 1:51 |
| cid-scd | CID and SCD files | 1:22 |
| sld | The single-line diagram (SLD) | 1:25 |
| simulation | Simulate IEDs and change values | 0:54 |
| goose | GOOSE and the lab network | 1:19 |
| real-devices | Connect to real devices | 0:59 |
| ied-explorer | IED Explorer and Find IEDs | 1:20 |
| scada-gateway | Be the SCADA gateway | 1:31 |
| goose-sld | Send GOOSE from the SLD (new) | 1:29 |
| point-panels | Point panels and the smart point picker (new) | 1:42 |
| commands | Select before operate, typed commands and tap changers (new) | 1:38 |
| snmp | SNMP devices: switches and clocks (new) | 0:51 |
| activation | Activate or go offline | 0:56 |

## Evidence

- Every lesson passed a fast dry run on the video app (scratchpad `fast.py`, all 13 PASS) and
  the full `make.py` render; contact sheets of every final video were inspected.
- Only the demo station, the sample IED and SampleTyped appear; no `test_cids/` or
  `vendor_cids/` file was opened.

## Limitations

- Narration was checked by text and length, not listened to by a person.
- The SampleTyped import's single IED checkbox did not tick with a mouse click in headless Edge;
  the lesson ticks it with Playwright `check()` (not investigated as an app bug).
- The real-devices health check ends on "Not checked" for FEED2 (the second app copy that plays
  the relay is still connecting at that moment); the narration only says it gives one verdict.
- The getting-started title still says "in one minute"; the lesson is 1:51.
- The goose lesson shows this PC's adapter name and link-local/LAN addresses in the network-card
  list, as the previous videos did.
- The GOOSE "got it" chip shows for about two seconds, so the lesson clicks during the narration.
