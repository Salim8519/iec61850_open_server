# 0106: Stopped, partly running and fully running never look alike

Date: 2026-10-19. Owner request: "how does the user know when the SLD is running, or one IED or
several are running? Currently if the user runs one IED, the SLD goes green too." Proposals were drawn
for Devices and the SLD; the owner said go ahead, without breaking earlier UI decisions.
Spec: [run one IED](../../specs/run-single-ied.md) (RUNONE-07, RUNONE-08). Builds on ADR-0067.

## Finding

With one of six IEDs running, the Run pill became the green *Stop*, the sidebar dot said "Live
diagram: IEDs are running" and the SLD badge said "Running · drawing is locked". Only the small top-bar
words "Running 1 of 6 IEDs" told the truth. Equipment of stopped IEDs said "Awaiting value", which
reads as a fault. The backend already reports a stopped IED's switch as unknown, so its wires used
the existing "Maybe live" style, never "Live"; that stays as it is. At 1366 px the page subtitle was
already cut by 46 px during any run.

## Change

- `copy/runShape.ts`: one rule for all places: ○ stopped, ◐ some IEDs, ● all IEDs (left-out IEDs
  never count), plus the words.
- Top bar: the run state shows the mark; a partial run is an amber dashed pill and a white
  *Stop all* (green only when all run). The idle "All changes saved" yields its place while IEDs run
  (the drawing is locked); "Saving..." and "Not saved" still show. Subtitle no longer cut at 1366 px.
- Sidebar: Devices and SLD dots are half filled with "1 of 6 IEDs running" during a partial run.
- Devices: one line "1 of 6 running: C03_Y1105. The other 5 are stopped." with *Run the other 5*;
  the list shows "Running · N" first and "Not running · N" after, running rows with a solid left bar.
  Row pills keep their words from change 0098 ("Publishing", "Be this IED · stopped").
- SLD: the lock badge names the live IEDs with *Run all*; equipment whose state comes only from
  stopped IEDs is dashed and dimmed with "IED stopped" and a hover sentence. Wire colours unchanged.
- Backend: `Coordinator.start()` without ids during a run adds every stopped IED that is not Off
  (it used to refuse with "Stop the current simulation first"); with nothing left it still refuses.

## Evidence

| Check | Result |
| --- | --- |
| `python studio/tools/verify.py` | all PASS |
| Frontend tests | 170 pass (new `runShape.test.ts`; existing running-dot contract kept unchanged) |
| Python tests (repo root) | 512 pass, 11 skipped, incl. new protocol test: one IED runs, `POST /runtime/start` adds the other with both answering `studio_probe`, a second call is refused |
| Dev app in browser, 1366×768, on a copy of the dev project (another session used `studio/data` and port 8765) | one IED: "◐ Running 1 of 6 IEDs", white *Stop all*, Devices line and groups "Running · 1 / Not running · 5"; SLD badge "1 of 6 IEDs live: C03_Y1105", QA1 dashed with "IED stopped" and its wire "Maybe live"; *Run all* → "● Running · 6 IEDs", green *Stop*, line and groups gone; subtitle overflow 0 px in both states; run stopped afterwards |

## Limitations

- Relay panels keep their own row values ("—" for a stopped IED); they get no "IED stopped" chip.
- Equipment with only a command link to a stopped IED is not greyed (its state comes from elsewhere).
- 1920×1080 not checked; no EXE built.
