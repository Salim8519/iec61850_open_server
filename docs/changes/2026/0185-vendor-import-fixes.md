# 0185 Vendor SCL import fixes

Date 2026-10-24. Go engine only. Decision [ADR-0114](../../decisions/ADR-0114-import-robustness.md),
specs [SLD import](../../specs/sld-import.md) (SLD-17), [signal flow](../../specs/signal-flow.md)
(FLOW-12), [GOOSE](../../specs/goose.md) (GOOSE-29 to 31). Native code, the frozen Python engine and
the frontend are unchanged.

## Problem (owner-approved bug list)

A bulk test pushed 181 vendor SCL files (`vendor_cids/`, `test_cids/`) through the app. Only 108 files
passed the import stage; 380 of 655 previewed IEDs imported (the earlier note's 379/500 counted a
subset). Root causes, each reproduced with a minimal synthetic SCL: Unicode255/Octet values parsed as
numbers (34 IEDs), Dbpos values written as names, LDevice inst starting with a digit (17 IEDs), FCDA
without doName blocking the whole IED (9 IEDs), quoted/multi-line strings, the 20 MB limit, anonymous
errors ("Unresolved EnumType None") that blocked a whole batch, Enum without Val read as 0 (Mod/Beh/
Health 0 on 18 of 24 running IEDs), DAs without trigger options (reports silent, GOOSE "applied" but
stNum unchanged), silent command drops, XSD failing in a data folder deeper than 260 characters,
picker requests hashing the whole SCL each time, and several misleading diagnostics.

## Change

| Item | Status | Where |
| --- | --- | --- |
| 1 Unicode*/Octet* values are text, missing → "" | fixed | `scl/types.go` |
| 2 Dbpos names → 0..3 (also Tcmd); other spellings refused naming IED + attribute | fixed | `scl/types.go` |
| 3 LDevice/FCDA ldInst follow XSD tLDInst | fixed | `scl/document.go`, `import_ied.go`, `reports.go` |
| 4 Enum without Val = lowest ord | fixed | `scl/types.go` |
| 5 IEC 61850-7-3 trigger options when the DA states none, with a note | fixed | `scl/trigger_defaults.go`, `builder_attr.go` |
| 6 Errors name IED + SCL path; preview flags IEDs that would fail | fixed | `scl/import_notes.go`, `builder*.go`, `preview.go`, `importer.go` |
| 7 FCDA without doName: only its dataset and blocks left out, with warnings | fixed | `scl/dataset_gaps.go`, `reports.go`, `catalogue.go`, `goose_controls.go`, `goose_links.go` |
| 8 `"`/line breaks: visible replacement with a note (native has no escape) | fixed (workaround) | `scl/import_notes.go`, `builder_attr.go` |
| 9 Command-only SPC/DPC and odd command shapes listed in a note | note only (native needs a status) | `scl/builder.go`, `controls_extended.go` |
| 10 Limit 100 MB | fixed (XSD helper still stops at 64 MiB) | `scl/document.go` |
| 11 XSD with a long data folder | fixed (Go side) | `store/xsd/workfile.go` |
| 12 Picker caches keyed by source identity, bounded | fixed | `scl/source_cache.go`, `data_classes.go`, `controls_extended_read.go` |
| 13 GSE/SMV without APPID: one warning | fixed | `scl/communication.go`, `goose_controls.go`, `sv_controls.go` |
| 14 Inputs from IEDs outside the file are a flow problem | fixed | `scl/flow_outside.go`, `flow_sources.go` |
| 15 SLD says why an IED cannot be loaded | fixed | `scl/preview.go` (`UnloadableIEDs`), `project/sld/bindings.go`, `project/sld_import.go` |
| 16 Export refusal in plain words + raw detail | fixed | `scl/export.go` |
| 17 LGOS setSrcRef ObjectReference form | fixed | `scl/goose_links.go` |
| 18 Unheard publisher outside the project counts as remote | fixed | `acquisition/goosedom/link_live.go` |

Parity: `scl/python_twin.go` switches the rules off for the golden tests only (`asPythonTwin`); the
goosedom and liveapi parity tests map the GOOSE summary; no golden file changed.

## Evidence

- Failing tests first: 15 of the new `scl` tests fail with the rules switched off (master behaviour), the
  flow and 100 MB tests fail on a master extract, `TestUnheardPublisherOutsideTheProjectIsRemote` and
  `TestLongDataFolderValidates` (real `studio_xsd.exe`, 301-character data folder: "The source file of
  the validate request could not be opened") fail on master; all pass on this branch.
- Go: full `go test -trimpath -p 2 ./...` through `verify.py` with the Python oracles (venv junction):
  PASS; `verify.py` 7/7 PASS. The Python golden suites (`TestCorpusMatchesPython` 1170 files over 299
  cases, `TestExportMatchesPython` 2407 exports, goose sources, template, XML errors) still match
  byte for byte with `asPythonTwin`; no golden file changed. Baseline before the change: same suite,
  only environment failures (`licensing` temp file once, two "Access is denied" test binaries).
- Bulk harness (same scripts, this branch's `studio_server.exe`, ports 8901-8903, fresh data folders,
  never Run), 181 files: import stage pass 108 → 154 (27 files have no importable IED at all and are
  now flagged in preview instead of failing), files with every IED imported 95 → 119, IEDs imported
  380/500 → 435/500 on the same files and 525/591 with the three 22-23 MB stations now inside the
  limit; 42 files improved, none worse. Remaining refusals per IED: arrays (future work), several
  servers (future work), duplicate type ids, non-standard Dbpos `bad`/`intermediate_state`, Enum
  without type, unknown LNodeType, `TimeStamp` spelling, INT32 Val `false`: each named in preview.
- Protocol (independent clients, native worker, same synthetic SCL without trigger options): Python cfg
  `DA(stVal 0 0 0 0 0)` gave no dchg report (`studio_probe` reports: GI only) and GOOSE stNum 1 → 1
  after the change; Go cfg `DA(stVal 0 0 0 1 0)` gave one dchg report (value true) and stNum 1 → 2.
  `studio_probe read` of the Unicode255 `NamPlt.d` returns "Feeder 1 protection".
- Long data folder: the harness probe validates and imports with 160, 215 and 245-character data
  folders (before: 503 from 247 characters).
- Size: parse 0.37-0.55 s, preview (parallel trial build) 1.1-1.5 s, import of all IEDs 3.9-4.4 s
  (30×SEL, 32×SEL; NRH stops at its multi-server IED) in a Go benchmark; ~3 GB allocated and 570-600 MB
  live heap after importing 30-32 IEDs, process ~1 GB. Over HTTP the harness saw 4-7 s per 20-IED batch.
- Picker cache, NRH 22 MB, TX2BCU: `CachedDataClasses` hit 28.0 ms → 60 ns per request.

## Limitations

- Native support needed (not changed): the XSD helper stops at 64 MiB, so 64-100 MB files fail schema
  validation; commands without status (932 in 28 files) cannot be operated because the native control
  binding needs a status attribute; the cfg format has no escape for `"` and line breaks.
- Preview of a 22 MB station takes about 1 s longer than before (it builds every IED once).
- The 7-3 trigger table was transcribed without a copy of the standard in the repository.
- Item 5 protocol evidence comes from a scratch script, not a committed protocol test (the protocol
  suite uses the frozen Python importer).
- Memory for very large stations (≈600 MB live for 30 SEL IEDs) is future work (FUTURE-WORK.md).
- Not checked in the browser UI or the packaged EXE (not built, by rule); the frontend is unchanged.
