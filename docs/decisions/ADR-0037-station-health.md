# ADR-0037: Station health check against the SCD

Accepted: 2026-10-14. Specification: [Station health](../specs/station-health.md).

Engineers need one answer to "does the real station match its SCD?". Studio already
compared each real IED's model on connect, and listened passively to GOOSE. It didn't
compare report- or GOOSE-block settings, mode or goID, and it hid most details of
unknown publishers.

**Decision**
- **No new connections or writes:**
  - Health uses the running live sessions and the GOOSE listener.
  - New reads go through the existing read-only `read` operation (GO attributes,
    `LLN0.Mod/Beh`, `LPHD.Sim`) and through the `rcb_status` answers the session
    already has.
  - The `studio_live` operation table is unchanged (LIVE-01).
- **Native listener:**
  - Expected goID is now checked and reported as `go_id_mismatch`.
  - Unknown publishers carry source MAC, VLAN, goID, confRev, stNum and the test flag.
  - Reaching the 500-entry cap emits `goose_unknown_cap`.
- **Verdict logic:**
  - Pure domain modules (`station_health`, `health_model_checks`, `goose_consistency`,
    `health_reads`) produce OK / Check / Problem / Not checked.
  - Not checked always carries a reason and ranks above OK, so it can never pass as OK.
- **Exports:** CSV and an escaped, self-contained HTML snapshot. Signal-flow GOOSE links
  show the latest observed wire verdict.

**Compatibility**
- **Additive:** report entries gain `conf_rev`, and GOOSE watches gain `received_*` and
  `observed_at`.
- **Native rebuild needed:** the native programs must be rebuilt for the goID check and
  the richer unknown publishers. Older `studio_live` builds still work, without those
  two details.
- No project-format change.
