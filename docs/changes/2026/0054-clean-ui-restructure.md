# 0054: Clean four-place UI restructure

Date: 2026-10-14. Owner request: complete restructure; clean like an iPhone,
beginner and older-user friendly, pro level. Decisions:
[ADR-0032](../../decisions/ADR-0032-clean-four-place-ui.md),
[ADR-0033](../../decisions/ADR-0033-feature-consolidation-format-12.md).

Change:
- New shell: `app/Sidebar.tsx`, `app/TopBar.tsx`, `app/places.ts`, routing in `App.tsx`.
- New design tokens and base styles; shared controls in `src/ui/`.
- New places in `features/diagram/`, `features/devices/`, `features/test/` and
  `features/settings/`. Removed: Palette, Inspector, Signal Test, Scenarios, Real
  IEDs tab, Network tab, endpoint table and bulk delete UI.
- Spec [UI structure](../../specs/ui-structure.md) with 7 enforced rules.

Evidence (browser, isolated data folder, real CIDs):
- Imported the Hitachi RED670, GE P543 and Siemens files.
- Breaker CB-2 was linked to the Siemens `CSWI1.Pos`; its position filled itself in.
  Run, then Close: the canvas showed Closed and an independent probe read Dbpos `10`.
- A test point was pulsed, and a sine generator on the Hitachi `CMMXU1.A.phsA` read
  110 → 102 → 90 A over MMS.
- GE was set to Real device against a stand-in: Live by report block `urcbA01`, and
  the comparison showed 15,209 equal points. The block was free again after Stop.
- Review-driven fixes during the walkthrough:
  - one-IED files are pre-selected on import;
  - the IED list shows how many breaker controls each device has;
  - search rows show the short reference;
  - zoom is limited for a single symbol;
  - no false "another tab" message;
  - the live state now refreshes.

Tests: frontend 59 pass, Python suite passes, build and verify pass.

Limitations: owner visual acceptance pending. Lab-network Apply still not executed on
this PC (UAC). Screens were checked at 1366×768 and 1920×1080 only.
