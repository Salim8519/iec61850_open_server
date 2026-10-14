# 0060: Station health check

Date: 2026-10-14. Roadmap feature B. Specification: [Station health](../../specs/station-health.md).
Decision: [ADR-0037](../../decisions/ADR-0037-station-health.md).

## Change

**Native (`studio_live`)**
- `go_id_mismatch` is now reported.
- Unknown publishers are listed with source MAC, VLAN, goID, confRev, stNum and the test
  flag. Hitting the cap emits `goose_unknown_cap`.

**Backend**
- `domain/health_reads.py` decides which references to read and names the modes.
- `domain/goose_consistency.py` checks GOOSE blocks on the device and on the wire, and
  lists unknown publishers.
- `domain/health_model_checks.py` covers the data model, datasets, report settings and
  mode.
- `domain/station_health.py` produces the verdicts; `domain/health_export.py` writes the
  CSV and HTML.
- `acquisition/health_checks.py` and `acquisition/health_snapshot.py` run the reads on
  the existing session. `real_session.py` reads health at connect and the mode every
  10 s.
- `api/health.py` provides `GET /health`, `POST /health/refresh`, `GET /health.csv` and
  `GET /health/report.html`.
- The report dictionaries gain `conf_rev`.

**Frontend**
- **Devices → Whole station → Health check:** the station verdict and counts, then one
  row per IED (worst first) that expands into its checks with the reason and when each
  was seen. It offers Run check again, CSV, Report, and the unknown GOOSE table.
- New wording for `go_id_mismatch`.
- Signal-flow links show the latest wire verdict.

## Evidence

- pytest unit and integration: 227 passing.
- Protocol suite, run with the new build (`STUDIO_NATIVE_BIN=studio/native/build/station/Release`)
  and `STUDIO_GOOSE_ADAPTER` set to the VMware adapter: 32 passing. This includes:
  - `test_health_uses_existing_session_only`;
  - `test_go_id_mismatch_is_reported`, a real wire frame from `studio_ied` with goID
    `events` checked against an expected value of `not-events`.
- Frontend tests: 66 passing.
- **Browser** (isolated instance, IED1_XCBR set as a real device against a separate
  stand-in on 127.50.0.240:18140):
  - Station verdict *Check this*: 1 warning and 6 not checked.
  - IED1_XCBR: the connection, data points, datasets and report settings (EventsRCB
    dataset and confRev) match.
  - The GOOSE block is switched off on the device; the stand-in doesn't publish in
    loopback mode.
  - `Beh` = 0 is shown as an invalid mode.
  - GOOSE on the network is *Not checked*, with the reason that it needs the lab network.
  - Run check again, the CSV (19 rows, same as the view) and the HTML report all work.
  - After Stop, every IED reads *Not checked: Press Run*.

## Limitations

- Evidence comes from a Studio stand-in only; no third-party relay was used.
- The normal `studio/native/build/Release` binaries weren't replaced while the owner's
  dev session was running. They need the rebuild in `studio/native/README.md`.
- Simulated IEDs report no model checks.

**Observed, unrelated to this change**
- Simulated IEDs whose SCD enables reports emit frequent `report_control` (report
  created) events.
- Real-IED polling emits a `signal` event per poll even when nothing changed.
- Both fill the Activity list.
