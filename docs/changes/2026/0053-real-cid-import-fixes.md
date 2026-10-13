# 0053: Real vendor CID files import and run

Date: 2026-10-14. Owner request: test with real CID files. Decision:
[ADR-0033](../../decisions/ADR-0033-feature-consolidation-format-12.md).

Cause: an import audit of 134 owner CID files (GE P543/C60, Siemens, Hitachi RED670,
copied into the session scratchpad, never committed) found 51 failures:
- 50 had `bType="PhyComAddr"` (GOOSE tracking `ALTRK1.GocbTrk.dstAddress`), which
  the type map did not know.
- 1 GE C60 had a `ReportControl name=""`.

Change: `scl/type_resolver.py` maps PhyComAddr to libiec61850 type 29 (structured,
no initial value); `scl/reports.py` and `scl/report_catalogue.py` skip unnamed report
blocks with an actionable warning.

Evidence:
- Audit after the change: 134/134 import (max 4.7 s, max 54,871 points), 133 schema
  valid, 41 publish GOOSE. Every one of the 134 starts in the native worker and
  answers a read (max start 2.0 s).
- Through the UI, real Hitachi, GE and Siemens files were imported, run and browsed
  with the independent probe: 4,156 / 17,513 / 4,000 MMS leaves.
- Regression tests: `test_real_world_scl.py` (synthetic fixtures, no customer data).

Limitations: PhyComAddr values are served as their MMS structure but cannot be edited
in the value table.
