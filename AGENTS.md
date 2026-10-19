# Substation Studio engineering rules
-the app should be always production ready, user hates diffreneces between dev and production, howeever dont make exe, while the app should always mathc prodcution. exe make only when user asks
## Product contract

- Build the portable Windows application in `studio/`; preserve the existing demo.
- The runtime is Flask + Waitress, React/TypeScript, native libiec61850 workers and the Go
  GOOSE engine `studio_goose.exe` (ADR-0073).
- Start at `docs/llm/START-HERE.md`, then read the linked architecture/design
  documents relevant to the change.
- Keep the interface offline, English, and usable without development tools.
- Do not claim protocol support or interoperability without recorded evidence.
- Development and the packaged application must have identical product behavior,
  defaults, project data, and native protocol semantics. Development tooling may
  add reload/diagnostics only; users must not change parameters between modes.
- Avoid stale-build/cache surprises. A launcher must identify an existing session
  before reusing it and must not silently run incompatible backend/native code.
- Do not build or promote the packaged EXE unless the user explicitly requests it.

## File and responsibility boundaries

- Maintained source, tests, scripts, config, and documentation must be **under 300
  physical lines** (maximum 299), including comments and blank lines.
- Split by responsibility before exceeding the limit. No compressed formatting,
  arbitrary part-number modules, catch-all helpers, or generated-code disguises.
- Imported engineering data, original third-party source/licenses, generated
  outputs/lockfiles, runtime logs, and untouched legacy demo files are excluded.
- Exclusions are explicit in `studio/tools/check_file_lengths.py`.
- One responsibility per file. Route handlers delegate to application services.
- Domain modules must not import Flask, persistence, networking, or worker code.
- Native workers own wire protocol state; the coordinator owns simulation sources.
- Frontend components display state and send commands; never duplicate authoritative
  simulation logic in the browser.
- Avoid circular imports and side effects at module import time.

## Fixed design

- Follow `docs/design/layout.md` and `docs/design/interactions.md`.
- Use shared design tokens and existing controls; no independent visual systems.
- Display quality, switching position, selection, and energized state distinctly.
- Never rely on colour alone for equipment state or failures.
- Changes to architecture, design, contracts, or dependencies require a decision
  record explaining the need and compatibility impact.

## Engineering data and network safety

- Retain original SCL and vendor extensions. Never silently discard unsupported data.
- Preserve exact IED references; runtime IP overrides are separate from SCL addresses.
- Invalid or unsupported input produces actionable diagnostics, not invented values.
- Only the network helper changes adapter settings after the app's explicit preview.
- Journal app-owned aliases/firewall rules; cleanup must preserve pre-existing state.
- Never put credentials, tokens, or full customer models into diagnostic logs.
- Preserve user changes; never reset the worktree or replace unrelated files.

## Verification and traceability

- Run `python studio/tools/verify.py` and the tests appropriate to the changed layer.
- Verify real MMS behaviour with an independent client, separately from UI tests.
- Record changes under `docs/changes/` and update its index with evidence/limitations.
- Append one compact machine-readable entry with `studio/tools/history_log.py` for
  every completed change or evidence-only diagnosis. Never edit, reorder, rehash,
  or delete an existing `docs/history/ledger-*.jsonl` line. Ledger entries contain
  intent, reason, changed files, expected results, evidence and tags; no status field.
- Keep decisions, test evidence, and known issues linked from `docs/INDEX.md`.
- Split growing indexes by date before the file limit; do not overwrite history.
- Runtime logs use run/IED/correlation identifiers and distinguish request, applied
  state, and externally observed results.
- Runtime event segments and lifecycle indexes are append-only. Do not rewrite or
  delete app-owned log history. New indexes must be derived or append-only.
- Never mark the project goal complete until the release and acceptance evidence exist.
