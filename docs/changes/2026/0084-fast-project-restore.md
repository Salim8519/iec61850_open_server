# 0084: Faster launch: restore imports each shared SCL source once

Date: 2026-10-17. No decision record: no contract, format, or dependency change.

## Why

The owner reported that the installed app opens slowly. Measured on the installed
0.1.x build, the cause was not the app window: Edge loads the page in about 0.6 s, even
with a fresh profile. The backend took **29 s** to become ready with the owner's
autosave and 0.7 s without it.

`ProjectService.restore()` called `import_scl` once per stored IED. The project held 8
IEDs from one 14.8 MB SCD (plus 1 CID), so the same document was parsed twice and
XSD-validated once **per IED** on every launch.

## Changed files

- `studio/backend/studio_app/projects/stored_import.py` (new): groups stored IEDs
  by identical source and filename, imports each group once with all of its names
  selected, and gives each IED its own copy of the validation report. It keeps the
  existing error messages for malformed, duplicate, and missing IEDs.
- `studio/backend/studio_app/projects/service.py`: `restore()` uses `import_stored`.
- `studio/tests/unit/test_stored_import.py` (new): one import per shared source,
  identical models and IDs after restore, and per-IED validation objects.

## Evidence

- Owner's real autosave (copy): old path 28.7 s, new path 6.2 s. All 9 rebuilt models
  are identical to the old per-IED import (JSON compare, random `id` excluded).
- Installed app timing with an empty data folder: ready in 0.7 s. Edge `--app` page
  load with a fresh profile: 0.64 s; the second launch takes 0.54 s.
- Restore-related unit and integration tests: 48 passed. Full unit suite: 290
  passed. 2 failures in `test_live_contract` come from the Tools pane's in-progress
  native live operations, which this change does not touch. `verify.py`: PASS.

## Limitations

- The remaining ~6 s is the real model build for 8 large IEDs. Getting below that
  would mean showing the window first and restoring in the background, which is a
  launcher/UI design change that needs its own decision record.
- The app-window browser profile grows (~500 MB of Edge components). It does not
  slow launch measurably and is left as is.
- Needs a new installer build to reach users; it ships with the next build.
