# Append-only engineering history

`ledger-NNNN.jsonl` is the compact machine-readable change/evidence index. Segments
contain at most 200 one-line records so maintained files remain below 300 physical
lines. Records are SHA-256 chained across segments.

Each record contains:

- `id`, immutable timestamp `at`, user `intent` and engineering `reason`;
- exact `files` as path/change pairs;
- `expected` results, observed `evidence`, and retrieval `tags`;
- `prev` and `hash` for chain verification.

There is deliberately no status field. Later discoveries append a new record that
references the earlier ID in its reason/evidence; they never rewrite history.

## Append and query

```powershell
python studio/tools/history_log.py append --id 20260916-example `
  --intent "Describe the user outcome" --reason "Describe why" `
  --file "path/to/file=Describe the change" `
  --expected "Describe a testable result" --evidence "test or artifact" `
  --tag area

python studio/tools/history_log.py query --tag controls --limit 20
python studio/tools/history_log.py query --file controls.cpp
python studio/tools/history_log.py query --text "SBOw(false)"
python studio/tools/history_log.py verify
```

Only the append command may write ledger files. Never edit, reorder, truncate,
delete or regenerate existing lines. `studio/tools/verify.py` checks schema, hashes,
segment sequence and chain continuity. In a tracked worktree it also rejects Git
diffs that remove or replace ledger lines.

Runtime evidence is separate under `studio/data/logs/<run-id>/`. New runs use
append-only `events-NNNN.jsonl` and `manifest.jsonl`; `studio/data/logs/runs.jsonl`
is the root lifecycle index. Query without mutation:

```powershell
python studio/tools/query_runtime_logs.py --run <run-id> --kind control_applied
python studio/tools/query_runtime_logs.py --ref C03_Y1105System/Bkr0XCBR1.Pos
python studio/tools/query_runtime_logs.py --lifecycle --run <run-id>
```

Legacy `manifest.json` files remain historical evidence. Do not migrate or delete
them in place.
