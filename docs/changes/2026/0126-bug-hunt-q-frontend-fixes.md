# 0126: Real-time bug hunt (lane Q): 7 frontend bugs fixed, speed findings

Date: 2026-10-20. [Speed plan](../../architecture/go-engine/speed-plan.md) phase 0; owner: "fix and
check all bugs". Every function tested in the built-in browser at 1366x768 on a separate product
launcher instance (temp data folder, port 20517, demo station `scd/open_substation.scd` moved to
127.67.0.2-5:20102, worker copies, gateway on 127.0.0.1:20404/20406, independent IEC 104 client
`studio_test_iec104_master`; network setup preview only). The owner's app was not touched.

## Bugs fixed (frontend only; no Python change, so nothing for Go to mirror)

- **B7 wires saved but never drawn.** North/west terminals were target-only handles; a wire whose
  source end was north or west (north-north, north-west, west-west) was saved and used by
  energization but React Flow could not draw, select or delete it. Each terminal is now a target
  and a source handle with the same id (`equipment/EquipmentNode.tsx`, `Terminal`); existing wires
  unchanged; 9 of 9 edges drawn after the fix.
- **B4 Start/Connect press lost after typing a setting.** Blur-save set `busy`, disabling the
  button between mousedown and click; the gateway then started on the old port on a second click.
  `features/tools/gateway/actionQueue.ts` queues saves quietly before Start/Connect
  (`GatewayTool`, `MasterTool`, `ConverterTool`); one click now saves 20406 and starts on it.
- **B5** master connect waited up to 10 s with no feedback: spinner "Connecting to host:port..." and
  gateway "Starting..." (`MasterConnect.tsx`, `GatewaySettings.tsx`, `copy/master.ts`, `copy/gateway.ts`).
- **B6** network sentence said "allow port 102" for IEDs on 20102; button read "Put  IEDs". Uses
  the firewall item's `ports` (`api/network.ts` type, `setupWords.ts`, `NetworkSetup.tsx` `putLabel`).
- **B1** clipped hint text in Devices "Whole station" rows (owner rule): rows wrap (`devices.css`).
- **B2** disabled "Add test point" while running had no reason: tooltip and run-aware subtitle.
- **B3** sequence steps showed raw "set"/"assert" and lower-case "failed": `sequenceWords.ts`.
- Plurals ("1 logical devices", "1 checks", "1 values", "1 notes"): `copy/count.ts` `countOf`.

## Evidence

- Re-checked every fix in the browser. Frontend tests 189/189 (11 new in `bugHuntQ.test.ts`);
  `npm run build` clean (dist rebuilt); Python unit+integration pass; file lengths pass.
- Speed findings (Python today): operate click-to-symbol 140-200 ms (first after Run 1084 ms);
  Run 4 IEDs 1.76 s cold, 0.45-1.1 s warm; Stop ~0.35 s; Explorer operate 405 ms + 247 ms read;
  gateway visible 1.1 s (1.5 s status poll); master connect to a refused port waits t0 = 10 s;
  `GET /project` 7 times in the first second (`state/useStudio.ts` refresh per replayed event);
  `/activity` polled every 2 s on every page. Good: test buttons 9-65 ms, writes 4 ms, page
  switches 1-12 ms, add equipment 6 ms.

## Not fixed (design or owner decision)

- Activity panel has no filter and is flooded by generator events; Find IEDs "This PC" is fixed to
  127.50.0.x; Explorer cannot pick a running simulated IED; Export SCL reason only in a tooltip;
  the sheet keeps the last "Test: open set..." note after Stop/Run; empty generator list without a
  hint for IEDs with no measurements. Speed items go to the push/UI and native lanes.
