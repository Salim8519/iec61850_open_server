# Persistence inventory 3/4 - recovery archives, startup recovery, small JSON stores

## 1. `recovery/` folder

All files are `<data>/recovery/<prefix>-<uuid4 hex 32>.subsim`. The folder is created with
`mkdir(exist_ok=True)` (parent must exist). Every writer first refuses a symlinked folder.
Nothing ever deletes, lists, rotates or reads them back automatically (user restores via
"Open project"). Paths returned to the UI are absolute `str(Path)` (backslashes).

| Prefix | Writer | Content | Write mode / verification | Errors |
|---|---|---|---|---|
| `before-format-8-migration-` | `projects/migration_backup.py:8-23` (from `service.restore` when format < 15) | exact input bytes of the legacy archive | `write_bytes` (overwrite allowed; uuid makes it unique), then `read_bytes() == payload`, then `ZipFile(...).testzip()` | "Recovery directory must not be a symbolic link"; "Project migration recovery verification failed"; "...recovery archive is corrupt"; "...recovery archive is invalid" (StudioError) |
| `before-ied-delete-` | `projects/bulk_removal.py:21-45` | `service.export()` = freshly encoded current project (format 15) | `open("xb")` (exclusive create), write, sha256 compare | "Recovery directory must not be a symbolic link"; "Recovery backup verification failed; nothing deleted"; "Project changed. Review a fresh deletion preview" (fingerprint mismatch) |
| `unreadable-autosave-` | `projects/startup_recovery.py:18-30` | byte copy of autosave (`shutil.copyfile`) | no verification | RuntimeError "Recovery folder is a link; cannot safely preserve unreadable autosave"; "Cannot preserve unreadable autosave; startup stopped to protect it" |
| `before-backend-restart-` | `studio/tools/check_project.py:46-57` (dev tool, `--backup`) | exact autosave bytes | `open("xb")`, sha256 verify | tool-only |
| legacy (read never) | older builds | `before-format-6-migration-`, `before-mapping-migration-`, `before-immediate-restart-` | - | observed in `studio/data/recovery`; Go must leave them alone |

### Bulk delete fingerprint (transient, but must match between preview and delete calls)

`bulk_removal.py:10-18`: `sha256(json.dumps({"project": compact(project), "ids": sorted(keys)},
sort_keys=True).encode())` - default separators, ensure_ascii. Keys must be 1..20 unique str
("Select 1 to 20 unique IEDs"). Preview returns `{ieds:[{id,name}] (sorted by id), bindings,
sources, test_controls, scenario_steps, scenarios_removed, fingerprint}`. Single delete
`DELETE /ieds/<id>` = preview + remove in one call, returns `{id,name,<impacts>,recovery_file}`.
Removal (`projects/removal.py:32-49`): drops IED, its sources, panel entries, nulls
`state_mapping`/`control_binding` that use it, filters relay statuses, drops scenario steps and
then empty scenarios. Gateway points are kept. Order: backup -> swap -> `save()` -> `changed()`
(rollback of memory on save failure; backup stays).

## 2. Startup recovery (`projects/startup_recovery.py`, `projects/background_recovery.py`)

Runs on thread `project-restore` started by `create_app` (`app_factory.py:49,73`).

| Case | Result (`recovery.result`, shown in `GET /project` as `startup_recovery`) |
|---|---|
| no `autosave.subsim` | `None` (new empty project kept in memory; nothing written) |
| exists but symlink / not regular file | thread error: RuntimeError "Autosave is not a regular local file; inspect it before starting ScadaBench" |
| `st_size > 100 MiB` or any exception in `restore(persist=False)` | copy to `recovery/unreadable-autosave-*.subsim`, result `{"status":"unreadable","message":"The last autosave could not be reopened. A separate recovery copy was preserved before allowing new work.","details":str(error),"backup":"<abs path>"}` |
| success | `{"status":"restored","message":"Recovered the last saved substation. All IEDs remain stopped."}` |

- After "unreadable", the in-memory project is the empty default; the next save overwrites
  `autosave.subsim` (the copy in `recovery/` is the only original).
- Request gate (`api/startup_gate.py`): every `/api/*` request except paths starting with
  `/api/v1/session`, `/api/v1/license`, `/api/v1/preferences` waits up to 300 s; timeout ->
  503 `{"code":"project_opening","error":"ScadaBench is still opening your last project. Try
  again in a moment."}`; thread error -> 503 `{"code":"startup_recovery_failed","error":
  "ScadaBench stopped to protect your last autosave: <err>. Close ScadaBench, inspect the data
  folder, then start it again."}`.
- After recovery, `_resume_gateway` (`app_factory.py:114-119`) auto-connects the master and
  auto-starts the gateway when the stored `gateway.master.auto_connect` / `gateway.auto_start`
  flags say so (persisted state that triggers side effects at start).
- Never starts IEDs; restore at startup bumps revision to 1.

## 3. Stored SCL sources

Not separate files: original SCL text lives only inside `project.json` `scl_sources`
(file 2 §4). Read paths: `projects/scl_sources.py:12-26` builds `{id16: {id, filename,
ieds:[names], text}}` per distinct text in IED order (lru_cache on `source_id`, maxsize 20);
`projects/stored_import.py` rebuilds models at restore. Downloads:
`GET /ieds/<id>/source` -> `ied["source"].encode("utf-8")` (BOM not restored), mimetype
`application/xml` (Flask adds `; charset=utf-8`), `download_name=ied["filename"]`;
`POST /ieds/<id>/export-scl` -> `<name>.edited.scd`. Unknown id -> 404 code `not_found`
"This SCL file is not in the project".

## 4. `preferences.json`

| Field | Value |
|---|---|
| Path | `<data>/preferences.json` (temp `preferences.json.tmp`) |
| Code | `projects/preferences.py:18-61`, `projects/list_filters.py`; API `api/preferences.py` (never waits for startup) |
| Write | `json.dumps(current, indent=2)` -> `write_text(utf-8)` (**CRLF**, no trailing newline), then `os.replace`; under a per-store `threading.Lock`; `mkdir(parents)` first |
| Key order | `tour_seen_version, sld_goose_layer, sld_readers_layer, list_filters` (`tour_version` is NOT stored) |
| Types | ints (bools rejected); `tour_seen_version` 0..1000, `sld_*` 0..1; `list_filters` = `{listId: {key: str<=120 | bool}}` |
| list_filters rules | list id `^[a-z][a-z0-9-]{0,39}$`, key `^[a-z][a-zA-Z0-9_]{0,31}$`, <=40 lists, <=16 keys per list (first 16 kept on read), PUT replaces one list wholesale |
| Defaults | `{"tour_seen_version":0,"sld_goose_layer":0,"sld_readers_layer":0}`, `list_filters {}`; GET adds `tour_version: 1` |
| Missing/corrupt | `OSError`/`ValueError` (incl. UTF-8 BOM, since `read_text("utf-8")` keeps U+FEFF) -> `{}` -> defaults; wrong-typed values silently ignored; invalid list entries dropped |
| PUT errors | "Preferences must be a JSON object"; "Unknown preference: a, b" (sorted); "Preference X must be a whole number"; "Preference X is out of range"; "list_filters must be an object of lists"; "Unknown list name: 'x'" (Python repr quoting); "Filters of X must be short text or true/false values"; "Too many remembered lists" -> 400 |
| Observed | `{\r\n  "tour_seen_version": 1,\r\n  "sld_goose_layer": 1,\r\n  "sld_readers_layer": 0,\r\n  "list_filters": {...}\r\n}` |

## 5. `tool-settings.json` (tool memory)

| Field | Value |
|---|---|
| Path | `<data>/tool-settings.json` (temp `tool-settings.json.tmp`) |
| Code | `tools/tool_memory.py:45-65`; writers `tools/explorer_auto.py:41` (explorer), `tools/ied_scan.py:54` (find range/port), `api/tools.py:120` (find choice); readers `tools/explorer.py:244`, `tools/explorer_auto.py:24,37`, `tools/ied_scan.py:138`, `GET /tools/memory` (`api/tools.py:111`) |
| Write | read+clean, `current[tool].update(changes)`, `_clean`, `json.dumps(indent=2)` -> `write_text` (**CRLF**, no trailing newline) -> `os.replace`; under lock |
| Schema/order | `{"explorer": {"host": str<=64, "port": int, "interval_ms": int 500..60000 (else 1000), "watch_by_host": {host<=64: [{"ref": str<=200, "fc": str<=4, "type": str[:32]}] (<=200)} (last 50 hosts kept)}, "find": {"range": str<=64, "port": int, "choice": str<=200}}` |
| Defaults | host "", port 102, interval_ms 1000, watch {}, range "", choice "" |
| Quirk | `port` check is `isinstance(int)` so `true/false` survive as bools; out-of-range ints kept |
| Missing/corrupt | defaults, never an error |

## 6. `update_check.json`

| Field | Value |
|---|---|
| Path | `<data>/update_check.json` (temp `update_check.json.tmp`) |
| Code | `updates/service.py:32-157`, `updates/versions.py` |
| Write | `json.dumps(state, indent=2)` -> `write_text` (**CRLF**, no newline) -> `os.replace`; under lock |
| Keys | `auto` bool (default true when missing / not `false`), `dismissed` version str or null, `release` record or null, `etag` str or null, `error` null or one of `offline`, `no_server`, `no_release`, `server_error`, `bad_answer`, `checked_at` int epoch seconds |
| Key order | `check()`: `{**stored, release, etag, error}` then `checked_at` appended/kept position; `configure()`: stored order + `auto`/`dismissed` appended if new. Observed: `release, etag, error, checked_at` |
| release record | `{version, date, min_version ("" if none), notes (stripped, <=600), page, installer:{url,sha256}, portable:{url,sha256}|null}` - only after Ed25519 check of `b"scadabench-release-v1\n" + payload` |
| Schedule | first check 20-120 s after start, then hourly (+0-600 s) if `auto` and `now - checked_at >= 86400`; manual "check now" reuses answer within 60 s; `SCADABENCH_UPDATE_CHECK=off` disables thread |
| Network | `GET <server>/api/v1/release`, `If-None-Match: <etag>`, 304 keeps release; body max 16 KiB; UA `ScadaBench` |
| Missing/corrupt | `{}` -> defaults; failures stored as error codes, never raised |
| configure errors | "Only auto and dismissed can be changed", "auto must be true or false", "dismissed must be a version" |

`status()` (API) derives `available`, `important` (min_version newer), `dismissed` (only when
not important), `latest`; version regex `^\d{1,4}\.\d{1,4}\.\d{1,4}(\+[0-9A-Za-z.-]+)?$`.

## Parity risks (file 3)

1. `before-ied-delete` uses exclusive create and sha256 verification before any change; the Go
   port must keep "backup first, verified, else nothing deleted" and absolute path strings.
2. Startup "unreadable" path copies the autosave and then allows new work that overwrites
   it; any Go restore strictness difference (file 2) silently turns users' autosaves into
   recovery copies. Test with real legacy autosaves (formats 1-15) before release.
3. The three small JSON stores are CRLF `indent=2` with Python key order, written atomically
   via `<name>.tmp` + replace; Go must also tolerate LF/CRLF/BOM on read the same way (BOM ->
   treated as corrupt -> defaults).
4. Python type quirks: `bool` is an `int` (tool-settings `port` keeps bools, preferences reject
   them explicitly), `int` vs float distinctions, and error strings using Python `repr`
   (`'x'`) leak into API messages.
5. Bulk delete fingerprint is sha256 of Python `json.dumps(..., sort_keys=True)` with default
   separators and ensure_ascii; any Go canonicalisation works only if preview and delete both
   run in Go (do not mix with stored Python fingerprints).
6. Update record verification uses the release domain prefix and Ed25519 keys from
   `licensing/release_key.py`; `dismissed`/`auto` semantics (missing = true) must not change.
7. Startup gate path prefixes and 300 s wait, and gateway/master auto-start after restore,
   are user-visible behaviours tied to persisted flags.
