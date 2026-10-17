# 0079: Tools place, with GOOSE & Sampled Values traffic as its first tool

Date: 2026-10-17. Decision: [ADR-0054](../../decisions/ADR-0054-tools-place.md).
Spec: [Tools](../../specs/tools-page.md).

Owner request: add SCADA helper tools in one tab called Tools: an IED client/browser,
finding IEDs, traffic monitoring and a gateway simulator. The owner chose the scope and order
(2026-10-17):
1. IED Explorer, Find IEDs and the GOOSE/SV inspector first;
2. then the gateway simulator, speaking IEC 104, Modbus TCP and DNP3. It has simulated
   sources plus read-only real IEDs, and an editable point list proposed from the SCD.

## Change

- **The sidebar has five places:** Devices, SLD, Test, **Tools**, Settings (Wrench icon).
  - `app/places.ts`, `Sidebar.tsx` and `App.tsx` were updated.
  - The open tool is held in `App`, so Settings can jump straight to it.
- **The Tools page** (`features/tools/ToolsPage.tsx`, `tools.css`, `copy/tools.ts`):
  - It shows large cards with an icon, a name, one line and a "What is this?" hint.
  - An open tool has an "All tools" back button.
  - Only delivered tools are listed.
- **First tool: GOOSE & Sampled Values traffic** (`features/tools/TrafficTool.tsx`).
  - It reuses the unchanged `GooseTraffic` and `SvTraffic` cards.
  - These cards moved out of Settings → Network, which now shows one link to them.
- **Tour:** a new Tools step (8 steps). The tour version is unchanged, so it does not reopen
  by itself.
- **Specs:**
  - UI-01 now has five places; the tour and SV interface lines were updated.
  - New `tools-page.md` (TOOLS-01..04).
  - Proposed specs: `ied-explorer.md`, `find-ieds.md`, `traffic-inspector.md` and
    `gateway-simulator.md`.

## Evidence

- `npx tsc --noEmit`: no errors.
- `node --test studio/tests/frontend/*.test.ts`: 105 pass, including the new
  `tools.test.ts` (4 tests) and the renamed `sidebar has exactly five places`.
- `python studio/tools/verify.py`: see the ledger entry.

## Limitations

- **Not checked in the running app yet:** the owner's visual acceptance is pending.
- **No backend or native change.** The traffic cards behave exactly as before.
- **Tutorial videos** still show GOOSE traffic under Settings. The Settings link covers this
  until they are re-recorded.
- **Still only proposed:** IED Explorer, Find IEDs, the inspector extensions and the gateway
  simulator. They are not capabilities yet.
