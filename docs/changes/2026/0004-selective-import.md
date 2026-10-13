# Change 0004: choose IEDs before importing

Date: 2026-09-09. Requested by user after the portable preview.

CID/ICD/SCD uploads now open a searchable IED-selection dialog. Nothing is selected
by default, including single-IED files. Users choose the IEDs and confirm, or cancel
without adding anything. Existing names and unsupported/no-server entries show a
reason and cannot be selected. The dialog shows available project slots.

POST /scl/preview parses XML safely but neither compiles models nor mutates the
project. POST /scl/import requires an explicit selected_names array. Only those
IEDs are compiled. Unknown, empty and duplicate selections fail without insertion.
The selected IED retains the complete original SCL; reopening a project compiles
only its selected IEDs, without pulling other IEDs back in.

Evidence: integration tests cover non-mutating preview, selection of one of four
IEDs, exact original XML retention, project reopen, existing-name diagnostics,
and rejection of missing/empty/unknown/duplicate selections. Frontend builds pass.
Browser visual acceptance is still pending because browser automation setup fails.

Concurrent runtime work adds atomic native update batches for measurement value,
quality and timestamp. Missing companions are not invented. Signal table metadata
reads the same quality/time attributes served by MMS. This is separate from the
remaining richer quality editor and switching behavior work.
