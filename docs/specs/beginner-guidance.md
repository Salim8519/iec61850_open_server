# Spec: Beginner guidance

Decision: [ADR-0028](../decisions/ADR-0028-beginner-guidance-and-network-tabs.md).
Owner policy (2026-10-13): everything must be beginner friendly, look good and show
hints everywhere.

## Purpose

People who do not know IEC 61850 must be able to use every screen without a manual.
Jargon is explained where it appears (a small "?"), screens carry one short subtitle,
and every status says in words what happened and what to do next. After the owner
review of 2026-10-14 hints are not placed under every field: they appear in empty
states, wizards and next to settings whose meaning is not obvious (ADR-0032).

## Behaviour contract

- Shared components only: `Hint`, `InfoTip`, `EmptyState`, `StepWizard`,
  `StatusPill`, `Disclosure`, `SegmentedControl`, `Menu` in `studio/frontend/src/ui/`.
- All wording lives in `studio/frontend/src/copy/`. Components never hard-code hint
  text, so the wording can be reviewed and tested in one place.
- A status is always text plus an icon plus a line pattern; colour is only extra.
- Every abbreviation shown (GOOSE, APPID, MAC, VLAN, RCB, GI, MMS, SCD, Npcap, ...)
  has a glossary entry reachable from an InfoTip.
- An error names the object, says what went wrong and how to fix it.

## Rules

| ID | Rule | Enforced by |
| --- | --- | --- |
| GUIDE-01 | Every hint key used by the guided features exists and is non-empty. | `studio/tests/frontend/guideCopy.test.ts::every referenced hint exists` |
| GUIDE-02 | Every glossary term has a plain one-sentence explanation. | `studio/tests/frontend/guideCopy.test.ts::glossary entries are plain sentences` |
| GUIDE-03 | Every status tone maps to a label, an icon and a line pattern. | `studio/tests/frontend/guideCopy.test.ts::status tones never rely on colour` |
| GUIDE-04 | Network conflicts and live-access problems map to a fix hint. | `studio/tests/frontend/guideCopy.test.ts::every problem code has a fix hint` |
| GUIDE-05 | Guided feature components take wording from the copy modules. | `studio/tests/frontend/guideCopy.test.ts::guided components do not hard-code hints` |
