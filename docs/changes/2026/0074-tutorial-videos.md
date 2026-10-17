# 0074: Narrated tutorial videos from the real app, on public demo data only

Date: 2026-10-17. Owner requests:
- real tutorial videos with no "Coming soon";
- teach SLD, GOOSE, IEC 61850, CID and SCD files, simulation and real devices;
- a brisk pace, with the Gemini 3.8 Flash Lite TTS voice through fal.ai and Remotion editing;
- better picture quality;
- never show private customer or vendor files.

Decision: [ADR-0049](../../decisions/ADR-0049-tutorial-videos.md).

## Change

- **`marketing/videos/`:**
  - `director.py` runs a throwaway app copy (`serve_copy.py`) and paces lessons on the voice;
  - `capture.py` captures the page through the DevTools screencast at 2× (3200×1800);
  - `lessons_basics.py` and `lessons_more.py` hold seven lessons;
  - `voices.py` and `fetch_voices.py` handle the narration (fal.ai `google/gemini-3.8-flash-lite-tts`,
    voice Charon, one WAV per caption);
  - `compose.py` plus `remotion/` produce the cards, click-following camera, caption pills,
    narration, 1080p H.264/AAC and posters;
  - `make.py` records and renders everything in one command.
- **Seven lessons:**
  1. IEC 61850 and ScadaBench (data model, MMS on port 102);
  2. CID and SCD files;
  3. the SLD;
  4. simulation and test values;
  5. GOOSE and the lab network (review only, nothing applied);
  6. real devices (a second ScadaBench plays the relay, as narrated);
  7. activation (real screens with demo answers).
- **Public demo data only:** `demo_station.py` builds `demo/scadabench_demo.scd` from the OpenSCD
  example (FEED1, FEED2, BUS1, BUS2, TR1, TR2; 50 kV only), with LNode links so every switch
  is live. An earlier take used a customer/vendor SCD from `test_cids/`; all of its recordings,
  renders and screenshots were deleted and nothing from it was published.
- **Website:** the Tutorials page has seven `<video>` players with posters and teaching steps.
  `deploy.cmd` copies `marketing/videos/out/*.mp4|jpg` to `/videos/`.
- User-facing "Studio" text in the frontend and backend now reads "ScadaBench"; internal
  identifiers and the "Substation Studio" firewall rule names are unchanged.

## Evidence

- All seven lessons rendered: 86, 48, 51, 44, 58, 47 and 45 s. Each file is under 21 MB (Pages
  limit 25 MiB) and has H.264 video plus AAC narration.
- Frames inspected:
  - data-model text is sharp at 1080p;
  - the SLD shows live "Open" on every breaker and disconnector;
  - the GOOSE table shows FEED1 → BUS1, BUS2 with APPID and MAC;
  - the network review step lists the 172.16.1.x changes and Apply is never pressed;
    `ipconfig` shows no 172.16.1.x address afterwards;
  - the real device reaches Live with a free report block, Compare shows 457 equal points, and
    Health check gives one verdict.
- The frames show no customer or vendor names (only the OpenSCD demo IEDs and the sample
  Feeder01).
- Python 333 passed after the rename (one assertion updated); frontend 98 passed; `tsc` clean;
  `verify.py` pass.

## Limitations

- Published 2026-10-17 after owner review: scadabench.com/tutorials shows 7 video players and no "Coming soon"; every /videos/*.mp4 answers 200 video/mp4.
- The owner's internet line blocked some Cloudflare hostnames by name for a while (TLS cut after the server name was sent, Kaspersky off). `website/set-cloudflare-token.cmd` lets uploads use only api.cloudflare.com if it happens again.
- The narration was not listened to by a person; levels and timing were checked by tooling.
- The voice files are generated assets (ignored by git); regenerating them costs about $0.11.
