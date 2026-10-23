# ADR-0114: Import robustness for vendor SCL (Go engine)

Accepted: 2026-10-24. Owner-approved bug list from a bulk test that pushed 181 vendor SCL files
(`vendor_cids/`, `test_cids/`, 655 IEDs) through the app: only 108 files imported completely.
Change [0185](../changes/2026/0185-vendor-import-fixes.md). Follows
[ADR-0074](ADR-0074-go-engine.md)/[ADR-0076](ADR-0076-go-only-development-and-parallel-workstreams.md):
the rules below exist in the Go engine only; the frozen Python engine keeps its behaviour.

## Decision

1. **Size limit 100 MB** (`scl.MaxXML`, was Python's 20 MB; owner decision). Real stations of
   22-23 MB (NRH multivendor, 30×SEL, 32×SEL) now import. The XSD helper reads at most 64 MiB
   (`native/xsd/file_io.h`), so a 64-100 MB file still fails schema validation until the helper's
   limit is raised (native change, not made here).
2. **Value kinds.** `Unicode*` and `Octet*` values are text, like `VisString*`; a missing value is
   `""`, never `0`. Unicode255 keeps its cfg value (`="…"`, libiec61850 `UNICODE_STRING_255`, read
   back over MMS by an independent client); Octet values stay out of the cfg as before.
3. **Two-bit enumerations by name.** `Dbpos` Val `intermediate-state`/`off`/`on`/`bad-state` → 0..3
   and `Tcmd` `stop`/`lower`/`higher`/`reserved` → 0..3 (IEC 61850-7-2 / SCL). Other spellings
   (`intermediate_state`, `bad`) stay refused, with the IED, the attribute and the standard names.
4. **`LDevice@inst` and `FCDA@ldInst`** accept the XSD `tLDInst` pattern `[A-Za-z0-9][0-9A-Za-z_]*`
   (≤ 64), so "1" or "252L2" import. Everything accepted before stays accepted.
5. **Enum without Val** reads the **lowest ord** of its EnumType (Mod/Beh → on(1), Health → Ok(1)),
   not 0, which is not a value of most enumerations. An EnumType without values still reads 0.
6. **Trigger options.** When a DA of a DOType states none of `dchg`, `qchg`, `dupd` (IEC 61850-6
   tDA), it gets the TrgOp of its position in IEC 61850-7-3 Ed2 clause 7 (the common data class
   tables' TrgOp column); one import note counts them. Without them libiec61850 never reports or
   publishes a change (ABB IED670 Ed1: 341 dataset members, NR TX2BCU: 98). A DA that states any of
   the three keeps exactly what it states; a BDA inherits its DA's options as before.

   | FC | Attribute | TrgOp |
   | --- | --- | --- |
   | ST | stVal; general, dirGeneral, phsA/B/C, neut, dirPhsA/B/C, dirNeut (ACT/ACD); cnt (SEC); valWTr (BSC/ISC); stSeld | dchg |
   | ST | actVal (BCR), hstVal (HST) | dchg, dupd |
   | ST | frVal (BCR) | dupd |
   | ST, MX | q | qchg |
   | MX | mag (MV), cVal (CMV) | dchg, dupd |
   | MX | range, rangeAng, mxVal (APC/BAC) | dchg |

   Not defaulted (no TrgOp in 7-3, or outside status/measured values): t, instMag, instCVal, subVal
   and the FCs CF, DC, SP, SG, SE, OR, BL, EX. The table was transcribed from the standard's tables;
   the repository holds no copy of 7-3, so an expert check of the list is welcome.
7. **Errors and preview.** An import error names the IED and the SCL path ("IED R1 at
   R1LD0/GGIO1.Ind1.dirX: bType is Enum but the attribute names no EnumType …"); Python's "None"
   no longer appears in import errors. `POST /scl/preview` builds each IED as the import would
   (without the XSD check) and gives an IED that would fail `unavailable_reason` "Cannot be imported
   at …", so the user can leave it out instead of one IED blocking the batch.
8. **FCDA without doName** (optional in the XSD, "all data of the LN"): ScadaBench does not guess its
   members. That dataset is left out with a warning naming the member, and the report and GOOSE
   blocks using it are not published (like a block without dataset); the rest of the IED imports.
9. **Text the native model cannot hold.** The cfg format ends a value at the first `"` and a record
   at a line break and has no escape (`config_file_parser.c` `setValue`/`terminateString`, read only).
   Such values are simulated with `'` and a space; one note names every attribute. The stored SCL is
   unchanged, so export and re-import are unaffected (explicit, narrow, visible, reversible).
10. **Commands that cannot be operated** are noted, never silent: commandable SPC/DPC without stVal
    (the native control path needs a status attribute, `worker/controls/controls.cpp`), and
    INC/ENC/APC/BAC/BSC/ISC whose status or ctlVal type is not their class's (Ed1 INC Mod with Enum,
    BSC with BOOLEAN ctlVal, DPC with Enum stVal). Exposed controls are unchanged.
11. **Communication and links.** A GSE/SMV without APPID gives one "has no APPID" warning and no
    second "no network address" one. LGOS `setSrcRef` in the ObjectReference form `LD/LLN0.cb` binds
    like `LD/LLN0$GO$cb`. Signal flow reports inputs without serviceType from an IED outside the file
    as one problem per source IED. The SLD says why an IED of the file can never be loaded. GOOSE
    health counts an unheard publisher that is in the file but not in the project as outside the
    project (`remote`), not "not heard". An export refusal of a schema-invalid file starts in plain
    words and ends with "Detail: line N: <XSD message>".
12. **Long data folders.** The XSD work file is passed to the helper as an extended-length path
    (`\\?\C:\…`, `\\?\UNC\…`), so a data folder deeper than 260 characters validates.
13. **Picker caches** (`CachedDataClasses`, `CachedExtendedControls`) find an answer by the identity of
    the source text, hashing a text only the first time it is seen; at most 64 entries each.

## Parity with the frozen Python engine

`scl/python_twin.go` holds one switch that turns these rules off. Only the golden tests set it
(`asPythonTwin`), so they still prove every other byte of preview, validate, import and export equal to
Python's goldens; each rule has its own test in `vendor_*_test.go`. Outside `scl`, the tests map the one
changed answer instead: GOOSE link summaries (`goosedom` reverse, `liveapi` `goExpected` forward) and
the SLD golden test passes no unloadable reasons. No golden file was regenerated or edited.

## Compatibility

Projects re-import their stored SCL on load, so an opened project gets the new enum defaults, trigger
options and notes; signal references do not change, so mappings and diagrams keep working. IEDs that
only Go can import (for example with Unicode255 values) cannot be restored by the Python engine. More
reports and GOOSE retransmissions follow from item 6; that is the device's standard behaviour.
