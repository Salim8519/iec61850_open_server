# 0191: GOOSE follows the IEDs' network; real devices ask once; one clear Npcap line

Date: 2026-10-26. Owner: GOOSE should use the same network as the IEDs automatically, with the usual
protection on where it goes; Npcap as one clear red line with a download link. Trigger: a PC
(IED 130) and a VMware VM (IED 1005) on 0.1.27 never exchanged GOOSE; the cause was two different
networks, and nothing on screen said so. Decision: [ADR-0119](../../decisions/ADR-0119-goose-follows-the-ieds.md).

## Change

| File | Why |
| --- | --- |
| `goose_engine/internal/engine/real.go` (new), `share.go`, `engine.go`, `stats.go`, `ops.go` | a vendor (globally administered) source MAC on the adapter holds unconfirmed sharing (`real_hold`) before any frame leaves; `uplink.confirm_real` (absent = confirmed, so the Python backend is unchanged); `real_frames` count; capability `goose_engine_v2` |
| `engine/internal/project/netset/reach.go` (new), `network.go`, `project/ied_settings.go` | Go-only `goose_local` key (stored only while true; `goose_reach` keeps Python's values so the Python engine opens every project); no lab notice; Python twin switch |
| `engine/internal/acquisition/goosedom/reach_auto.go` (new) | `Reach`, `Shares`, `Confirmed`, `Hold`, `Risk`, `Status`, `EngineCapabilities`; Python twin switch |
| `engine/internal/acquisition/goose_share.go`, `goose_share_auto.go` (new), `goose_engine.go`, `goose_monitor.go`, `manager.go` | auto Apply/Start, `goes_out` and `alone` in the preview, monitor `Waiting` |
| `frontend/src/features/network/GooseReach.tsx`, `useGoesOut.ts` (new), `goose.css`; `app/TopBar.tsx`; `copy/gooseReach.ts`, `copy/hints.ts`, `copy/status.ts`; `api/network.ts`, `api/types.ts` | new card: two choices, adapter line, protection list, Review and confirm, peer hint, red Npcap line; top-bar pill from the Go state (push, no polling) |
| `docs/specs/goose.md`, `docs/reference/API.md` | GOOSE-32..34; API shape |

Golden and parity tests that replay Python recordings now run as the Python twin (`netset` and
`goosedom` `AsPythonTwin`), as in change 0190.

## Evidence

- `studio_goose`: `go test ./...` pass, engine package 3x; new real-device tests (held before any
  frame, stops running sharing, another ScadaBench never counts).
- Go engine: goosedom and project reach tests; golden/parity suites pass as the Python twin;
  `TestFoldBeforePython` and `TestPythonArchiveRoundTrip` pass (they caught a first design that
  stored `auto` and made projects unreadable by Python; fixed with `goose_local`).
- `verify.py` (first run): file lengths, architecture, documentation, history, specifications and
  GOOSE engine PASS; Go failures were only the two fold tests (since fixed and re-run) and three
  runtime tests that cannot listen on 127.63.x while the installed ScadaBench holds the ports
  (WinError 10013), unrelated. The second full run was stopped at the owner's request.
- Frontend 485/485, `tsc` clean.
- Built app (server, `studio_goose`, frontend) on port 8891 with scratch data, 1366x768: loopback
  project "Same network as your IEDs" stays inside; Advanced VMnet8 shows "Goes out at Run",
  adapter line and protection list; "Inside ScadaBench only" works; a real IED shows "Waiting for
  you", the high-risk warning, and after confirm "Goes out at Run". Run was not pressed, so no
  GOOSE was sent to the owner's VMnet8.

## Limitations

- The wire tests (GOOSE-18/20/21/22) need Npcap and `STUDIO_GOOSE_ADAPTER`; they were not run, and
  no PC-to-VM GOOSE exchange was recorded yet (owner test pending).
- The "nobody there" hint needs 10 s of sending and a GOOSE still waiting; it cannot tell *which*
  network the other PC is on.
- A real device that only appears after confirmation is not asked about again for that adapter;
  the gocbRef conflict guard still applies.
