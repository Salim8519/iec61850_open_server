# ADR-0049: Narrated tutorial videos recorded from the real app

Accepted: 2026-10-17. Owner request: real tutorial videos instead of "Coming soon". They should
teach SLD, GOOSE, IEC 61850, CID and SCD files, simulation and real devices, at a brisk pace,
with a Gemini voice through the owner's fal.ai account, edited with Remotion.

**Decision**
- **Footage is the real app, never mock-ups.** `marketing/videos/director.py` starts a
  throwaway app copy (`serve_copy.py`): development build, temporary data, real native workers.
  Microsoft Edge, driven by Playwright, shows it at 1600×900. The page is captured through the
  DevTools screencast at 2× device pixels (`capture.py`): full-quality frames at their real
  timestamps, assembled into near-lossless 30 fps footage. Playwright's own recorder was too
  blurry. A drawn cursor and click ripple are injected into the page only.
- **Only public demo data.** Lessons load the generic demo station
  `marketing/videos/demo/scadabench_demo.scd` (`demo_station.py`). It is built from the OpenSCD
  example `scd/model_substation.scd` (IEDs FEED1, FEED2, BUS1, BUS2, TR1, TR2), keeping 50 kV
  only and adding LNode links so every switch shows its live position. Lessons can also use the
  built-in sample IED. Customer or vendor files (for example `test_cids/`) are never used in
  public material. Lessons (`lessons_basics.py`, `lessons_more.py`) click
  through the product like a user. They never press Apply in the network wizard, and no
  Windows change is made.
- **Honest demos.**
  - The real-device lesson connects to a second ScadaBench that simulates the relay, and the
    narration says so.
  - The activation lesson shows the real activation screens with demo answers (the development
    build has no licence lock), an example key, and the live offline-activation page.
- **Narration:** fal.ai `google/gemini-3.8-flash-lite-tts`, voice Charon. There is one WAV per
  caption line, named by the hash of its text (`voices.py`, `fetch_voices.py`). The recorder
  starts the next line only after the previous one has been spoken, so actions run under the
  voice and the pace follows the narration. The cost is about $0.03 per 1,000 characters, about
  $0.11 for all seven lessons.
- **Edit:** Remotion 4 (`marketing/videos/remotion`, new dependency, marketing only):
  - animated title and end cards in the website brand;
  - a camera that eases toward each logged click;
  - caption pills and the narration clips.

  Output is 1080p H.264/AAC MP4 plus a poster. `website/deploy.cmd` copies them to
  `/videos/` on scadabench.com.
- Captions are burned in and spoken, in English. Recordings, voices and renders are generated
  outputs (`marketing/videos/out/`, ignored by git) and are rebuilt with
  `python marketing/videos/make.py`.

**Compatibility and licences**
- The product is unchanged; this is marketing tooling only. User-facing "Studio" strings now say
  "ScadaBench", so the videos and the app match. Internal identifiers keep their names.
- Remotion is free for individuals and companies with up to three people. A larger company
  needs a Remotion company licence to render. Viewing the videos needs nothing.
- Playwright and Edge are already used for the icon pipeline; ffmpeg/ffprobe must be on PATH.

**Amendment 2026-10-24 (change 0188): record the released engine**
- The recordings now run the Go engine, as the product ships (ADR-0083), instead of the frozen
  Python backend. `video_app.py` builds `studio/build/video-app` from the current source: the Go
  `studio_server.exe`, the production frontend bundle (`vite build`) and the native workers,
  compiled in their own CMake folder (`studio/build/video-native`) so a developer's running app
  and its build folders are never touched. `serve_copy.py` starts that server on a free port with
  a temporary data folder and stops it like Ctrl+C, so its IED workers stop too.
- Not the packaged EXE: there is no license gate (the activation lesson still uses demo answers),
  but engine, frontend bundle, defaults and native protocol code are the shipped ones.
- Public data stays the demo station and the sample IED; lessons about typed commands and the tap
  changer also use `SampleTyped`, a ScadaBench-made test IED (`studio/tests/fixtures/typed_controls`,
  "not a manufacturer model").
- Narration stays fal.ai `google/gemini-3.8-flash-lite-tts`, voice Charon, generated through the
  fal.ai MCP (owner choice 2026-10-24: this model only).
