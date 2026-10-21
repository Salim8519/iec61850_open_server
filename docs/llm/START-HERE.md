# LLM start here

This is the stable entry point for incremental work. Treat repository evidence as
authoritative and user wording as intent; never invent protocol behavior to bridge
a gap between them.

## Read in this order

1. Repository-root `AGENTS.md` for non-negotiable engineering rules.
2. [Current handover](../HANDOVER.md) for the saved project, recent evidence and
   unresolved decisions.
3. [File ownership](../reference/FILE-MAP.md), then the relevant architecture and
   design documents linked from [the documentation index](../INDEX.md).
4. [Feature specifications](../specs/README.md) for any feature you touch. Rules
   there are contracts with enforcing tests; change a rule before its code.
5. [API reference](../reference/API.md) and [algorithms](../reference/ALGORITHMS.md)
   before changing a boundary or duplicating behavior.
6. Query [engineering history](../history/INDEX.md) by file, tag or text. Read the
   linked decision and change records for the affected behavior.

## Work loop

1. Inspect the working tree and preserve unrelated user changes.
2. State the observed evidence, assumption and intended compatibility effect.
3. Change the smallest owning layer. Route handlers validate/delegate; domain code
   stays pure; native code owns MMS state; the Go GOOSE engine (`studio/goose_engine`,
   ADR-0073) owns where GOOSE frames go; the browser only projects backend state.
4. Add or update tests in the changed layer. Protocol claims require an independent
   client or packet-level record in addition to UI tests.
5. Update an ADR for architecture, design, contract or dependency changes. Append a
   dated narrative under `docs/changes/` with evidence and limitations.
6. Append one engineering-ledger record. Never edit an earlier ledger line.
7. Run `python studio/tools/verify.py` plus focused tests. Do not build the packaged
   executable unless the user explicitly requests it.

## Traceability commands

```powershell
python studio/tools/history_log.py query --file studio/native/worker/controls/controls.cpp
python studio/tools/history_log.py query --tag controls --limit 20
python studio/tools/query_runtime_logs.py --kind control_applied --limit 20
python studio/tools/verify.py
```

Use `history_log.py append --help` for the append contract. Each record has intent,
reason, exact file/change pairs, expected results, evidence and tags. It has no
mutable status. Runtime and engineering JSONL files are evidence: do not rewrite,
truncate, reorder or delete them.

## Safety boundaries

- Keep customer SCL and packet captures local. Documentation may record a hash,
  frame number, exact mapped reference and bounded diagnostic facts.
- Do not log credentials, launch tokens or full customer models.
- Preserve original SCL/vendor extensions and exact references.
- Check for an active Studio run before rebuilding native binaries or changing
  backend code that triggers development reload.
- A compatibility workaround must be explicit, narrowly scoped, visible and
  reversible. Never label it standard IEC behavior.
