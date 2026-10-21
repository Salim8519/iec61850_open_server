# Persistence inventory 1/4 - data folder, launcher, lock, resources

Scope: every disk artefact the Python backend (`studio/backend/studio_app`) reads or writes.
Paths in `file:line` are relative to `studio/backend/studio_app/` unless prefixed `studio/`.
Files: 1 = folder/launcher, 2 = autosave.subsim, 3 = recovery + small JSON stores,
4 = licensing, network, runtime logs, JSON conventions.

## Global Windows text-mode rule (applies to every file in this inventory)

Python `Path.write_text()` / `open(..., "w"/"a")` without `newline=` translates `"\n"` to
`"\r\n"` on Windows. Verified on disk: `data/preferences.json`, `data/update_check.json`,
`data/tool-settings.json` are CRLF. Only `observability/log_writer.py` passes `newline="\n"`.
Bytes writers (`write_bytes`, zip) are untouched. The Go port must emit CRLF exactly where
Python text mode did (listed per artefact as **CRLF** or **LF**). No writer emits a BOM.
`indent=2` JSON files have **no trailing newline**.

## 1. Data folder resolution

| Mode | Data folder | Decided in |
|---|---|---|
| Dev (`tools/run_dev.py`) | `studio/data` (passed explicitly `--data-dir`) | `studio/tools/run_dev.py:125` |
| Source, no `--data-dir` | `studio/data` (`Path(settings.py).parents[2] / "data"`) | `settings.py:42-43` |
| Frozen folder build, `<exe dir>\data` exists | `<exe dir>\data` | `settings.py:20-26,40` |
| Frozen folder build, no `data` dir, `LOCALAPPDATA` set | `%LOCALAPPDATA%\ScadaBench\data` | `settings.py:26` |
| Frozen folder build, `LOCALAPPDATA` empty | `<exe dir>\data` | `settings.py:24` |
| Frozen one-file portable exe | `<exe dir>\ScadaBench-data` | `launcher/portable_runtime.py:15,25`; `settings.py:36-39` |
| `--data-dir X` (any mode) | `X` (overrides all of the above) | `launcher/entrypoint.py:48-49` |
| `--self-test` | temp `<data>\studio-check-XXXXXXXX` (deleted after) | `launcher/self_test.py:41-45` |

- One-file detection: `is_single_file(exe_dir, sys._MEIPASS)` is true when the unpack dir is
  neither the exe dir nor under it (`portable_runtime.py:19-22`). Go equivalent: a build flag
  or "is this the portable binary" marker; behaviour must stay identical.
- Frontend: frozen = `_MEIPASS/frontend.pak` if file else `_MEIPASS/frontend`; dev =
  `studio/frontend/dist` (`settings.py:33-34,43`).
- Native worker: dev `studio/native/build/Release/studio_ied.exe`; frozen
  `_MEIPASS/native/studio_ied.exe`; portable `<data>/workers/<key>/native/studio_ied.exe`.
  `studio_live.exe`, `studio_probe.exe`, `studio_control.exe`, `studio_gateway.exe` are
  siblings (`settings.py:15-17`, `app_factory.py:52`, `self_test.py:31`).
- License home is NOT the data folder: `%LOCALAPPDATA%\ScadaBench` (fallback
  `~/AppData/Local/ScadaBench`) in every mode, including portable and dev
  (`licensing/service.py:27-29`). See file 4.
- Data folder created with `mkdir(parents=True, exist_ok=True)` by DirectoryLease
  (`directory_lease.py:11`), ProjectService (`projects/service.py:30`) and `_serve`
  (`entrypoint.py:165`).

### Environment variables read/written

| Variable | Use | Where |
|---|---|---|
| `LOCALAPPDATA` | packaged data folder, license home, legacy move, Chrome lookup | `settings.py:40`, `licensing/service.py:28`, `entrypoint.py:45`, `app_window.py:21` |
| `STUDIO_DEV_TOKEN` | default `--dev-token` (default `"dev"`) | `entrypoint.py:32` |
| `STUDIO_FRONTEND_URL` | default `--frontend-url` (`http://127.0.0.1:5173`) | `entrypoint.py:34` |
| `SCADABENCH_UPDATE_CHECK=off` | disables update thread | `updates/service.py:146` |
| `STUDIO_LICENSE_CERT_PATH` | SET by backend (enforce only) to `<home>\license.cert`; read by native `license_guard.cpp:257` | `licensing/service.py:61-62` |
| `STUDIO_MODULE_PATH`, `STUDIO_MODULE_SIG` | SET when `studio_app.pyd` + `.sig` exist; native verifies module | `entrypoint.py:74-82`, `native/common/license_guard.cpp:272-274` |
| `SystemDrive` | volume serial for HWID | `licensing/hwid.py:103` |
| `ProgramFiles`, `ProgramFiles(x86)` | Edge/Chrome lookup | `app_window.py:16-22` |

## 2. Legacy folder move (packaged only)

| Item | Value |
|---|---|
| Writer | `launcher/legacy_folder.py:13-26`, called `entrypoint.py:43-45` (frozen only, before Settings) |
| Action | `%LOCALAPPDATA%\SubstationStudio` -> `%LOCALAPPDATA%\ScadaBench` via `os.replace` |
| Conditions | only if old is a dir and new does not exist; if both exist -> nothing ("kept-both"); OSError -> silently "nothing" |
| Effect | moves both license home and `data\` of installed copies |

## 3. `studio.lock` (single writer per data folder)

| Field | Value |
|---|---|
| Path | `<data>/studio.lock` |
| Writer/locker | `launcher/directory_lease.py:10-28`; held for whole process (`entrypoint.py:137-148`) |
| Content | single byte `0` (ASCII 0x30) written only when file is empty; file opened `a+b`; never truncated or deleted |
| Lock | `msvcrt.locking(fd, LK_NBLCK, 1)` after `seek(0)` = Win32 `LockFile` on byte range [0,1), non-blocking. Mandatory lock: other processes get "resource busy" reading byte 0 (observed with `od`). Unlock `LK_UNLCK` then close |
| Symlink | refuses: `RuntimeError("ScadaBench lock file is a link; inspect the data folder before starting")` |
| Lock failure | `RuntimeError("This ScadaBench data folder is already open or cannot be locked. Close its other launcher before opening it again.")`; dev re-raises; packaged -> `show_running_instance` |
| Go port | `windows.LockFileEx(h, LOCKFILE_EXCLUSIVE_LOCK|LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, ol{Offset:0})` on same byte so Python and Go builds exclude each other |

## 4. `launcher.json`

| Field | Value |
|---|---|
| Path | `<data>/launcher.json` |
| Writer | `launcher/entrypoint.py:98-100` `write_launcher_record`; called at `:166` (url only, every launch incl. dev) and `:133` (packaged, after window opens, adds `window_pid`) |
| Format | `json.dumps(record)` default separators `", "`/`": "`, ensure_ascii, insertion order: `{"url": "http://127.0.0.1:8765"}` or `{"url": "...", "window_pid": 1234}` (omitted when falsy). No newline. UTF-8. Observed on disk: `{"url": "http://127.0.0.1:8765"}` (32 bytes) |
| Atomicity | none: `write_text` truncate+write (not temp+replace) |
| Reader | `entrypoint.py:111-114` (second launch, lock failed): `json.loads(...).get("window_pid")`; any OSError/ValueError/AttributeError -> `None` |
| Missing/corrupt | ignored; falls back to named-event show request |
| Port | default 8765; if busy (non-dev) bind port 0 -> url carries the ephemeral port (`entrypoint.py:158-164`) |

## 5. Single-instance "show" event (not a file, but cross-process contract)

`Local\ScadaBench-Show-` + `sha256(os.path.normcase(os.path.abspath(data_dir)).encode("utf-8")).hexdigest()[:24]`
(`launcher/single_instance.py:16-18`). `normcase` lowercases and turns `/` into `\`.
Auto-reset event; second launch `OpenEventW(EVENT_MODIFY_STATE)` + `AllowSetForegroundWindow(ASFW_ANY)` + `SetEvent`.

## 6. Portable worker staging `workers/<key>/native/`

| Field | Value |
|---|---|
| Path | `<data>/workers/<16 hex>/native/*.exe` (portable one-file only) |
| Writer | `launcher/portable_runtime.py:36-60` |
| Key | `sha256( concat over sorted(Path(src).glob("*.exe")) of name.utf8 + b"\0" + sha256(file bytes).digest() ).hexdigest()[:16]` (`:29-33`). Sorted by `WindowsPath` order, which compares the case-folded (`_str_normcase`) full path, i.e. case-insensitive; hashed name keeps original case |
| Reuse | if every expected exe exists in target -> reuse as-is (no content check) |
| Atomicity | copy (`shutil.copy2`, keeps mtime) into `<key>.partial-<pid>/native`, `rmtree(target.parent)`, `os.replace(staging, target.parent)`; on OSError cleans staging and re-raises unless a concurrent copy completed |
| Errors | `RuntimeError("No native workers were unpacked in <src>")` |
| Why it matters | Windows firewall rules name the worker path; a different key = new path = new rules + UAC prompt |
| Cleanup | old key folders are never deleted |

## 7. App window browser profiles

| Field | Value |
|---|---|
| Paths | `<data>/browser-profile`, `browser-profile-2`, `browser-profile-3`; fallback `%TEMP%\ScadaBench-browser-*` (`launcher/app_window.py:54-64`) |
| Owner | Chromium (Edge/Chrome `--user-data-dir`). Backend only `mkdir` and probes `<profile>/lockfile` by trying to `unlink` it: FileNotFound -> free; other OSError -> in use (`:43-51`) |
| Packaged only | `--app=<url>&window=app --no-first-run --no-default-browser-check --window-size=1440,900` |

## 8. Launcher log (windowed exe only)

| Field | Value |
|---|---|
| Path | `<data>/logs/launcher-YYYYMMDD.log` (local date at start; `launcher/windowed.py:36-48`) |
| When | only when `sys.stdout`/`stderr` is None (double-clicked windowed exe); terminal launches log to console |
| Format | Python `logging` FileHandler, append, UTF-8, text mode => **CRLF**; `"%(asctime)s %(levelname)s %(name)s: %(message)s"`, asctime like `2026-10-19 14:03:22,517` (local time, comma millis); tracebacks multi-line |
| Level | INFO |
| Rotation | none; one file per start date; never deleted |
| Mkdir failure | logging to NullHandler |

## 9. `frontend.pak` (read-only packaged resource)

| Field | Value |
|---|---|
| Reader | `launcher/frontend_assets.py:42-51,61-67` (writer = `pack()` same file, used by `studio/tools/pack_frontend.py`) |
| Layout | `b"SSPAK1"` + 16-byte nonce + XOR(body, SHAKE-256(key32 || nonce) keystream). Body = `<u32 LE header length>` + header JSON `{"relpath":[offset,length]}` (`separators=(",",":")`, sorted names) + concatenated `zlib.compress(level 9)` blobs. Nonce = `sha256(body)[:16]`. Key `8c1e5a7f...5f83` (`:17`) |
| Serving | `index.html` -> `Cache-Control: no-cache`; `assets/*` -> `public, max-age=31536000, immutable`; `.js` forced `text/javascript`, else `mimetypes.guess_type` or `application/octet-stream`; missing -> `404 "Not found"` text. Folder mode uses Flask `send_from_directory` (different headers: ETag/Last-Modified, conditional 304) |
| Missing UI | `GET /` -> `503 "ScadaBench frontend build is not available yet."` (`app_factory.py:101-105`) |

## 10. Self-test temp data (`--self-test`)

`launcher/self_test.py:39-117`: `tempfile.TemporaryDirectory(prefix="studio-check-", dir=<data>)`
creates a full private data folder (autosave, recovery, logs, runtime cfg) then deletes it.
Prints `json.dumps(result)` (default separators, insertion order) to stdout.

## 11. Data folder tree (observed `studio/data`, plus code-derived entries)

```
<data>/
  studio.lock                 (§3)          launcher.json          (§4)
  autosave.subsim, autosave.tmp (file 2)    preferences.json[.tmp] (file 3)
  tool-settings.json[.tmp]    (file 3)      update_check.json[.tmp] (file 3)
  recovery/*.subsim           (file 3)      runtime/<ied_id>.cfg   (file 4)
  logs/runs.jsonl, logs/<run_id>/{manifest.jsonl,events-NNNN.jsonl} (file 4)
  logs/launcher-YYYYMMDD.log  (§8)          network/journal.jsonl, network/requests/* (file 4)
  workers/<key>/native/*.exe  (§6)          browser-profile*/      (§7)
%LOCALAPPDATA%/ScadaBench/  license.cert, client_config.json, license_state.json,
  .time_anchor, license.cert.<verdict>-<ts>  (file 4)
```

## Parity risks (file 1)

1. CRLF: every Python text-mode write on Windows is CRLF; Go `os.WriteFile` writes LF.
   Must translate per artefact or old/new files differ (and `read_text` consumers differ).
2. `studio.lock` must lock the same byte (offset 0, length 1) with LockFile semantics, else
   an old Python build and a new Go build can open one data folder at once and clobber autosave.
3. Data folder decision order (`--data-dir` > portable `ScadaBench-data` > `<exe>\data` exists >
   `%LOCALAPPDATA%\ScadaBench\data` > `<exe>\data`) and the legacy `SubstationStudio` move must
   run before any folder is created, or installed users lose license/projects.
4. Portable worker key must be byte-identical (`name\0sha256digest`, sorted, first 16 hex),
   else every upgrade-in-place creates a new worker path, new firewall rules and UAC prompts;
   also Python `WindowsPath` sort is case-insensitive.
5. `launcher.json` is not atomic and is rewritten mid-start with `window_pid`; second-launch
   focus logic depends on it. Event name hash depends on `normcase(abspath())` exact string.
6. `STUDIO_MODULE_PATH/SIG` points native workers at the signed `.pyd`; a Go backend needs a
   new module-signing story or native `verify_module` must be changed (ADR needed).
7. `frontend.pak` format (SHAKE-256 XOR, zlib, LE header) must be reproduced exactly by the Go
   reader, and folder vs pak serving headers differ today (keep or document).
