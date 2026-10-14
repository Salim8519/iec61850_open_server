# Proposed specifications

All five roadmap specifications below are promoted (changes 0058-0062). This folder
stays for future proposed specifications.

A proposed specification fixes **what** a large feature must do before any code is
written, so that the implementation follows an agreed contract instead of being
improvised. Proposed rules describe planned behaviour, not delivered behaviour.
Nothing in this folder may be quoted as a capability of Studio.

## How a proposed specification is used

1. Read the whole proposed specification before starting the feature.
2. Resolve every item under **Open questions** with the owner first. Record each
   answer in the specification with the date.
3. Write the decision record named in the specification.
4. Write the enforcing tests named in the **Enforced by** column first. Then write
   the code until they pass.
5. **Promote** the specification: move it to `docs/specs/`, delete the
   *Proposed* banner, add it to the table in [the specifications README](../README.md),
   and run `python studio/tools/verify.py`. From then on `check_specs.py` enforces
   every rule.
6. A rule that turns out to be wrong is changed here **before** the code, with
   the reason written underneath it. Rules are never silently dropped while building.

Rule IDs in this folder are reserved. They keep their numbers when promoted. The
checker does not scan this folder, because the tests it names do not exist yet.

## Roadmap: station-level engineering features

Owner goal (2026-10-14): reach and exceed the station-level features of
commercial SCD-driven test tools. Studio does this by reading the whole-substation
SCD and checking the real station against it, with no SCD authoring.
Build order chosen by the owner: **A → C → B → D**.

| Order | Specification | Feature | Depends on |
| --- | --- | --- | --- |
| A | [SLD import](../sld-import.md) (promoted) | Build the single-line diagram from the SCD `Substation` section | Existing import, diagram contract |
| C | [Signal flow](../signal-flow.md) (promoted) | Show who publishes GOOSE/reports to whom, from the SCD | Existing GOOSE links |
| B | [Station health](../station-health.md) (promoted) | Check the live network and real IEDs against the SCD | Real IEDs, GOOSE, C for drill-down |
| D1 | [Sampled Values](../sampled-values.md) (promoted) | Listen to IEC 61850-9-2 streams and check them against the SCD | GOOSE capture path, B |
| D2 | [Test mode](../test-mode.md) (promoted) | Show and handle Mod/Beh/Sim and simulation-flagged traffic | B, D1 |

Decisions already taken by the owner for this roadmap:
- Busbars from the SCD are drawn with `JUNCTION` nodes. Retired kinds are **not**
  reintroduced, so the project format stays at 12. Transformers and other
  unsupported equipment are listed as *not drawn*, never silently dropped.
- Sampled Values are **listened to only**. Studio does not publish SV.
- Station-level features stay inside the four existing places (UI-01).
- Real IEDs remain read-only (ADR-0031). No feature in this roadmap adds a write or
  command to a real device.

## Rule ID prefixes reserved here

| Prefix | Specification |
| --- | --- |
| `SLD` | SLD import |
| `FLOW` | Signal flow |
| `HEALTH` | Station health |
| `SV` | Sampled Values |
| `TMODE` | Test mode |
