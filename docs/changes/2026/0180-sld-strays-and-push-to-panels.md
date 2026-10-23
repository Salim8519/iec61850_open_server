# 0180 SLD: far-away symbols never hide the drawing; push buttons into point panels

Date 2026-10-23. Frontend only (the point panel is a Go-only kind, ADR-0102; nothing new is stored).

## Problem (owner report on 0.1.23)

The owner's project opened on an empty SLD. The saved diagram was complete (586 symbols, 129 wires).
One unwired symbol, INFEED-5, sat at y = 23 460 while the drawing spans y = -360 to 3 995: it was added on
22 Oct by the old Add placement (position from the node count, fixed for new symbols in 0176 B7). Fit view
fitted the drawing plus that symbol, hit the minimum zoom 0.2 and centred on empty canvas.

## Change

- `equipment/strayNodes.ts`: a symbol with no other symbol within 3 000 flow units is a stray, when strays are
  fewer than a quarter of all symbols (a drawing of scattered single items is left alone). Initial Fit view
  and the Fit view button fit the drawing without the strays (`Canvas.tsx`).
- `features/diagram/StrayNotice.tsx`: a quiet note at the bottom of the canvas names the strays ("INFEED-5 is
  far away from the drawing, so the view leaves it out.") with "Bring it here", which moves them just below the
  drawing's left edge in one Undo step. The note shows only while strays exist.
- `equipment/pushToPanel.ts` + `features/diagram/PushToPanels.tsx` (owner request): in a push button's
  properties, "Put push buttons into point panels" turns every unwired latching status or command push button
  into rows of one point panel per IED, placed where that IED's buttons were. Latching status buttons become
  ON/OFF status rows (group "Status inputs", the row's ON/OFF sets the same signal); command buttons become
  command rows (group "Commands"). More than 300 rows split into "<IED> points 2". Wired, self-resetting and
  unbound buttons stay buttons. One diagram change, so Undo brings the buttons back.

## Evidence

- `tests/frontend/pushToPanel.test.ts` (3 tests): grouping per IED, statuses first, position, refs kept,
  wired/self-resetting kept, 300-row split, stray detection and Fit view options. Frontend 453/453, tsc clean,
  file lengths PASS, verify.py (see the ledger).
- Copy of the owner's project on a dev server (owner's data untouched): the SLD opened on the drawing with
  the note naming INFEED-5. "Put push buttons into point panels" turned 452 buttons into 8 panels (44-61 rows),
  the engine accepted and saved the diagram. With the station running: a status row set ON and OFF
  (C03_Y1105Master/GGIO1.Ind012.stVal) and a command row sent GGIO2.SPCSO002, both answered 200.

## Limitations

- Converted rows use the button names; the panel groups rows only as "Status inputs" and "Commands".
- A push button's own colours, rest value and reset time are not kept (a panel row sets ON or OFF directly).
- The owner's installed app needs a release with this change; their project is not changed until they press
  the button.
