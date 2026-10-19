# 0103: Tutorial videos re-recorded for the current app (nine lessons)

Date: 2026-10-19. Owner request: the videos on scadabench.com were outdated after many UI changes
and new features; delete them or make new ones. Recommendation accepted: replace them in place
(same addresses), no "v2" set. Decision: unchanged [ADR-0049](../../decisions/ADR-0049-tutorial-videos.md)
(same pipeline, voice and public demo data); this change only updates the lessons.

## Change

- **Lessons rewritten against the current UI** (`marketing/videos/`):
  - `lessons_basics.py`: tour with the Tools step (8 steps), data points search, roles
    (Be this IED / Read real IED / Leave out), GOOSE partner hint, SLD Operate and Test buttons;
  - `lessons_more.py`: simulation, GOOSE (partners, Signal flow, one-page "Put my IEDs on a
    network" review, Tools traffic), real devices with "Read real IED";
  - new `lessons_tools.py`: **IED Explorer and Find IEDs** (scan of "This PC", open in Explorer,
    CSWI1 Close after the confirmation) and **Be the SCADA gateway** (IEC 104 gateway, proposed
    point list, Close from the Send button, Commands tab, master/converter roles);
  - new `lessons_license.py`: activation, moved out of `lessons_more.py`;
  - `make.py`: nine lessons, `--record-only` for selector checks; `voices.py` reads all lesson files.
- **Capture fix** (`capture.py`, `director.py`): the DevTools screencast only sends a frame when
  the screen changes, so a still last screen was cut and the final narration line was lost.
  The last frame is now held until 1 s after the last narration line (measured on the capture
  clock, so no long still tail either).
- **Deploy safety** (`website/deploy.cmd`): ships the installer and portable exe named by
  `VERSION` in `src/data/site.ts`, never just the newest file in `studio/dist-protected`.
- **Website**: `tutorials.ts` lists nine lessons with new steps and measured lengths; the
  tutorials page and FAQ say "nine". Slugs of the seven existing lessons are unchanged, so every
  shared `/videos/<slug>.mp4` link keeps working; `ied-explorer` and `scada-gateway` are new.

## Evidence

- All nine lessons recorded from the dev build (`studio/frontend/dist` of 2026-10-19 02:12, native
  workers of the same night) and rendered: 95, 73, 70, 45, 67, 57, 72, 75 and 46 s (10 min in all).
  1080p masters 7-27 MB; 720p web copies 1.4-4.1 MB, H.264 + AAC.
- 46 new narration lines (fal.ai `google/gemini-3.8-flash-lite-tts`, voice Charon); every caption
  has its voice file (`voices.py` lists none missing).
- Frames inspected on contact sheets of every lesson:
  - SLD: Operate Close turns CBR_F1 to Closed ("Close done · sent 1"), Test Open puts it back;
  - IED Explorer: Find IEDs on "This PC" lists 6 simulated IEDs; Close on FEED1LD1/CSWI1 after the
    confirmation answers "selected and operated it (sbo-with-enhanced-security)", now reads on (closed);
  - gateway: "Listening on 127.0.0.1:2404", 374 proposed points, Close from the Send button listed in
    Commands as operated, observed "closed (ON)";
  - real device: Live with report block ReportControl_00101, Compare shows "Everything matches the SCD";
  - GOOSE: network review lists 172.16.1.20-41 as Free; Apply never pressed, `ipconfig` shows no
    172.16.1.x address afterwards.
- Only OpenSCD demo IEDs and the sample Feeder01 appear; no customer or vendor file.
- Published 2026-10-19: Pages deployment a29c9828; scadabench.com/tutorials says "10 minutes" and
  "Nine short narrated lessons"; all nine `/videos/*.mp4` answer 200 video/mp4 and `goose.mp4` is
  byte-identical to the local web copy (md5 6013d19d…); the download page still serves
  ScadaBench-Setup-0.1.10.exe (200, 24 530 262 bytes) and the unversioned name redirects to it.
- `deploy.cmd` now takes the installer and portable exe of the version in `site.ts`; before, it took
  the newest file and would have shipped (or stopped on) the 0.1.13 test build in dist-protected.

## Limitations

- The network card list in the GOOSE lesson shows this PC's own adapter addresses (no customer data).
- Network setup is reviewed only; "Put 6 IEDs on Ethernet" is never pressed (no Windows change).
  GOOSE pulses on the SLD are narrated, not shown, because they need the lab network.
- The real device and Explorer targets are ScadaBench copies on 127.50.0.x, as narrated.
- The narration was not listened to by a person; levels and timing were checked by tooling.
- Voice files are generated assets (ignored by git); 46 new lines cost about $0.10.
