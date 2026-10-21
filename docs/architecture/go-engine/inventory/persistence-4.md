# Persistence inventory 4/4 - licensing, network, runtime, logs, downloads, JSON conventions

## 1. License home `%LOCALAPPDATA%\ScadaBench\` (`licensing/service.py`)

Home = `os.environ["LOCALAPPDATA"]` or `~/AppData/Local`, + `ScadaBench` (`:27-29`), in every
mode. Enforced only when frozen (`app_factory.py:36-38`); development never writes license
files but still reads `client_config.json` (`server_url()`, used by the update check).
Generic write `_write` (`:71-75`): `mkdir(home)`, `<name>.tmp` `write_text(utf-8)` (**CRLF**
where text has newlines), `os.replace`. Reads `_read_json` (`:65-69`): OSError/ValueError ->
default.

| File | Writer | Format | Reader / when missing or corrupt |
|---|---|---|---|
| `license.cert` | `_install` `:164`; refresh in `verify_online_if_due` `:244` | `json.dumps({"payload": <str>, "signature": <b64url str>}, indent=2)` -> CRLF, no newline. Input accepted: raw bundle or `{"certificate": {...}}` wrapper (`certificate.py:19-30`) | `_evaluate` `:117-124`; native workers via `STUDIO_LICENSE_CERT_PATH` (`native/common/license_guard.cpp:256-260`, nlohmann JSON, CRLF ok). Missing -> `NO_LICENSE` or stored `revoked_reason`; parse/verify fail -> CertificateError text ("This is not a license file...", "The license file is incomplete...", "The license file is damaged.", "The license file signature is unreadable.", "The license file is damaged or was changed. Activate again.", "This ScadaBench build has no license key configured...") |
| `license.cert.<revoked|moved>-<epoch>` | `os.replace(cert, ...)` `:236` | renamed old certificate | never read |
| `client_config.json` | `set_server_url` `:201-209` | `json.dumps(config, indent=2)` (CRLF); keeps unknown keys; `server_url` str (stripped, no trailing `/`, must start `https://`, `http://127.0.0.1`, `http://localhost`, or be "") | `server_url()` `:105-107`: configured or `https://license.scadabench.com` (`release_key.py:7`) |
| `license_state.json` | `_install` `:165-167`, `verify_online_if_due` `:247` | `json.dumps(state)` default separators, no indent/newline; keys `revoked_reason` (str|null), `last_verify` (int epoch) in prior/insertion order | `_evaluate` (`revoked_reason`), verify scheduling (every 7 days, lifetime plan only) |
| `.time_anchor` | `TimeAnchor.advance` (`time_anchor.py:43-52`) | ASCII decimal epoch seconds, no newline; temp `.time_anchor.tmp` (`with_suffix` on a dot-file appends); only moves forward; OSError ignored | `max_seen()`: anchor int (strip, `int()`, errors -> 0) max with newest `"at":"..."` ISO time in last 8 KiB of `<data>/logs/runs.jsonl`; native reads it with `atoll` (`license_guard.cpp:281-283`) |

- Status cache 60 s; anchor advanced on every licensed evaluation.
- Certificate payload (JSON inside `payload` string, `domain/license_policy.py:52-74`):
  `version` (1), `type` ("trial"|"lifetime"...), `hwid` (64 hex), `issued_at`, `expires_at`,
  `license_id`, `email`, `kid`. Signature: Ed25519 over the exact UTF-8 payload string,
  base64url without padding; `kid` = `sha256(pubkey)[:8]`.
- HWID v2 = `sha256("substation-studio-hwid-v2|" + MachineGuid.strip().lower() + "|" +
  "%08X" volume serial of %SystemDrive% + "|" + SMBIOS type-1 UUID raw 16 bytes hex)`;
  legacy v1 without the UUID accepted for certificates issued before 1814400000
  (`licensing/hwid.py`). Must equal native `compose_hwid()`.
- Tamper check: `integrity.EXPECTED` (filled by `compile_protected.py`) sha256 of shipped
  workers + debugger check -> "ScadaBench program files were changed or damaged. Reinstall
  ScadaBench." / "ScadaBench cannot run while a debugger is attached."
- API errors: `{"error", "code"}`; 503 for offline/no_server/server_error, 409 for
  bound_elsewhere/transfer_limit/trial_used, 400 otherwise (`api/license.py:16-24`).

## 2. Network journal and helper requests (`networking/`)

| File | Writer | Format | Reader |
|---|---|---|---|
| `<data>/network/journal.jsonl` | `journal.py:20-29` (append under lock, `mkdir(parents)`) | one record per line: `json.dumps(record, sort_keys=True)` default separators `", "`/`": "`, ensure_ascii, + `"\n"` in text mode => **CRLF** lines. Keys: `at` (`datetime.now(utc).isoformat()`), `event` (applied/removed/failed/skipped), `fingerprint`, `item`, `result` (str[:500]) | `records()` `read_text().splitlines()`, `json.loads` each non-blank line (a corrupt line raises -> request error); `outstanding()` replays owned items: applied -> add, removed -> drop, keyed `alias:<guid>:<ip>`, `unblock:<name>`, `firewall:<name>` |
| `<data>/network/requests/<apply|remove>-<uuid32>.json` | `service.py:149-155` | bytes `json.dumps({"action","fingerprint","items"})` default separators, insertion order, items without `description`; never deleted | elevated helper (`<app exe> --network-helper <request> <sha256 hex of bytes> <worker exe>`, `entrypoint.py:21-23`, `helper/runner.py:29-39`) verifies hash then validates every item |
| `<data>/network/requests/<...>.result.json` | helper `runner.py:38` | `json.dumps({"refused":bool,"results":[{key,ok,message[:500]}]})` or with `reason`; `write_text` (no newlines) | `service.py:159-171`; missing -> "The network helper finished without a result; nothing was recorded as changed"; refused -> "The network helper refused the request: <reason>" |

Journal item shapes (`change_plan.py:24-49`): alias `{kind,adapter_guid,adapter_index,
adapter_name,ip,prefix,for,owned,action}`; firewall `{kind,name,group:"Substation Studio",
addresses(sorted),ports(sorted),program,owned,action}`; gateway_firewall `{kind,name,
group:"ScadaBench",program,owned,action}`; unblock `{kind,name,display,program,owned,action}`.
Rule names: `SubstationStudio-MMS-` + `sha256(json.dumps([sorted(addrs), ports, str(program)],
sort_keys=True))[:12]`; `ScadaBench-Gateway-` + `sha256(str(program).lower())[:12]`. These
names identify pre-existing Windows rules ("keep" vs "add") and cleanup targets.

## 3. Runtime worker configs `<data>/runtime/<ied_id>.cfg`

`runtime/supervisor.py:54-56`: `write_text(ied["cfg"], encoding="utf-8")` on every IED start
(overwrite, not atomic, never deleted). `cfg` = libiec61850 model config from
`ModelBuilder` (`"\n".join(lines) + "\n"`, `scl/importer.py:103`) => **CRLF** on disk.
Absolute resolved path is sent to the native worker in the start request.

## 4. Runtime logs (`observability/log_writer.py`, `simulation/coordinator.py:132`)

A `RunLog(<data>/logs)` is created at every Run start and closed at Stop/failed start.

| File | Lines | Notes |
|---|---|---|
| `logs/runs.jsonl` | `run_started` and `run_closed` only | lifecycle index across runs; append-only |
| `logs/<run_id>/manifest.jsonl` | `run_started`, `segment_opened` per segment, `run_closed` | `run_id` = uuid4 hex |
| `logs/<run_id>/events-NNNN.jsonl` | every bus event emitted while the run log is open | `NNNN` = `%04d` from 0000 |

- Encoding: `json.dumps(value, allow_nan=False, separators=(",", ":"))` + `"\n"`, opened with
  `newline="\n"` => **LF**, UTF-8, ensure_ascii, insertion order, append mode.
- Lifecycle records: `{"kind":"run_started","run_id","at","application":"0.1.0"}` (constant
  string, not the app version), `{"kind":"segment_opened","run_id","at","file":"events-0000.jsonl"}`,
  `{"kind":"run_closed","run_id","at"}`. `at` = `datetime.now(timezone.utc).isoformat()` ->
  `2026-10-14T09:54:34.341681+00:00` (microseconds omitted entirely when 0).
- Event line = `{"run_id": ..., **event}` where event = EventBus `{"id": seq int, "kind",
  "timestamp": time.time() float, **payload}` (`runtime/event_bus.py:13-19`); worker payload
  keys `kind/id/timestamp` are renamed `native_kind/native_id/source_timestamp`
  (`coordinator.py:80-81`). Example: `{"run_id":"...","id":4,"kind":"signal","timestamp":
  1791971674.8238513,"ied_id":"...","ref":"...","value":false,"quality":0,
  "source_timestamp":0,"source":"initial_native_snapshot","updates":{...}}`.
- Segment roll: when a segment is open and `size + len(record_utf8) > 10 MiB` the file is
  closed and the next index opened (first record of a segment is always written). Segment
  file and its `segment_opened` line are created lazily at the first event.
- Flush after 100 buffered `signal` events or immediately for any other kind; close flushes.
- `allow_nan=False`: a NaN/Inf value in an event raises ValueError inside `emit()`.
- No rotation/retention: run folders accumulate forever (109 in `studio/data/logs`).
- Readers: `studio/tools/query_runtime_logs.py` (sorted run dirs, `events-*.jsonl`, or
  `--lifecycle` -> `runs.jsonl`/`manifest.jsonl`; filters kind/ied/ref/control_ref/status_ref/
  text; invalid line -> error naming `path:line`), and the license time anchor (tail of
  `runs.jsonl`, literal `"at":"` match - requires compact separators).

## 5. Not persisted (memory only) - listed so they are not invented as files

Gateway/master traffic and command logs (`gateway/traffic_log.py`, deque 1000/200 entries,
`seq`, `at`=time.time()), EventBus (5000 events), live values, GOOSE engine state,
revision/session ids, migration notice, network previews/adapters cache, license status cache.

## 6. HTTP downloads / CSV (generated on request, never written to disk)

| Endpoint | Body | Headers |
|---|---|---|
| `GET /project/download` | `encode(project)` zip | `send_file`: `application/zip`, `attachment; filename=substation.subsim`, `Cache-Control: no-cache`, Content-Length, Accept-Ranges |
| `GET /ieds/<id>/source`, `POST /ieds/<id>/export-scl` | UTF-8 XML | `application/xml; charset=utf-8`; non-ASCII names -> `filename="ascii"; filename*=UTF-8''pct` (werkzeug) |
| `GET /tools/gateway/points.csv` | `csv.writer(lineterminator="\r\n")` cols `type,address,ied,reference,scale(:g),description` sorted by kind,address | `text/csv; charset=utf-8`, `attachment; filename=gateway-<p>-points.csv` |
| `GET /tools/master/points.csv`, `/health.csv`, `/live/<k>/comparison.csv`, `/tools/explorer/watch.csv`, `/signal-flow/edges/<id>.csv` | csv module output (default `\r\n`) | similar `attachment` headers (signal-flow quotes filename) |
| `GET /health/report.html` | HTML report | `text/html; charset=utf-8`, attachment |
CSV imports strip a leading U+FEFF (`domain/gateway_csv.py:25`), max 2 MB.

## 7. JSON conventions the Go port must reproduce

| Producer | Settings | Output shape |
|---|---|---|
| `flask.jsonify` (all REST, Flask 3.1.2 DefaultJSONProvider, no overrides in `app_factory.py`) | `sort_keys=True`, `ensure_ascii=True`, compact separators `(",",":")` (app.debug is False even in dev), NaN allowed (emits `NaN`), `default` handles datetime (HTTP date), UUID, Decimal, dataclass | body + trailing `"\n"`, `Content-Type: application/json` (no charset). Verified: `{"k":{"B":2,"a":3,"b":1},"s":"<a>&\u007f \u0001\b","x":NaN}\n` |
| SSE `/events` (`api/events.py:22`) | `json.dumps(event)` default separators `", "`/`": "`, insertion order, ensure_ascii | `retry: 1000\n\n`, `: heartbeat\n\n` every 10 s idle, `id: N\ndata: {...}\n\n`; resync event when cursor fell out of the 5000 buffer |
| autosave `project.json` | default separators, insertion order, ensure_ascii, `allow_nan=False` | see file 2 |
| run logs | compact, insertion order, ensure_ascii, `allow_nan=False`, LF | §4 |
| journal | sort_keys, default separators, ensure_ascii, CRLF | §2 |
| preferences / tool-settings / update_check / license.cert / client_config | `indent=2` (item sep `","`+newline, key sep `": "`), insertion order, ensure_ascii, CRLF, no final newline | files 3, 4 |
| launcher.json, license_state.json, helper request/result | default separators, insertion order, no newline | |
| worker stdin protocol | compact, `allow_nan=False` (`runtime/worker_protocol.py:26`) | |

Error envelope (`app_factory.py:76-97`): StudioError -> `{"code","details","error"}` 400 (409
when code `conflict`); pydantic ValidationError -> 400 `{"details":[...],"error":"Invalid
request"}` (`errors(include_context=False, include_input=False)`); ValueError/KeyError -> 400
`{"error": str(e)}` (KeyError str is the quoted repr, e.g. `"'file'"`); RuntimeError/
TimeoutError -> 503; HTTPException -> its code with `{"error": description}`.
Python float text: `1.0`, `1e-05`, `1e+16`, shortest round-trip repr; `ensure_ascii` escapes
every char >= 0x7f as lowercase `\uXXXX` (surrogate pairs above BMP), control chars as
`\b \f \n \r \t` or `\u00XX`; `<`, `>`, `&` are NOT escaped.

## Parity risks (file 4)

1. Go `encoding/json` differs from Python on every axis that matters: HTML escaping of `<>&`
   (must disable), raw UTF-8 vs `\uXXXX` (needs a custom ASCII escaper incl. `\u007f`),
   float format (`1.0` vs `1`, `1e-05` vs `0.00001`), struct field order vs sorted keys
   (jsonify sorts, files keep insertion order), NaN (`NaN` emitted by jsonify, error in Go),
   and trailing `"\n"` from jsonify.
2. ISO timestamps: Python `isoformat()` gives `+00:00` and drops `.ffffff` when zero; Go
   RFC3339Nano gives `Z` and trims zeros. The license time anchor scans `"at":"` in
   `runs.jsonl` and parses with `fromisoformat`; journal/run logs must keep the Python form.
3. Firewall rule names and the portable worker path are hashes of Python `json.dumps` text
   and `str(Path)`; a different canonical string orphans existing Windows rules (cleanup can
   no longer find them; journal `outstanding()` still lists old names).
4. License files are shared with native C++ (`license.cert`, `.time_anchor`) and must keep
   location (`%LOCALAPPDATA%\ScadaBench`, not the data folder), CRLF/`indent=2` JSON, the
   HWID formula, the rename-on-revoke scheme and the `STUDIO_LICENSE_CERT_PATH` env contract.
5. Run logs are append-only, LF, 10 MiB segments, lazy segment creation, constant
   `"application":"0.1.0"`, and must never be rewritten; journal is append-only CRLF.
6. The elevated helper is the same exe with `--network-helper` and a sha256 of the request
   bytes on the command line; the Go binary must accept the same argv and request/result files.
7. Text-mode CRLF also applies to `runtime/*.cfg` and the launcher log; confirm native config
   parsing tolerates whatever the Go port writes before changing line endings.
