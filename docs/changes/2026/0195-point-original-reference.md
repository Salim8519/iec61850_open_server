# 0195: A renamed point still shows its IEC 61850 reference

Date: 2026-10-27. Owner, on the point panel: "when user rename the iec61850 path, we cant see the
original path ... it is ok to edit the name ... but what about the original iec61850? can u add
small line for it?"

## Change

| File | Why |
| --- | --- |
| `frontend/src/features/diagram/points/PointRow.tsx` | once a point has its own name, a small line under the row shows "IEC 61850 LD/LN.DO.DA" (the original reference, IED name dropped as in the row); unnamed points show the reference as their name, so no extra line |
| `frontend/src/copy/pointPanel.ts` | the words "IEC 61850" and a tooltip: renaming never changes the reference |
| `frontend/src/features/diagram/points/point-dialog.css` | `.pp-ref`: 12 px, secondary text colour, wraps long references |

## Evidence

- Dev app, demo station (FEED1 point panel, `LD1/XCBR1.Pos.stVal`): before renaming, one line;
  after renaming to "Feeder 1 breaker" the row shows "IEC 61850 LD1/XCBR1.Pos.stVal" under the name.
- Frontend 488/488; types clean; file lengths pass.

## Limitations

- Display only; the search box already matched references. The reference line is not shown on the
  SLD card pins.
