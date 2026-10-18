# 0102: GOOSE between machines made visible, retransmissions within TAL

Date: 2026-10-19. Decision: [ADR-0070](../../decisions/ADR-0070-goose-visible-between-machines.md).
Spec: [signal layers](../../specs/signal-layers.md) (LAYER-20, 22, 23). Follows
[0101](0101-goose-host-vm-0.1.12.md).

The owner asked: fix the 0101 findings, but first make sure they were not made up because some
clicks were not possible; plan, check, then fix; use our own files to prove them.

## Re-check before fixing (own files only)

| Finding (0101) | How it was re-checked | Result |
| --- | --- | --- |
| 3 Burst gaps over TAL | Dev worker publishing a copy of `scd/open_substation.scd` with MinTime 4 ms / MaxTime 1000 ms (scratch file, not committed) through Npcap on VMnet8, MMS on loopback; tshark | **Real.** Gaps 4, 15.4, 15.7 ms with TAL 12 ms, 7 of 21 frames late. The unmodified sample (500 ms default) had none, which is why it was never seen |
| 1 SLD movement invisible | Dev app (scratch data folder, lab mode written into its own project file, no Windows change), `model_substation.scd` BUS1 only; FEED1 published from outside by the dev worker every 4 s; dot and label measured in the page | **Real when zoomed out:** at zoom 0.2–0.29 the dot was 2–2.9 px and off screen, the label 3.3–4 px. **Narrower than 0101 said:** at zoom ~1 a 10.6 px dot did travel ~370 px into the tag. When the owner said "I don't see GOOSE moving", nothing was changing (idle repeats draw nothing by design) |
| 2 Lab IED without lab address | Same dev project plus a sample IED added after "lab setup" | **Real:** card "127.50.0.3:102 · On the lab network adapter", role line "this PC only", top bar "2 IEDs reachable from VMnet8" |
| 4 Partner status | Same; FEED1 arriving (stNum 33), FEED2/TR1/TR2 silent | **Real:** all four "Outside project"; backend link map had FEED1 `receiving` |
| 5 Clipped labels | A node labelled `PUSH_BUTTON-2` | **Real:** scrollWidth 97 > 93 |

## Changes

- `native/worker/main.cpp`, `CMakeLists.txt`: the IED worker raises the Windows timer to 1 ms
  while it runs (winmm).
- SLD overlay (`SignalOverlay.tsx`, `PulseDot.tsx`, `signalLayout.ts`, `signal-layers.css`):
  - the outside sender's dot starts 220 px × tag scale beside the receiver;
  - the dot draws a fading line;
  - dot, line and label use the tag scale.
- `IedTag.tsx`, `signalLayerStore.ts`, `signalPulses.ts`: the receiver tag shows "← FEED1 · stNum N"
  for ~2 s.
- `features/network/endpoints.ts` `onLabNetwork` is used by `TopBar.tsx` ("1 of 2 IEDs reachable
  from …"), `DeviceOverview.tsx` ("no address on the lab network yet") and `RoleLine.tsx`.
- `copy/signalLayers.ts` `linkStatus`: a receiving outside sender shows its live state.
- `equipment-node.css`: names and state pills wrap.
- `tools/history_log.py`: reads `git diff` as UTF-8.
  - `verify.py` crashed on Windows (cp1252) once this change's ledger line contained "←".
  - It was found after the ledger line was written, so that line does not list this file.
- Tests:
  - new `tests/frontend/gooseVisibility.test.ts` (5 tests);
  - the outside-sender test in `signalLayers.test.ts` is renamed to the new start point.

## Evidence

| Check | Result |
| --- | --- |
| Burst after the fix (same scratch file, rebuilt `studio_ied`) | gaps 3.3-4.8 ms, 0 of 20 late; MaxTime repeats 999.6-1014 ms |
| Dev app, zoom 1.0, pane visible | dot 9.3 px moving ~165 px into the tag, line fading 0.89 → 0.28, tag "← FEED1 · stNum 116" |
| Dev app, zoom 0.2 | dot 6 px, label 8.7 px, both inside the canvas, ~100 px travel (before: 2 px, off screen) |
| Devices (dev app) | FEED1 "Receiving", FEED2/TR1/TR2 "Outside project"; top bar "1 of 2 IEDs reachable from “VMware Network Adapter VMnet8”"; Bay card "Only programs on this PC can reach it: it has no address on the lab network yet" |
| Clipped-text check at 1366×768 (`clip_check.js` selectors plus equipment label/state, reach chip, role line, h2, buttons) | Devices 0 of 44, SLD 0 of 34 (incl. `PUSH_BUTTON-2`, now on two lines) |
| Frontend `node --test` + `tsc --noEmit` | 161 passed; clean |
| Python unit + integration + packaging, protocol (GOOSE adapter VMnet8) | see the ledger entry |

## Test build 0.1.13

- Built from this change, before it was committed, with `compile_protected.py --version 0.1.13`.
  Not published.
- `ScadaBench-Setup-0.1.13.exe`: 24,558,570 B, SHA-256 `59dd1fb5…1f8a`.
- `ScadaBench-Portable-0.1.13.exe`: 29,413,157 B, SHA-256 `59bd35ff…7824`.
- Leak check clean, license enforcement on, one-folder `--self-test` ok.
- **Host:** upgraded in place with a silent per-user install. The owner's project loaded (9 IEDs,
  lab mode on VMnet8). C09_BCU now reads "Only programs on this PC can reach it: it has no address
  on the lab network yet".
- **VM:** not installed. A Windows TextInputHost window held the foreground and blocked the
  automation; the owner chose to commit instead.

## Limitations

- The VM still runs 0.1.12, so GOOSE between two machines on 0.1.13 is not yet retested.
- The browser pane must be visible for animations. A hidden pane froze the dot in one measurement,
  which was repeated with the pane shown.
- Not tested with a third-party GOOSE device.
