# ADR-0004: Verified value export preserves source scope

Status: accepted, 2026-09-09.

An edited export is a copy of one original source document, not a newly merged
station. It retains all original IEDs, including unselected GOOSE peers, namespace
extensions, comments, communication addresses and type declarations. Only explicit
scalar instance values of the selected IED may change. Import selection remains
independent of which IED declarations exist in the exported engineering document.

The exporter requires a bundled, passing XSD, validates its output, then re-imports
and compares exact signal references, types, FCs and values plus report/dataset
definitions. Failure returns a diagnostic, never a supposedly verified file.
Original-source download remains separate and untouched.

Supported edits initially cover Boolean, integer, floating-point, declared enum,
text and timestamp values. Indexed overrides, setting-group selection and other
bit/octet lexical encodings are rejected rather than guessed. Edited exports do
not change the active project, native model, engineering addresses or source file.

The browser editor stages explicit initial-value changes, with an editable list
before download. It explains source scope and unchanged engineering addresses;
it does not silently capture live values. Unsupported scalar encodings are disabled.

This establishes the export boundary. Multi-source SCD merging, physical topology
and explicit simulation-address export remain separate unfinished work; this is
not claimed as complete station export. Browser interaction acceptance is pending.
