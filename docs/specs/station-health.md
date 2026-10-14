# Spec: Station health (live vs SCD)

Decision: [ADR-0037](../decisions/ADR-0037-station-health.md). Change: [0060](../changes/2026/0060-station-health.md).
It extends [Real IEDs](real-ieds.md) and [GOOSE](goose.md) without changing
their rules. Owner policy carried over (2026-10-13): real IEDs are read-only, only free
report blocks are used, polling is the fallback, and no commands are ever sent.

## Purpose

One screen answers the question "does the real station match its SCD?". Each IED
gets a verdict: is it reachable, does its data model, datasets and report blocks
match, is its GOOSE on the wire as engineered, and which mode is it in. Every
difference has a plain explanation and a fix hint for the engineer.

## Checks

| Area | What is compared | Source of live data |
| --- | --- | --- |
| Reachability | MMS association succeeds | Existing live session |
| Data model | Signals, types and FCs (existing `compare()`) | `discover` |
| Datasets | Members (existing) | `discover` |
| Report blocks | Block exists; `DatSet` and `ConfRev` equal SCD | `rcb_status` (read only) |
| GOOSE blocks on device | `GoEna`, `DatSet`, `GoID`, `ConfRev` vs SCD | MMS read of GO attributes (read only) |
| GOOSE on the wire | Seen, timeout, confRev, dataset, goID, test flag | Passive Npcap capture |
| Unknown GOOSE | Publishers not in the SCD, with APPID, MAC, VLAN, goID, confRev, stNum | Passive capture |
| Mode | `LLN0.Mod`, `LLN0.Beh` per logical device, `LPHD.Sim` | MMS read (read only) |

## Behaviour contract

**Verdicts**
- Every check result is one of **OK**, **Warning**, **Problem** or **Not checked**.
- *Not checked* always carries its reason, for example "Npcap not installed", "IED set
  to simulated" or "not connected yet". It is never shown as OK.
- The IED verdict is its worst check result. The station verdict is the worst IED
  verdict.
- Every result carries the time it was observed and its source (`scd`, `mms_read`,
  `report`, `wire`). Mode results older than 30 s (three refresh periods) are marked stale.

**Data access**
- The health check reuses the running live sessions and the GOOSE listener. It
  doesn't open extra associations or reserve extra report blocks, and adds no writes.
- GO attribute reads and mode reads use the existing read-only `read` operation;
  report-block settings come from the `rcb_status` read the session already makes.
- Report-block settings, GOOSE-block settings and the model are read at connect and
  again on **Run check again**. Mode is re-read every 10 s.
- With `polling` access Studio never reads report blocks, so their settings read
  *Not checked* with that reason.
- The GOOSE listener is extended:
  - It checks the expected goID; a mismatch is reported as `go_id_mismatch`.
  - It enriches *unknown* publishers with confRev, goID, stNum, source MAC and VLAN.
    It keeps the existing 500-entry cap and says so when the cap is reached.

**Simulated IEDs**
- These show only wire checks, as evidence that Studio's own publishers are on the
  network.
- Model checks read *Not checked: simulated by Studio*.

**Output**
- The station view lives in the Devices place (Devices → Health), with a summary first
  and per-IED rows.
- A row expands into its checks, worst first. Details live in the IED's *Compare with
  SCD* tab, and GOOSE links in *Signal flow* show the latest wire verdict and when it
  was observed.
- Fix hints use plain language, for example: "The relay sends GOOSE with confRev 3 but
  the SCD says 2. Load the latest CID into the relay or get the current SCD."
- Exports:
  - **CSV:** one row per check.
  - **Snapshot report:** a self-contained, escaped HTML file with the station, time
    and run identifier.
- Customer models and values never go into diagnostic logs (AGENTS rule). Exports are
  written only to a location the user chooses.

## Out of scope

- Fixing differences on the device.
- Writing CIDs or settings.
- Time-synchronisation accuracy analysis.
- Cybersecurity scanning.
- Network topology discovery (LLDP/SNMP).

## Rules

| ID | Rule | Enforced by |
| --- | --- | --- |
| HEALTH-01 | The health check adds no write operation; `studio_live` stays read-only apart from the report-block reservation allowed by LIVE-01. | `studio/tests/unit/test_live_contract.py::test_health_reads_are_read_only` |
| HEALTH-02 | *Not checked* always has a reason and is never counted as OK. | `studio/tests/unit/test_station_health.py::test_not_checked_is_never_ok` |
| HEALTH-03 | The IED verdict is its worst check; the station verdict is the worst IED. | `studio/tests/unit/test_station_health.py::test_verdict_is_worst_result` |
| HEALTH-04 | A report block whose `DatSet` or `ConfRev` differs from the SCD is a problem naming both values. | `studio/tests/unit/test_model_compare.py::test_report_block_settings_are_compared` |
| HEALTH-05 | A GoCB on the device whose `GoID`, `DatSet` or `ConfRev` differs from the SCD is a problem naming both values. | `studio/tests/unit/test_goose_consistency.py::test_device_gocb_attributes_are_compared` |
| HEALTH-06 | GOOSE on the wire with an unexpected goID is reported as `go_id_mismatch`. | `studio/tests/protocol/test_goose_consistency_wire.py::test_go_id_mismatch_is_reported` |
| HEALTH-07 | Unknown publishers carry APPID, MAC, VLAN, goID, confRev and stNum; the cap is stated when reached. | `studio/tests/unit/test_goose_consistency.py::test_unknown_publisher_details_and_cap` |
| HEALTH-08 | Every result has an observation time and source, and is marked stale after its supervision time. | `studio/tests/unit/test_station_health.py::test_results_carry_time_source_and_staleness` |
| HEALTH-09 | The health check opens no extra association and reserves no extra report block. | `studio/tests/protocol/test_live_client.py::test_health_uses_existing_session_only` |
| HEALTH-10 | Every verdict and problem has a plain label, an icon and a fix hint. | `studio/tests/frontend/stationHealth.test.ts::every health result has text icon and hint` |
| HEALTH-11 | The CSV export contains exactly the rows shown, with time and source. | `studio/tests/integration/test_station_health_api.py::test_csv_matches_results` |
| HEALTH-12 | Diagnostic logs never contain signal values or model content from the check. | `studio/tests/unit/test_station_health.py::test_logs_exclude_customer_data` |

## Acceptance evidence

Recorded in change [0060](../changes/2026/0060-station-health.md):
- Protocol tests against the stand-in `studio_ied` (`STUDIO_NATIVE_BIN` set to the new
  build): health reads go only through `read` on the existing association. Mode values
  are integers, and the report block is readable and free.
- A GOOSE wire test on the VMware lab adapter: a listener that expects another goID
  reports `go_id_mismatch` with both values. The ordinary wire test still passes.
- Browser, isolated instance: IED1_XCBR as a real device against a separate stand-in.
  Connection, model, datasets and report settings match. The GOOSE block is switched off
  on the device, and Beh = 0 (stand-in) is shown as an invalid mode. GOOSE on the wire is
  *Not checked: needs the lab network*. The CSV has the same 19 rows as the view.
- Pending: deliberately mismatched stand-in models, real third-party relays, and the
  owner's acceptance. Interoperability is not claimed.

## Answered questions

1. Wire and mode checks run continuously; model, report-block and GOOSE-block settings
   are read at connect and on **Run check again**. (Owner delegated the proposed
   answers, 2026-10-14.)
