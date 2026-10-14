# 0057: Proposed specifications for station-level features

Date: 2026-10-14. Owner request: plan station-level features that reach and exceed
commercial SCD-driven test tools. Each large feature gets its own specification
before any code is written, to avoid improvised implementation.

Change (documentation only, no code):
- New [proposed specifications](../../specs/proposed/README.md) folder. It holds the
  roadmap, the owner decisions and the promotion procedure:
  resolve open questions → write the ADR → write the tests → write the code → move the
  spec into `docs/specs/`.
- Five proposed specs with reserved rule IDs and the tests that will enforce them:
  - [SLD import](../../specs/sld-import.md) (`SLD-01..14`; promoted in 0058)
  - [Signal flow](../../specs/signal-flow.md) (`FLOW-01..10`; promoted in 0059)
  - [Station health](../../specs/station-health.md) (`HEALTH-01..12`; promoted in 0060)
  - [Sampled Values](../../specs/sampled-values.md) (`SV-01..11`; promoted in 0061)
  - [Test mode](../../specs/test-mode.md) (`TMODE-01..08`; promoted in 0062)
- The specs README explains proposed specs. Known issues links to them.

Owner decisions recorded (2026-10-14):
- Build order: SLD import → signal flow → station health → SV/test mode.
- Busbars are drawn as `JUNCTION`. No retired kind returns, and the project format
  stays at 12.
- Sampled Values are listened to only, never published.

Why the specs live outside `docs/specs/`: `check_specs.py` requires every rule to name
an existing test, and the specs README says planned behaviour is not a specification.
Proposed specs are therefore kept in `docs/specs/proposed/`, which the checker doesn't
scan. No checker or architecture change was made, so no decision record is needed.
Each feature writes its own ADR when it is built.

Facts the specs are based on (read from the code on 2026-10-14):
- Nothing reads `Substation` or `LNode` today.
- `scd/model_substation.scd` carries `esld` layout coordinates, and
  `scd/open_substation.scd` carries `LNode` and `SampledValueControl`.
- `ClientLN` is not parsed.
- SV is compiled off (`goose_options.cmake`).
- Expected goID is stored but not checked.
- Mod/Beh/Sim are not read.

Evidence: `python studio/tools/verify.py` passes. Every spec file is under 300 lines.

Limitations: these are plans only. None of these features exist, and none may be
described as a Studio capability until its spec is promoted with evidence.
