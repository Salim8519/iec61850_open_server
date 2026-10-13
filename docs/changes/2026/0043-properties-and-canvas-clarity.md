# 0043: Properties cards, setup checklist and canvas clarity

Date: 2026-09-29. Decision: ADR-0025.

Audit (headless screenshots of every object on an isolated project copy):
mapped-signal boxes showed only "Change ▸ Exact path" because the name
collapsed to zero width; headers showed codes (`PUSH_BUTTON`); sections were
separated only by thin rules with cryptic IED/MMS badges; actions sat among
fields; no object said whether it was fully set up; canvas cards were
identical white boxes whose states differed only in tiny grey/blue text; the
Operate card was last while running. Loading and clicking objects bumped the
project revision (13 -> 16) with an identical saved diagram.

Changes (frontend only): `kindInfo.ts` (names, categories, setup items, state
tone), `InspectorHeader.tsx`, `inspector.css`, card-style `PropertySection`,
reorganised `Inspector.tsx`, readable `StateMappingEditor`/`ControlMapping`
picks, single-IED preselection in pickers, `equipment-node.css`, category
tokens, and `sameEngineering` guarding no-op saves in `useStudio.ts`.

Evidence:

- New `kindInfo.test.ts` (5) and a `sameEngineering` test; frontend node
  tests: 35 pass. Backend unit + integration: 124 pass; `npm run check`,
  build and `verify.py` pass.
- After the guard, loading and selecting seven objects left the revision at
  16 (before: +3 per visit) on the isolated copy.
- Screenshots stopped and running (CB Operate first, solid Open pill, live
  source value, dashed Not mapped pills).

Limitations: backend and native untouched, so no new MMS evidence needed or
taken. IED, Signal Test, Scenarios and Connections tabs only inherit the
token changes. Owner visual acceptance pending.
