# 0085: Window opens in about 1 s; the project restores in the background

Date: 2026-10-17. Decision: [ADR-0058](../../decisions/ADR-0058-window-first-startup.md).
Follows [0084](0084-fast-project-restore.md).

## Changed files

- `studio/backend/studio_app/projects/background_recovery.py` (new): recovery thread.
- `studio/backend/studio_app/api/startup_gate.py` (new): engineering requests wait
  for recovery; a recovery that stopped to protect the autosave makes them answer 503.
- `studio/backend/studio_app/app_factory.py`: starts recovery last and registers the
  gate after `security.protect_request`.
- `studio/backend/studio_app/api/projects.py`: `startup_recovery` from the background
  result.
- `studio/backend/studio_app/launcher/self_test.py`: waits for recovery.
- `studio/backend/studio_app/scl/schema_validation.py` + `scl/importer.py`: reuse
  the parsed root (one XML parse per import instead of two).
- `studio/frontend/src/app/App.tsx`: "Opening your last project…" placeholder.
- Tests: new `tests/unit/test_startup_gate.py` (requests wait, preferences don't; a
  protective recovery stop refuses work and keeps the autosave; license 402 answers
  before the wait); `test_startup_recovery.py` waits for recovery.

## Evidence

Measured by running the launcher from source (`--no-browser`) on a copy of the owner's
autosave (9 IEDs):

| | Before (installed 0.1.x) | After |
|---|---|---|
| Window can open | 29 s | 0.9 s |
| Page loaded + session | ~29.5 s | 1.2 s |
| Project ready | ~29.5 s | 6.7 s |

- Python unit + integration: 341 passed. Frontend `node --test`: 107 passed. `tsc`
  is clean. `verify.py`: PASS.

## Limitations

- Still about 6 s until a large project is ready to work with; that is the model build.
  The window shows a status message meanwhile.
- These timings were measured from source; the installed build needs re-measuring
  after the next installer build.
