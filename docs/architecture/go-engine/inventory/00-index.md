# Go backend parity inventory: index

Read-only inventory of the Python backend `studio/backend/studio_app` (Flask 3.1.2 + Waitress
3.0.2, 206 files) as of 2026-10-19 (`master` at 3ccd473, with uncommitted changes 0110/0111 in
the working tree). Purpose: the checklist a Go rewrite must satisfy so users notice nothing.

## Files
| File | Content |
| --- | --- |
| http-conventions.md | middleware order, security/session, JSON encoder, error mapping, SSE format, static/downloads |
| api-routes-1.md | session, project, runtime, SCL/IEDs, signals, equipment, scenarios, test panel, traces, test mode |
| api-routes-2.md | network, IED settings, real IEDs, GOOSE/SV/MMS access, GOOSE sharing, SLD, signal flow, health, license, preferences, updates |
| api-routes-3.md | Tools: Explorer, Find IEDs, tool memory, activity, gateway, SCADA master/converter |
| frontend-callers-1.md, -2.md | every URL the frontend calls, file:line, fields read, polling intervals |
| frontend-events.md | SSE connection logic, event kinds the UI handles, parity-sensitive frontend assumptions |
| events.md | every bus event kind: fields, producer file:line, run-log rule, ordering |
| persistence-1..4.md | data folder, lock, launcher.json, autosave format 15, migrations, recovery, prefs, tool memory, update state, license files, journal, run logs |
| processes-1..3.md | studio_ied, studio_live, studio_goose, studio_probe, studio_control, studio_gateway, studio_master, network helper, PowerShell/Win32, launcher OS hooks |
| limits.md | limits, defaults, timeouts, cadences with file:line |
| behaviours.md | launch, single instance, dev session reuse, capability checks, license gate, update check, dev vs packaged, verbatim texts |

## Counts
- HTTP routes: 125 `/api/v1` rules (24 blueprints) + `/` + `/assets/<path>` = 127
  (Flask adds HEAD/OPTIONS automatically). 4 rules are not called by the frontend (tests,
  self-test and tools use them; keep them): `POST /scl/validate`, `GET /ieds/{id}/removal-preview`,
  `DELETE /ieds/{id}` (used by launcher/self_test.py), `POST /controls/operate`. `GET /sv`,
  `PUT /ieds/{id}/test-mode`, `/health*`, `/signal-flow*`, `/project/substation-sources` and
  `/project/diagram/from-scl/preview` exist but are missing from docs/reference/API.md.
- Frontend: about 121 URL+method uses, 24 polled paths (300 ms .. 30 min).
- Event kinds: 51 backend-originated (incl. synthetic `resync`) + 10 forwarded from
  studio_ied (`control_requested/applied/selection/test`, `connection`, `report_control`,
  `goose_control`, `goose_publishing`, `native_diagnostic`, `worker_failed`) = 61.
- Disk artefacts: about 30 (persistence-*.md), plus 10 download-only outputs.
- Processes: 7 native/Go executables + the elevated helper (same exe, `--network-helper`)
  + PowerShell/IP Helper queries + the Edge/Chrome app window.
- Limits/defaults/timings: 89 table rows in limits.md (about 150 individual values).

## Top parity risks (merged from all files)
1. JSON dialects: `jsonify` = sorted keys, compact, ASCII-escaped (incl. U+007F), trailing `\n`,
   Python float repr (`1.0`, `1e-05`), NaN allowed; SSE events and files = insertion order with
   `", "`/`": "`. Go `encoding/json` differs on every point (HTML escaping too).
2. SCL import/validation (lxml XSD, defusedxml, model builder) must be byte-for-byte equivalent;
   restore re-imports every stored SCL, so any difference turns autosaves "unreadable".
3. Pydantic lax coercion, `model_dump` defaults and error `details` for diagrams.
4. Hash-derived identities from Python strings: firewall rule names, fingerprints, portable
   worker key, named event, SCL source ids, signal-flow/link-map/SLD node ids.
5. Error-type to status mapping (StudioError 400/409, ValueError/KeyError 400 with Python text,
   RuntimeError/Timeout 503, other 500) and that every response, even 404/405/500, is JSON.
6. Middleware order: loopback/origin 403 -> 401 -> 402 license -> 503 startup gate -> routing.
7. Cross-version contracts: `studio.lock` byte lock, `launcher.json`, show event, license files,
   `STUDIO_LICENSE_CERT_PATH`, and the signed-module check workers do on the Python `.pyd`.
8. CRLF text files on Windows and Python ISO timestamps (`+00:00`) read back by the time anchor.
9. Native IPC: number typing (int vs float vs bool), event-before-reply ordering on one reader,
   per-call timeouts, capability checks with exact refusal texts.
10. Python-specific behaviour in logic: `isinstance(True, int)`, `casefold`, `random.Random`
    noise generator, cadence arithmetic, `int()` parsing of query strings, message formatting.
