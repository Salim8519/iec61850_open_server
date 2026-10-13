# ADR-0028: Beginner guidance kit, Network and Real IEDs tabs, feature specs

Tab list and "hint under every field" superseded by [ADR-0032](ADR-0032-clean-four-place-ui.md).

Accepted: 2026-10-14. Owner request: everything must be beginner friendly, look
good and show hints everywhere; add specification files for each big feature so
later LLM changes cannot drop or invent behaviour.

- Shared guidance components live in `frontend/src/components/guide/`: `Hint`,
  `InfoTip`, `EmptyState`, `StepWizard`, `StatusPill`. Features never build their own.
- All wording lives in `frontend/src/copy/` (`hints.ts`, `glossary.ts`,
  `status.ts`, `compare.ts`). `Hint` takes only an `id`; contract tests reject
  hard-coded hint text in guided components.
- A status is words + icon + line pattern (solid, dotted, dashed, double); colour
  only repeats it. Every backend/native status and problem code has a plain fix
  hint; the contract test extracts those codes from the backend and native source.
- The fixed tab list changes: **Connections becomes Network** (mode banner, lab
  wizard, endpoints, GOOSE traffic, clean-up) and **Real IEDs** is added (role per
  IED, connection, SCD comparison, live values). See `design/layout.md`.
- Feature specifications live in `docs/specs/`. Each rule has a stable ID and names
  its enforcing tests; `studio/tools/check_specs.py` (part of `verify.py`) fails
  when a rule has no test or names a test that does not exist.

Compatibility: no project-data change from this ADR. Users find the old
Connections content under Network. The 1366 px layout keeps all tabs on one row.
