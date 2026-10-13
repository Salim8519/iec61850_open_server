# ADR-0008: Compact engineering archives and bounded legacy recovery

Status: accepted, 2026-09-09.

A real 15-IED format-1 autosave expanded to 107,558,416 bytes after rebuilding
models, exceeding the previous 100 MiB reopen bound. Saving derived signal tables
and repeated original XML made a successfully saved archive unreadable.

Format 2 stores original XML in a SHA-256 keyed source table once per document.
IED entries retain ID, selected name, filename and runtime endpoint. Derived
signals, reports, controls and native configuration are regenerated on restore.
Diagram, mappings, sources, scenarios and project identity remain authoritative.
XML text is retained unchanged; checksum failures and missing references reject.

New saves reject JSON above 100 MiB before touching the destination. Compressed
input remains bounded at 100 MiB. Legacy format-1 recovery accepts at most
256 MiB unpacked; format 2 retains the 100 MiB unpacked limit. This bounded
compatibility allowance recovers affected existing projects without disabling
resource limits. Startup does not rewrite an archive; the next save uses format 2.

Older application versions cannot open format 2. Keep exact pre-upgrade backups.
Model interpretation follows the installed importer, as it already did for format 1.
This change does not add unsupported SCL features or persist transient MMS values.
