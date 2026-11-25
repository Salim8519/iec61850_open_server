# 0239 Devices commands to a real IED: every class, one click or step by step, search, sent in place

Date: 2026-11-23. Owner, after monitoring two live GE bay controllers (E08_BCU 172.26.0.38, E10_BCU
172.26.0.40) with 0.1.36: "add select before operate, or operate directly thing", "when user sends
command in the pop up command dont change his scrolled position", "add search command", "allow
sending command direct without need to open command", "make sure the scadabench can handle many kinds
of commands in ied mode". Decision [ADR-0149](../../decisions/ADR-0149-devices-command-steps-every-class.md).
Layers: native `studio_control` (typed session), Go engine (`realcmd`, `liveapi`, `typedcmd`,
`tools/control`, `app`), frontend (`features/devices`, shared point inputs), docs.

## What the live relays showed (read only, before the change)

- E08_BCU refused all four GGIO2.SPCSO commands with `object-value-invalid`: its `LLN0.Beh`,
  `GGIO2.Beh` and the System nodes read 3 (test) and 0.1.36 always sent Test = false. After the owner
  took the relay out of Test mode, single commands worked.
- E10_BCU accepted SPC; every DPC (DiscCSWI2-5, Disc0XSWI2-4) was refused at Select with AddCause
  10, and `DiscCILO1..8` EnaOpn/EnaCls and `BkrCILO1.EnaCls` all read false: the relay's own
  interlocking. `Disc0XSWI8` (Beh 5, off) answered blocked-by-mode. ScadaBench did send Select
  (ctlModel 4); the device answered it. The page showed only the raw words.

## Change

- `POST /live/{key}/command` (select, operate, cancel) for every CID command class through the typed
  `studio_control` session; one click selects and operates, step by step keeps the selection.
- Refusals in words (`realcmd.Explain`): AddCause and MMS error meanings, and the node's Beh read on a
  refusal (Test mode, test command to a normal relay, Off). The device's answer follows.
- Typed session binds the chosen card (`live.local`) per (card, device); capability `typed_local`.
- Commands pop-up: search, all classes (ON/OFF, Open/Close, number box, list, Lower/Raise), the shared
  Command options with the app-wide step-by-step switch; each row asks and answers in place, so the
  list keeps its scroll position. Data points: the `.Oper.ctlVal` row of a command sends in place.
- Defaults keep 0.1.36's wire content (interlock check on, orCat 2); Test follows the relay's Test mode.
- The select-before-operate outcome says "Not done:" instead of "Not selected:" (an Operate refusal
  is not a selection failure); this also applies to the SLD and point panels.

## Evidence

- Go: `realcmd` unit tests (select fields, card, window 27 s, Operate needs Select only step by step,
  ISC Raise from the read position, ENC label, five refusal explanations, step refusals) and wire
  tests over `studio_ied` + `studio_control` on 127.67.0.47: INC, ENC, APC, BSC, ISC, BAC in all four
  control models one click; SBO select/operate/cancel through local card 127.0.0.1; interlock block in
  words. `runtime`, `liveapi`, `tools/control`, `typedcmd` pass.
- Frontend: `tsc` clean; `tests/frontend/commandSearch.test.ts`.
- Browser (dev build, two loopback copies: one simulating E10_BCU from the site CID, one monitoring it
  as real): pop-up lists 296 commands (SPC 210, DPC 84, ENC 1, INC 1); search "CSWI3" gives 6; send at
  scroll 1500 keeps 1500 with the answer in the row; step by step DiscCSWI3.Pos Select → "operate
  within 26 s" → Operate → "selected and operated it (sbo-enhanced) … closed"; Select → Cancel;
  INC with a simulated interlock block → interlocking sentence; Data points DPCSO01 row sends in place.
  Run log: requested/selected/applied/canceled/failed/observed with one correlation id per command.
- Independent client: `studio_probe` read DiscCSWI3 closed, DPCSO01 open, SPCSO028 on.

## Limitations

- The live relays were unplugged when this was built; the new path is proven on the simulated E10_BCU
  and the wire rig, not yet on the GE relays themselves.
- Command options are kept per IED for the app session, not in the project.
- Vendor coverage stays as LIVE-13 records it (mostly GE); no other vendor claim.
