# 0010: Verified scalar SCL export

Date: 2026-09-09. Status: API verified; browser build passes, visual check blocked.

Implemented a preservation-first edited export and an explicit staged-value dialog.
XSD validation and semantic re-import compare exact references, kinds, FCs, values,
datasets and reports before download. Original XML/project state are never changed.
Source peers, vendor extensions, comments and engineering addresses are retained.
The UI distinguishes export from original download and explains unfinished topology.
See [decision](../../decisions/ADR-0004-preserving-scl-export.md).

Evidence:

- Eight focused tests pass: typed values, enum labels, timestamps, unknown references,
  invalid values, unsupported schemas, extension/comment/peer preservation,
  authenticated API, non-mutation and unsupported options.
- Complete Python suite: 60 passed in 13.99 seconds, including native protocol and
  portable-preview tests. Existing frozen executable predates this export change;
  its tests do not verify export inside a newly rebuilt portable bundle.
- Frontend production build and TypeScript checks pass (Vite 6.4.3).
- Read-only `audit_scl_export.py` samples two customer CIDs, performs a float edit
  in memory, validates and round-trips 36,021 selected-IED attributes. Both retain
  all five source IED declarations and unchanged source hashes; no customer XML
  or edited customer export is written into this repository.
- `440KV_Y210_BCU-1.CID` source SHA-256:
  `e30990b498c2242e2a99c15a33fb50a4651bf754dfc5cf42b33190f2fdad8b2f`.
- `210_ORGINALCID.CID` source SHA-256:
  `667abfc0bbfab4973d67037a62e461920d108596190b75c5a75aee4c49e973be`.
- Browser connection retried following the Browser skill; tool fails before setup
  with missing kernel-assets path (Windows error 3). No visual/click-through pass.

Limitations: indexed edits, setting-group selection, additional bit/octet encodings,
multi-source merge, topology export and simulation-address export remain unfinished.
The ongoing one-hour benchmark was not restarted or altered for these changes.
No full-release completion or hibernation is authorized by these partial results.
