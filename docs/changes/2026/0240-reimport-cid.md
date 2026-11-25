# 0240 Re-import an IED's CID, keeping its id, settings and links

Date: 2026-11-23. Owner: "can u add re import cid option? so same cid gets updated without deleting
equipment? or warns user what will be lost?". Decision
[ADR-0150](../../decisions/ADR-0150-reimport-cid.md). Layers: Go engine (`project`, `project/service`,
`store/recovery`, `api/projectapi`), frontend (`features/devices`, `copy`), docs.

## Change

- `project.Reimport` / `ReimportedIED` and the link pruner (`reimport.go`, `reimport_prune.go`);
  `dropNodeLinks` (from `WithoutIEDs`) and `settleTapCommand` (from `dropTapChangerOf`) shared.
- `Service.Reimport` with a comparison answer, a verified `before-ied-reimport-*` backup and one
  Undo step; `POST /ieds/{key}/reimport` (Go-only).
- Devices › Files: "Re-import CID…" dialog with the comparison, the links that cannot stay and the
  gateway points that will not update.

## Evidence

- Go: `reimport_test.go` (identity and settings kept, three precise losses in words, other IEDs
  untouched, gateway point named, result validates; same file changes nothing; another IED refused);
  `project`, `project/service`, `store/recovery`, `api/projectapi` pass.
- Real site file (local only): E10_BCU's stored CID with `Disc0XSWI3` removed. Preview in 1.1 s:
  data points 31,476 → 31,248, commands 294 → 290, 114 links kept, 1 lost ("SLD BB-1B DS 244 VT
  DISCONNECTOR command · E10_BCUSystem/Disc0XSWI3.Pos — not in the new file"); the unchanged file
  answers "same file", nothing lost. Applied through the dialog (headless Chrome 1366x768): file
  E10_v2.cid, role, lab and monitoring addresses unchanged, only that command cleared, backup
  written; Undo restored the old file, 31,476 points and the link.

## Limitations

- The IED name must stay the same (references start with it). A renamed IED is refused.
- A link whose reference moved to a new name is not followed; it is listed as lost.
