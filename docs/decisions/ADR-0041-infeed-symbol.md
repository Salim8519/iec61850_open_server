# ADR-0041: Infeed symbol, isolated from every engine

Accepted: 2026-10-14. Owner request: a new symbol for where power enters, which "must
not corrupt any engine, especially the SCD engines". Specification: [Live colouring](../specs/energization.md).

**Decision**
- A new drawing-only equipment kind `INFEED` (incoming line, power transformer or
  external grid) with `infeed_type` and `infeed_state` (`live`/`dead`).
- Isolation, each guarded by a test (INFEED-01..04):
  - It rejects state mappings and commands, so no runtime path can reach it.
  - The SCL exporter and model builder read only IED signals, so export stays
    byte-identical.
  - SLD import never produces it.
- The new fields default to `None` and are dropped from the stored form of every other
  kind. Projects without Infeeds are stored byte for byte as before, so the format
  stays 12 with no migration and no recovery archive.

**Compatibility:** additive. A project that contains an Infeed can't be opened by an
older build: its validation rejects the unknown kind, and startup recovery keeps the
file. Every other project is unaffected.
