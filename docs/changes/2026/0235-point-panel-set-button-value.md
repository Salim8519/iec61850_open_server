# 0235 Point panel: Set button sends the typed value

Date: 2026-11-22. Owner: "in sld, in the points panel, when we set any measurements value, it goes
zero, it happened in al kamil project". Layer: frontend only (`features/diagram/points/PointRowRun.tsx`).
No contract, engine or design change.

## Cause

The run log of the Al Kamil run (`point_panel_requested`, node `E05_BCU_points`, refs
`ACsrcMMXU1.VabRMS.mag.f` and `TotPF.mag.f`) shows the browser itself sent `"value": 0`. The points
have scale 1 and offset 0, so the conversion was not at fault. In the value box, clicking **Set**
first moves focus from the box to the button. The box's blur ended "typing", and the effect that
follows the live value reset the text to the current value before the click arrived (a normal
mouse press lasts long enough). `apply` then sent the old live value (0 for an idle meter).
Pressing Enter worked; clicking Set never did.

## What changed

- The value box ends "typing" only when focus leaves the box and its Set button together
  (`onBlur` on the wrapper, checking `relatedTarget`). Leaving the row without setting still shows
  the live value again; Escape still cancels.

## Evidence

- Real Chrome, the real component on a Vite test page (`.local/valuebox-harness`, scale 1, live
  value fed back after each set). Human-timed click (focus to Set, 120 ms, click):
  - old code: typed 33.3, sent 22.25 (the previous live value): bug reproduced;
  - new code: typed 33.3 sent 33.3, typed 0.95 sent 0.95; typed 77 then left the row: nothing sent,
    box back to 0.95; Enter with 12 sent 12.
- An instant automated click did not reproduce it on the old code, which is why it slipped through.
- `tsc --noEmit` clean; `pointPanel.test.ts` 12/12; `python studio/tools/verify.py` all PASS.

## Limitations

- Not clicked in the installed app (it was running the owner's Al Kamil session); the change is in
  the shared frontend only, so dev and packaged behave the same after the next build.
- No automated DOM test: the frontend tests run in Node without a browser.
