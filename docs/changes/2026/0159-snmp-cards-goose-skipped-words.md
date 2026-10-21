# 0159: SNMP device cards and Connect button never clip; a failed Start says so; skipped IEDs in GOOSE traffic

Date: 2026-10-22. Found during the real host/VM SNMP and GOOSE test
([test record](../../testing/vm-host-real-vm-2026-10-22.md), change
[0157](0157-vm-box-and-guide.md)). Design: [layout](../../design/layout.md) (no clipped text, errors
inline in the tool's left column), [interactions](../../design/interactions.md). Frontend only; the
Go engine and the frozen Python backend are unchanged (their GOOSE watch list stays byte-identical).

## Why

- An SNMP device whose address was not on the PC showed a blank card in the device list with its red
  problem text cut off on both sides. Cause: the card is a button, buttons are `white-space: nowrap`
  app-wide, and the button's centred grid made one column as wide as the unwrapped text.
- "Connect and read everything" in SNMP Read devices was cut off in the narrow left column.
- Pressing Start when no device could open its address seemed to do nothing ("Devices are stopped");
  the reason was only in the clipped card.
- GOOSE traffic called a skipped project IED "not in project", listed its GOOSE among messages "not in
  your project", and said "Nobody in project" when the only listener was a skipped IED. The engine leaves
  skipped IEDs out of its watch list, so the frontend now names them.

## What changed

- `features/tools/snmp/snmp.css`: device cards use one full-width column, start alignment and normal
  wrapping; the Connect button may wrap onto two lines.
- `SimulateView.tsx`, `copy/snmp.ts`: under Start, "The device could not start. The reason is under it in
  the list." (or "N devices ... each one"), ⚠ and words, shown while any device failed.
- `features/network/gooseSkipped.ts` (new), `GooseTraffic.tsx`, `TrafficTool.tsx`, `copy/hints.ts`: a
  skipped publisher reads "skipped in this project"; a block only skipped IEDs listen to reads
  "C19_Y1005 (skipped, so not listening)"; GOOSE on the wire from a skipped IED is listed apart with
  "from <IED>" and its own one-line explanation. Truly unknown IEDs still read "not in project".

## Evidence

- Frontend tests 319/319 (new `gooseSkipped.test.ts`, `snmpCards.test.ts`); `tsc --noEmit` clean.
- verify.py: lengths, architecture, docs, history, specs, GOOSE engine PASS; Go suite PASS except
  `internal/gateway`, whose test binary Kaspersky blocked in %TEMP% ("Access is denied"); rerun with the
  whitelisted temp folder: ok (26.5 s).
- Browser at 1366×768 (Go dev server built from this tree, demo data in a private folder): a device on
  10.250.250.250 shows "⚠ Problem" and the full reason wrapped inside its card, plus the red line under
  Start; the Connect button shows its full label on two lines; GOOSE traffic shows "skipped in this
  project" for a skipped C11 and "C19_Y1005 (skipped, so not listening)" for a skipped C19; a scan for
  `scrollWidth > clientWidth` found nothing clipped except the intended screen-reader-only label.

## Limitations

- Not yet in a release; the VM still runs 0.1.20.
- Only these four problems were fixed; a full page-by-page UI review is a separate step.
