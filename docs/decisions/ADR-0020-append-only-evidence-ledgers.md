# ADR-0020: Append-only engineering and runtime evidence

Accepted: 2026-09-16.

Future runtime event history and engineering change history are append-only. No app
path may rewrite or delete an earlier evidence record. This replaces automatic run
log pruning and mutable run manifests.

Each runtime run writes compact `events-NNNN.jsonl` segments, a compact append-only
`manifest.jsonl`, and start/close entries in root `runs.jsonl`. Segment-open records
make file discovery indexable without rewriting a file list. Normal close flushes
the active segment before appending close facts. Existing legacy `manifest.json`
folders remain readable and unchanged. Storage is intentionally unbounded; storage
export/archival, if later required, must be explicit and preserve verifiable copies.

Engineering history uses compact `docs/history/ledger-NNNN.jsonl` segments with at
most 200 entries. Each entry records user intent, engineering reason, exact file
changes, expected results, evidence and tags. It has no status field. A SHA-256 hash
chain detects content edits or reordering; repository verification also rejects
tracked line removal/replacement. Corrections and superseding evidence are new
records that refer to earlier IDs.

The JSONL ledgers are primary immutable indexes. Markdown ADRs and change records
remain human explanations and may add clarifications, but cannot replace ledger
evidence. Query tools read without modifying either log family.

Compatibility impact: readers that expect runtime `manifest.json` must support the
new append-only `manifest.jsonl` for new runs. No public HTTP API or project archive
format changes.
