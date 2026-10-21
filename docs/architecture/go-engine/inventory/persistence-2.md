# Persistence inventory 2/4 - `autosave.subsim` and `.subsim` archives

## 1. Files and writers

| Path | Writer | Notes |
|---|---|---|
| `<data>/autosave.subsim` | `projects/service.py:167-173` `save()` | only autosave; no versions kept |
| `<data>/autosave.tmp` | same; `destination.with_suffix(".tmp")` -> `autosave.tmp` (NOT `autosave.subsim.tmp`) | `write_bytes` then `os.replace(tmp, dest)` (MoveFileEx replace). No fsync. Leftover tmp never read or cleaned |
| HTTP download `substation.subsim` | `api/projects.py:35-38` | identical bytes to `encode(project)` |
| `recovery/*.subsim` | file 3 | same archive format (or exact old bytes) |

Readers: `startup_recovery.recover_startup` (file 3), `POST /api/v1/project/open`
(`api/projects.py:41-45`, multipart field `file`, needs runtime stopped), `self_test.py:108`,
`studio/tools/check_project.py`.

## 2. ZIP container (`projects/archive.py`)

| Property | Value (`archive.py:12-19`, verified on disk) |
|---|---|
| Entries | exactly one: `project.json` (decode refuses anything else: "Expected a single project.json entry") |
| Method | ZIP_DEFLATED (8), zlib default level (6); `compresslevel=None` |
| Header | `writestr(name, bytes)`: date_time = local `time.localtime()` at save (2 s resolution), create_system 0 (Windows), create_version/extract_version 20, flag_bits 0 (no data descriptor, no UTF-8 flag), external_attr `0o600 << 16` (0x01800000), no extra fields, no comment, no zip64 unless >4 GiB |
| Byte identity | impossible across saves (timestamp); readers must only require a standard zip |
| Size limits | compressed payload > 100 MiB -> "Project archive exceeds 100 MB"; entry `file_size` > 256 MiB -> "Unpacked project exceeds the 256 MB legacy recovery limit"; compact formats (2..15) with file_size > 100 MiB -> "Compact project exceeds 100 MB"; on write, JSON > 100 MiB -> "Engineering project exceeds the 100 MB save limit; nothing was written" |
| Bad zip / bad JSON | `StudioError("Invalid project archive")`; non-dict -> "Unsupported project format" |

`MAX_UNPACKED = 100*1024*1024`, `MAX_LEGACY_UNPACKED = 256*1024*1024` (`archive.py:8-9`).

## 3. `project.json` encoding

`json.dumps(compact(project), allow_nan=False).encode("utf-8")` (`archive.py:13`):
- separators `", "` and `": "`, **no** sort_keys, ensure_ascii=True (non-ASCII as `\uXXXX`
  lowercase hex, surrogate pairs; also DEL 0x7f escaped as `\u007f`), no indent, no trailing
  newline, no BOM. NaN/Infinity -> ValueError "Out of range float values are not JSON
  compliant" (surfaces as HTTP 400 or a failed save).
- Python float repr: `0.0`, `360.0`, `1e-05`, `1e+16`, `0.30000000000000004`; ints unbounded.
- Read: `json.loads(bytes)` (auto-detects UTF-8/16/32, accepts UTF-8 BOM in bytes), accepts
  NaN/Infinity tokens, duplicate keys last-wins.

### Top-level key order (insertion order, observed)

`id, name, diagram, sources, scenarios, test_panel, network, gateway, format, ieds, scl_sources`
(`compact()` copies project keys minus `ieds/format/scl_sources/migration_notice`, then appends
`format`, `ieds`, `scl_sources` - `document.py:10-19`).

| Key | Type | Source of truth |
|---|---|---|
| `id` | 32 lowercase hex (uuid4().hex) | restore requires `^[0-9a-f]{32}$` ("Invalid stored project identifier"); missing -> new uuid |
| `name` | str (truncated to 100 on restore; default "Untitled substation"/"Imported substation") | `service.py:35,214` |
| `diagram` | `{nodes:[...], edges:[...]}` pydantic `model_dump()` | §5 |
| `sources` | `[{ied_id, ref, spec}]` | `projects/validation.py:140-154`, `domain/source.py`, `domain/csv_trace.py:38-75` |
| `scenarios` | `[{id,name,enabled,steps:[{id,at(float),ied_id,ref,action,value}]}]` steps sorted by `at` (stable) | `domain/scenario.py` |
| `test_panel` | `[{id,label,ied_id,ref,report_ref|null,duration_ms,active_value}]` | `projects/test_panel.py:70-101` |
| `network` | `{mode,adapter_guid,adapter_label,nominal_hz,goose_reach,goose_uplink_guid,goose_share_guid}` | `domain/network_settings.py:11-12,35-53` |
| `gateway` | `{role,master,protocol,listen,real_commands,auto_start,settings,points}` | `domain/gateway_rules.py:19-24,120-148`, `domain/master_rules.py:30-34,156-173` |
| `format` | int, written as 15 | `document.py:6` |
| `ieds` | `[{id,name,filename,ip,port,role,live,lab_ip,goose_publish,test_mode,source_id}]` | `document.py:5,18` (`IED_FIELDS` order, only keys present) |
| `scl_sources` | `{sha256hex(source utf-8): source text}` in first-IED order; one entry per distinct text | `document.py:15-17` |

IED field values (observed): `ip` "127.50.0.N" (loopback, assigned `networking/loopback.py:5-16`
from .2 upward), `port` int (102), `role` simulated|real|off, `live`
`{ip,port,access(auto_reports|polling),poll_ms}`, `lab_ip` str, `goose_publish` bool,
`test_mode` `{devices:{domain:1..5}, simulation:bool}`.

Source specs: generator `{mode,offset(float),amplitude(float),period(float),seed(int),overridden:false}`;
csv `{mode:"csv",filename,csv_text,time_column,value_column,interpolation,repeat,points:[[at,value]...],row_count,duration,overridden}`.
Gateway point: `{kind,ied,ref,address,scale(float),description,id}`; stored points are passed
through unchanged on restore except `setdefault("id", "<ied>:<ref>:<kind>")`.

## 4. compact()/expand() (`projects/document.py`)

- `CURRENT_FORMAT = 15`; `COMPACT_FORMATS = {2..15}`.
- compact: `digest = sha256(ied["source"].encode("utf-8")).hexdigest()` (64 hex, full);
  each IED keeps only `IED_FIELDS` present + `source_id`. Derived data (`signals`,
  `controls`, `cfg`, `validation`, `state`, `goose_state`, ...) is never stored.
- expand: format 1 -> returned unchanged (IEDs carry inline `source`); format not in 1..15
  (including missing) -> "Unsupported project format"; `scl_sources` not dict / > 20 entries /
  `ieds` not list / > 20 -> "Invalid compact project source table"; any digest mismatch ->
  "Project SCL source checksum mismatch"; IED without valid `source_id` -> "IED refers to a
  missing project SCL source". Output: document minus `ieds/scl_sources` + rebuilt `ieds`.
- SCL `source` text = upload bytes `.decode("utf-8-sig")` (BOM stripped, CRLF kept;
  `scl/importer.py:103`). Non-UTF-8 SCL cannot be imported. API "source id" used by SLD/
  signal-flow/goose-links = `sha256(text)[:16]` (`projects/scl_sources.py:7-9`), a different
  (truncated) id from the archive key.

## 5. Diagram serialisation (pydantic 2.10.6 `model_dump()`, `contracts/project.py`)

Node: `{id, type:"equipment", position:{x:float,y:float}, data:{...}}`; unknown node/edge keys
dropped (`extra="ignore"`), unknown `data` keys rejected (`extra="forbid"`).
`data` order: `label, kind, rotation, state_mapping, control_binding, relay_statuses,
trip_command_polarity, trip_state_polarity, push_mode, push_value, push_rest_value,
push_reset_ms, initial_position, active_color, inactive_color, infeed_type, infeed_state`.
All defaults are written (null, [], "operate_false", ...). `infeed_type/infeed_state` removed
for non-INFEED and defaulted (line/live) for INFEED (`validation.py:107-116`).
`position` int input becomes float (`360` -> `360.0`, verified).
state_mapping variants: `{mode:"direct"|"single_to_dbpos", source:{ied_id,ref,report_ref}}`,
`{mode:"contacts_to_dbpos", open:{...}, closed:{...}}`. Edge: `{id,source,target,sourceHandle,targetHandle}`.
Limits: 2000 nodes, 5000 edges, 50 relay statuses, ids `^[A-Za-z0-9_-]{1,64}$`, edge id <=128.
Pydantic failure on restore = `ValidationError` (HTTP 400 `{"error":"Invalid request","details":[...]}`
for open; "unreadable" at startup).

## 6. Restore pipeline and migrations (`projects/service.py:175-251`)

1. `decode` (zip+expand). `version = int(raw.get("format", 1))`.
2. `ieds` must be list <= 20 ("Project must contain a list of at most 20 IEDs").
3. `import_stored` (`stored_import.py`): ids `^[0-9a-f]{32}$` ("Invalid stored IED
   identifier"), group by `(source, filename or "stored.scd")`, re-run full SCL import once per
   group with stored names; missing name -> "Stored IED is missing from its source XML";
   duplicates -> "Project contains too many or duplicate IEDs".
4. Per IED: `id`, `ip` (default ""), `port=int(item.get("port",102))`,
   `validate_ied_settings(item)` (fills format-11 defaults; error -> "Stored settings for IED
   <name>: <msg>").
5. `migrate_diagram(diagram, ieds, version)` (`mapping_migration.py:78-129`), cumulative:

| Step | Applies when | Effect / notice text |
|---|---|---|
| `_standard_commands` | always | pop `control_binding.compatibility`; non-"standard" -> notice "...the CONCEN command mode was removed; commands now follow standard IEC 61850 (Close = true, moves on Operate)" (formats 8-14) |
| `_retire_legacy` | version < 12 | drop TRANSFORMER/LOAD/BUSBAR nodes and their wires; pop `data.width` everywhere |
| return | version >= 10 | |
| `_button_edges` | version < 10 | remove wires touching PUSH_BUTTON nodes ("N wire(s) to push buttons removed...") |
| return | version >= 9 | |
| `_relay_entries` | version < 9 | RELAY `state_mapping` (direct/single_to_dbpos) -> `relay_statuses:[{id:"s1",label,source}]` |
| flat mapping | version <= 3 | `ied_id/status_ref/control_ref/report_ref/status_kind/position_values` -> `state_mapping`/`control_binding`, else notices |
| trip polarity | version < 7 | `trip_polarity` -> `trip_command_polarity` (v6 keeps value else `operate_false`), `trip_state_polarity="operated_true"` |

6. If `version < 15` (legacy): `migration_backup.preserve(dir, payload)` writes the EXACT
   input bytes to `recovery/before-format-8-migration-<uuid32>.subsim` (name fixed for every
   legacy format), re-reads and `testzip()` verifies. This happens even with `persist=False`
   (startup), and the autosave is not rewritten at startup, so every start of a legacy
   autosave creates another backup until the next save.
7. validate diagram, sources, scenarios, panel, `validate_network` ("Stored network settings:
   ..."), `normalize_gateway` ("Stored gateway settings: ..."), id check.
8. Swap project under lock; `migration_notice` = `{message, details:[...], backup:<abs path>}`
   for legacy (message varies: v14, v13, >=11, >=10, >=9, else); or GOOSE lab notice
   (`projects/goose_reach.py`) with `backup:""`; else None. Not persisted (omitted by
   compact). `assign_endpoints`; `save()` if persist; `changed()`. Rollback on exception.

## 7. When autosave is written (every call is a full rewrite)

| Trigger | Code |
|---|---|
| SCL import / template IED | `service.py:56` via `api/scl.py:40,48` |
| Diagram PUT (needs `revision`, `tab_id`) | `service.py:89-96`, `api/projects.py:16-26` |
| Node data during run (switch positions etc.) | `service.py:98-106` (no revision bump) via `simulation/equipment_control.py:82,103,125` |
| Source set | `service.py:108-115` via `simulation/coordinator.py:221` |
| Scenarios (saved twice) | `service.py:133-138` |
| Test panel | `service.py:144-154` (rollback on failure) |
| Gateway / master config | `service.py:156-165` via `api/gateway.py:54,75`, `api/gateway_master.py:36`, `gateway/manager.py:63,111,165`, `gateway/master_manager.py:49` |
| Explicit save | `POST /api/v1/project/save` (`api/projects.py:29-32`) |
| Open project | `restore(persist=True)` |
| IED settings / network / GOOSE reach / test mode | `projects/ied_settings.py:50,76`, `projects/goose_reach.py:25`, `simulation/test_mode_apply.py:40` |
| IED delete (bulk) | `bulk_removal.py:40` after backup |

No timer/debounce: saves are synchronous under `ProjectService.lock` (RLock) on the request
thread. There is no in-file revision. `revision` is an in-memory counter (starts 0 per
process, +1 per `changed()`), `session_id` = uuid4 per process; both only in the snapshot API.
Diagram PUT with stale revision -> 409 `"Another tab changed this project. This tab must
synchronize before saving."` code `conflict`.

## Parity risks (file 2)

1. `project.json` must keep Python's exact JSON dialect if byte-similar output matters:
   `", "`/`": "` separators, insertion key order (not alphabetical), ensure_ascii escaping
   (incl. `\u007f`), float repr with `.0` (`360.0`, `1e-05`). Go `encoding/json` writes
   `360`, `1e-05`->`0.00001`, raw UTF-8, and HTML-escapes `<>&`. Old Python builds still read
   ints where floats are expected (pydantic coerces, gateway `scale` passes through raw).
2. `scl_sources` keys are sha256 of the decoded Unicode text re-encoded as UTF-8 (BOM
   stripped); hashing raw upload bytes in Go would give "Project SCL source checksum mismatch".
3. Restore re-runs the full SCL import (XSD validation, model build) for every stored source:
   any Go importer difference changes `signals`/`controls`, which can make stored diagram
   bindings, scenarios, sources, panel or gateway points fail validation -> autosave reported
   "unreadable" at startup.
4. Migration chain semantics (version gates, notice wording, "before-format-8-migration"
   name, backup written even when not persisting) must be reproduced, including format 1
   (inline `source`) and missing `format` = unsupported.
5. Pydantic coercions and defaults (`extra=ignore` vs `forbid`, int->float, all defaults
   materialised, discriminated union by `mode`) and its error payload shape for 400 responses.
6. ZIP writer must produce a single DEFLATE entry without data-descriptor quirks that Python
   `zipfile` rejects; Python must be able to open Go-written autosaves (downgrade path).
7. Atomic save: Go must write `autosave.tmp` then `MoveFileEx(REPLACE_EXISTING)`; same temp
   name matters only for leftover cleanup, but a different name leaves stray files.
