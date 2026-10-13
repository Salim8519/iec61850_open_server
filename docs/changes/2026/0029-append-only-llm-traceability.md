# 0029: Append-only LLM traceability and repository map

Date: 2026-09-16. Implements
[ADR-0020](../../decisions/ADR-0020-append-only-evidence-ledgers.md).

The repository now has one LLM entry path, a current handover, an HTTP/event API
catalogue, core algorithm invariants and a responsibility-based file map. Root
engineering rules require agents to read that route, preserve unrelated work,
record assumptions/evidence and append an immutable change-ledger entry.

Engineering history uses compact SHA-256 chained JSONL segments with 200 entries
maximum. Each record has intent, reason, exact file/change pairs, expected results,
evidence and tags; there is no status field. The append/query tool supports ID,
tag, file and text retrieval. Verification checks schema, hashes, chain/segment
continuity and tracked Git line removals.

Future runtime runs use compact append-only event segments, per-run lifecycle
journals and a root run index. Run start, segment-open and close are separate facts;
no manifest is rewritten and automatic evidence pruning is removed. Existing run
folders and legacy manifests are not changed. A read-only query tool filters by run,
kind, IED, reference and text, and can retrieve root or per-run lifecycle indexes.

Files are grouped by reason in the engineering ledger entry
`20260916-append-only-llm-traceability`. Focused tests cover lifecycle appends,
rotation without deletion, hash-chain rotation and tamper detection. Repository
verification passed all four checks. The complete maintained Python/native suite
passed 126 tests, all 9 frontend contract tests passed, and the production frontend
build passed. No packaged executable was built.

Limitation: append-only runtime history intentionally has unbounded local storage.
A future archive/export policy must preserve verifiable records and cannot silently
restore automatic deletion.
