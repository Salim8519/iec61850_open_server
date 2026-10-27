# 0194: GOOSE stays inside until the user shares it

Date: 2026-10-27. Owner: GOOSE went out on VMware VMnet8 with no question in 0.1.28; "can we do it as
before?"; "inside by default, Also share needs a confirm; keep the Npcap line and the nobody-there
hint". Decision: [ADR-0122](../../decisions/ADR-0122-goose-inside-until-shared.md).

## Change

| File | Why |
| --- | --- |
| `engine/internal/acquisition/goosedom/reach_auto.go` | `Reach` is the stored `goose_reach`; `Shares` only when `outside` and an adapter is there; "inside" words no longer say the IEDs use this PC only |
| `engine/internal/acquisition/goose_share.go`, `goose_share_auto.go` | one Apply for both engines: `outside` needs `confirm:true`, Npcap and the adapter shown; the 0191 `applyAuto` is removed, the "nobody there" clock stays |
| `engine/internal/project/ied_settings.go`, `netset/network.go`, `netset/reach.go` | `SaveReach` stores the choice as Python does; `goose_local` is no longer kept; the ADR-0073 lab notice value is computed again |
| `frontend/src/features/network/GooseReach.tsx`, `goose.css`; `copy/gooseReach.ts`, `copy/hints.ts`, `copy/status.ts`; `api/network.ts`, `api/types.ts` | the ADR-0073 card: "Inside ScadaBench only" (default) and "Also share with the network" behind the warning; adapter line; Advanced locked while sharing; kept: protection list while shared, Npcap line, "nobody there" hint |
| tests: `goosedom/reach_auto_test.go`, `project/goose_reach_test.go`, `tests/frontend/gooseReach.test.ts`; `docs/specs/goose.md` (GOOSE-32..34), `docs/reference/API.md` | the restored rule |

## Evidence

- `go test` for `internal/project/...`, `internal/acquisition/...`, `internal/api/...`, `internal/app/...`
  with the venv Python: pass (the `goosedom` test program was built to a fixed path and run, because
  Windows Defender blocked it in the Go temp folder). `go vet` clean.
- Frontend 488/488; types clean.
- Dev app (Go engine, demo station on the ScadaBench Lab network): `/goose/share` answers `inside`,
  `goes_out: false`, "GOOSE stays inside ScadaBench. Nothing is sent to the network."; the card shows
  the two choices with "Inside ScadaBench only" selected; "Also share with the network" opens
  "Share GOOSE on “ScadaBench Lab”?" and Cancel leaves it inside; Share GOOSE gives `outside`,
  `goes_out: true` and the "Shares at Run" pill; "Inside ScadaBench only" puts it back in one click.

## Limitations

- A lab-network test from 0.1.28 that relied on automatic GOOSE now needs one "Share GOOSE".
