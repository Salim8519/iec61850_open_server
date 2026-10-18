# 0096: Release 0.1.10 (plans 1 and 2 merged, installer and portable published)

Date: 2026-10-18. Owner request: merge plan 1 and plan 2 into master, then "commit to master
and make the installer exe and portable and push to website".

## Contents

- [0093](0093-tool-layout-and-filters.md): tool pages and Devices fill the window; one
  remembered filter bar on every list ([ADR-0064](../../decisions/ADR-0064-tool-pages-fill-window.md)).
- [0094](0094-goose-mms-signal-layers.md): SLD signal layers for GOOSE and MMS clients,
  simulated subscribers apply received GOOSE, MMS access summaries
  ([ADR-0065](../../decisions/ADR-0065-sld-signal-layers.md),
  [ADR-0066](../../decisions/ADR-0066-goose-inputs-and-mms-access-events.md)).
- Not included: the IED Explorer fix on its own branch (change 0095), not merged by request.

## Merge

Both branches touched the doc indexes, the ledger, `API.md`, `preferences.ts` and three
Devices files. Resolved keeping both:

- **Preferences** hold `list_filters` and the two SLD layer switches.
- **Devices rows** show the GOOSE hint through plan 1's `hints` slot on `DeviceListRow`.
- **Ledger.** Both branches chained to the same last entry. 0093 was kept, and 0094 was appended
  again with `history_log.py`, with its original content and time, so the hash chain stays
  valid. No master line was changed.

## Evidence

| Check | Result |
| --- | --- |
| `verify.py` on merged master | PASS |
| Python unit + integration + packaging | 418 passed, 2 skipped |
| Protocol (after native rebuild of merged master) | 66 passed, 9 skipped; before the rebuild the 2 MMS-access tests failed against the old worker binary |
| Frontend `node --experimental-strip-types --test` | 147 passed; `tsc --noEmit` clean |
| Release build `compile_protected.py --version 0.1.10` | exit 0; leak check: no studio_app source, bytecode, symbols or loose UI files |
| `ScadaBench-Setup-0.1.10.exe` | 24,530,262 B, SHA-256 `f0509f98…8b5d7c` |
| `ScadaBench-Portable-0.1.10.exe` | 29,358,547 B, SHA-256 `80e23348…5b5471` |
| Portable `--self-test` in an empty folder | ok (MMS 42.5, arrays, export, recovery, deletion, saved panel and pulse) |
| Installed folder `--self-test` | ok |
| Website deploy (`deploy.cmd`) | installer on Pages, portable uploaded to R2 |
| Live site | `/`, `/download/`, `/changelog/` 200; download page links both 0.1.10 files; changelog shows 0.1.10; old setup link 302 to 0.1.10; both downloaded files match the SHA-256 above |

## Limitations

- Unsigned; SmartScreen warns. Not installed on a clean PC; the owner's installed copy was not
  updated.
- The release was not exercised in the running packaged app beyond `--self-test` (no lab-mode
  GOOSE or "Who is reading" run from the EXE).
- Feature limitations of 0093 and 0094 apply unchanged (no physical VLAN switch, no listen-only
  lab mode, no Sampled Values on the SLD).
- The first live check returned no response once (transient); a retry gave 200 for every page.
