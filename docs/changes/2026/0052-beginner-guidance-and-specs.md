# 0052: Beginner guidance kit and protected feature specifications

Date: 2026-10-14. Owner requests: "completely beginner friendly, nice design, hints
everywhere" and "add a spec file for each big feature to protect against LLM
changes or hallucinations". Decision: [ADR-0028](../../decisions/ADR-0028-beginner-guidance-and-network-tabs.md).

Change:
- `frontend/src/components/guide/` (Hint, InfoTip, EmptyState, StepWizard,
  StatusPill) and `frontend/src/copy/` (hints, glossary, status, compare).
- Tabs: Connections → Network; new Real IEDs.
- `docs/specs/` with numbered rules for lab network, GOOSE, real IEDs and guidance;
  `studio/tools/check_specs.py` added to `verify.py`.

Evidence: frontend contract tests 58 pass (new `guideCopy`, `gooseStatus`; the
problem-code test reads codes from backend and native source); `npm run check` and
`npm run build` pass; spec check passes with every rule mapped to an existing test.
Browser review fixed a card-squashing layout bug, a misleading GOOSE checkbox in
loopback mode, a duplicated status pill and misaligned connection inputs.

Limitations: owner visual acceptance pending. Hints are English only.
