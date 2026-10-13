# Project format and SCL

Projects use a versioned .subsim ZIP archive. Include manifest, engineering model,
diagram, mappings, source definitions, scenarios, original SCL and referenced CSV.
Save atomically through a temporary sibling and replace; keep a recovery autosave.
Reject traversal paths, oversized archives and unsupported project versions.

The implemented archive has one `project.json` entry. Formats 2 through 8 deduplicate
original XML in a checksum-verified source table and regenerates derived runtime
models on reopen. New saves enforce a 100 MiB JSON bound before writing; legacy
format 1 has a bounded 256 MiB recovery allowance. Older builds cannot read format 2.
See [compact archive decision](../decisions/ADR-0008-compact-projects.md).

Format 3 added `test_panel` definitions and flat equipment mapping fields. Current
writes use format 12 with independent typed mappings, equipment actions, Trip
command/state polarity and explicit external-control compatibility profiles.
Formats 1/2 restore with no saved test controls; formats 1-3 migrate supported
equipment mappings. Opening any format 1-7 archive first creates an exact verified
recovery copy. Formats 1-5 receive Trip command Operate=0 / Reset=1; format 6
preserves its explicit command polarity. Every migrated Trip receives state
Operated=1 / Reset=0. Format 7 preserves both explicit fields and migrates all
controls to standard IEC behavior. Format 8 can explicitly store the warned
`concen_inverted_sbow` DPC profile. Format 11 (ADR-0029) adds `network`
(mode, adapter) and per-IED `role`, `live`, `lab_ip` and `goose_publish`; formats
2-10 open in loopback mode with every IED simulated. Format 12 (ADR-0033) retires the
TRANSFORMER/LOAD/BUSBAR kinds and busbar width with a migration notice. Older builds
reject newer formats.
Native signal/control models are rebuilt; pulse timers and live overrides are not
restored as actions. See the [mapping decision](../decisions/ADR-0009-report-mapping-and-signal-tests.md)
and [typed control decision](../decisions/ADR-0011-typed-equipment-controls.md).
The format-6 compatibility boundary is recorded in
[ADR-0016](../decisions/ADR-0016-format-6-trip-and-live-canvas.md).
The format-7 polarity split and recovery rules are recorded in
[ADR-0017](../decisions/ADR-0017-trip-polarity-and-sbo-observability.md).
The format-8 compatibility field is recorded in
[ADR-0018](../decisions/ADR-0018-concen-sbow-compatibility.md).

Original XML is retained independently of normalized runtime data. CID/ICD/SCD
imports resolve DataTypeTemplates and exact references, including ldName/arrays.
Unknown data is preserved and unsupported runtime features are diagnosed explicitly.
Imported IED-only files do not create imaginary bays.

Runtime addresses do not rewrite engineering addresses. Exports use original
addresses unless the user explicitly requests simulation addresses.
Exported XML must validate and re-import before being described as verified.

`POST /api/v1/ieds/{id}/export-scl` accepts `{ "overrides": { "exact/ref": value } }`.
It downloads an edited copy of the entire original source, retaining unselected
peer declarations and original addresses; it does not mutate the project. See
[export decision](../decisions/ADR-0004-preserving-scl-export.md) for supported
value encodings and remaining station-export work.

Startup validates and restores autosave engineering state without rewriting it or
starting IEDs. Unreadable files receive a separate recovery copy before new work
is allowed; failed preservation stops startup. The launcher exclusively locks its
data directory. See [recovery decision](../decisions/ADR-0007-startup-recovery.md).
