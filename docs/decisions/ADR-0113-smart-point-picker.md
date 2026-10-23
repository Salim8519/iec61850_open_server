# ADR-0113: Smart point picker and the complete IEC 61850 kind table

Accepted: 2026-10-23. Owner request with a screenshot of a CB sheet ("Position comes from → Two points:
open and closed contacts", points such as `System/ConInGGIO1.Ind106.stVal`): finding a point in the
SLD pickers is still a hassle; the type filter must cover all IEC 61850 standard points. The owner
approved an interactive drawing. Change [0184](../changes/2026/0184-smart-point-picker.md), spec
[point-picker](../specs/point-picker.md). Extends [ADR-0095](ADR-0095-universal-point-kinds.md) (one
classifier, kind tokens, `KindBadge`) and follows [ADR-0076](ADR-0076-go-only-development-and-parallel-workstreams.md)
(new backend code in Go only; the browser displays the engine's answers).

## Problem

The picker's "All types" dropdown was easy to miss, listed kinds that cannot match the slot (an open
contact takes only on/off points but offered Double point), showed no counts, and a type alone cannot
find `Ind106` among hundreds of GGIO inputs. The classifier also answered UN (unknown) for several
standard cases: `subVal` (SV Dbpos) counted as a double point, `blkEna`, `opOk`, MX `range`, control
block attributes, ObjRef and Currency had no kind.

## Decision

1. **One complete kind table**, the same in Go (`signalindex/point_kind.go`) and the browser
   (`points/pointKind.ts`, `kindOf61850`), checked by both against one shared file
   `studio/tests/fixtures/point_kinds/iec61850.json` (786 cases: every 7-2 functional constraint ×
   every IEC 61850-6 leaf basic type, plus data-class cases). The rules apply in this order:

   | Rule | Kind |
   | --- | --- |
   | FC SV (subEna, subVal, subQ, subID) or BL (blkEna): set by an operator, never a live process value | CF |
   | Basic type Quality, Timestamp or EntryTime: helper attribute | AT |
   | FC CO: class SPC → SC, DPC → DC, INC/ENC/BSC/ISC/APC/BAC → SE; no class: a number → SE, else CM | SC/DC/SE/CM |
   | FC SP, SG, SE, CF: settings | CF |
   | FC DC, EX, SR (service tracking) and the control-block FCs BR, RP, LG, GO, MS, US | TX |
   | Any other FC (not 7-2): never guessed | UN |
   | FC ST, MX, OR of a setting class (SPG, ING, ENG, ORG, TSG, CUG, VSG, ASG, CURVE, CSG) | CF |
   | … of a description class (DPL, LPL, CSD), or a text type (VisString*, Unicode*, Octet*, ObjRef, Currency, Check, PhyComAddr, TrgOps, OptFlds, SvOptFlds, LogOptFlds, EntryID) | TX |
   | Dbpos | DP |
   | BOOLEAN (ST, MX and OR alike: opRcvd/opOk follow ST) | SP |
   | Number (INT8…INT128, INT8U…INT32U, INT24, FLOAT32/64): MX or class HST → MV; class BCR or SEC → CT; else IS | MV/CT/IS |
   | Enum or Tcmd (MX `range` included) | IS |
   | Anything else (vendor types, Struct) | UN |

   The data class only decides where the type cannot: commands, counters (BCR, SEC: numbers only, so
   SEC `sev` stays IS), histograms (HST), settings and description objects. A status or measurand class
   never overrides the attribute's own type (ACD `dirGeneral` is IS, BSC `valWTr.transInd` is SP).
   `kindOfDataClass` (lists that know only the class) covers every 7-3 Ed2/Ed2.1 class: SPS/ACT/ACD SP,
   DPS DP, INS/ENS IS, BCR/SEC CT, HST MV, VSS TX, MV/CMV/SAV/WYE/DEL/SEQ/HMV/HWYE/HDEL MV, SPC SC,
   DPC DC, INC/ENC/BSC/ISC/APC/BAC SE, settings CF, DPL/LPL/CSD TX. Changed answers compared with
   ADR-0095: SV Dbpos DP → CF, SV Quality AT → CF, MX/OR BOOLEAN UN → SP, MX Enum UN → IS, ST Tcmd UN → IS,
   EntryTime UN → AT, ST Enum with class BCR CT → IS (a counter is a number), the text and control-block
   types UN → TX.
2. **Go-only parameters of `GET /ieds/<id>/signals`** (added keys only; a request without them is
   answered byte-identically and the Python golden test still passes):
   `annotate=1` gives each item `point_kind`; `ln=<LD/LN>` filters one logical node (the reference up to
   the first "." after its "/"); `ln_counts=1` adds `{"LD/LN": n}` counted with every other filter but
   `ln`; `near=<text>` adds `suggested`: at most 8 matching items (every other filter applied, paging not)
   whose description, then reference, holds the text as a whole word, case ignored, each with
   `suggested_by` (`description` or `reference`). Words are runs of letters and digits, so `Q11` is in
   "Q11 open" and "Bay_Q11" but not in "Q110". Only text in the model counts; nothing is guessed.
3. **Speed.** Kinds are computed once per index and data-class version (`kind_cache.go`, keyed by the
   identity of the classes map that `scl.CachedDataClasses` returns for one SCL source) and read as one
   byte per signal; logical nodes are built on first use (`logical_nodes.go`), so `Build` is unchanged;
   matches, kind counts and node counts come from one pass (`scan.go`). 90 000 signals: the type filter
   with counts went from ~88 ms to ~2.9 ms per request; a full picker request takes ~6.3 ms.
4. **The picker** (`components/SignalSearchPicker.tsx` and `components/signalPicker/`): type chips with
   the shared badge and engine counts (only kinds with rows, the chosen one always, "All types (n)"),
   the slot's kind chosen for the user with a one-line reason (`preferredKind`: contacts SP, one
   position point DP, measurements and analog values MV, tap position IS, a number command SE, a named
   command kind), an object (logical node) select with counts, suggestions first, quality and time
   hidden until "Show quality and time" (shown only where helpers can appear), rows with Go's kind.
   The point panel adder gets the same object select. A type the user picked is remembered per picker
   name and slot type for the browser tab.

## Compatibility

No project data, IED model or Flask-shaped answer changes. The frozen Python engine ignores the new
parameters: the picker detects the missing `kind_counts`, starts on All types and shows the old type
dropdown, as before. Every list that uses `kindOf61850` (Devices data points, Compare, Explorer, Test)
shows the new kinds for the formerly unknown cases. Not added: AND-search of several words (q stays one
substring, so "106" and "Ind106" work, "GGIO1 106" does not).
