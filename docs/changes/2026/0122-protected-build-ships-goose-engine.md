# 0122: Protected build ships the GOOSE engine (dev/packaged parity fix)

Date: 2026-10-20. Found by the startup/packaging planning review of the Go speed plan.

## Problem

`studio/packaging/studio_protected.spec` (installer and portable release) listed six native
programs and left out `studio_goose.exe`, which `studio.spec` and development ship (ADR-0073).
The last protected build (`studio/dist-protected/ScadaBench/_internal/native`) indeed holds only
`studio_control`, `studio_gateway`, `studio_ied`, `studio_live`, `studio_master`, `studio_probe`.
In such a build `acquisition/goose_engine.py` answers "The GOOSE engine (studio_goose.exe) is not
built yet" when a project with GOOSE runs, while development works: a dev/packaged difference.

## Change

- `studio/tools/compile_protected.py`: the licensed-workers step also builds the Go GOOSE engine
  into the protected native folder (`build_goose_engine.build`, tests included; printed in
  `--dry-run`).
- `studio/packaging/studio_protected.spec`: `studio_goose.exe` required and shipped in `native`.
- `studio/tests/unit/test_protected_packaging.py`: new test, both specs must ship exactly the same
  set of native programs.

## Evidence

- `tests/unit/test_protected_packaging.py`: 13 passed (new parity test included); dry-run still
  lists all steps. No EXE was built (owner rule); the next release build will carry the engine.

## Limitations

- `studio_goose.exe` is not in the sealed-hash list (`protected_source.SEALED`): the engine does not
  verify module integrity today, same as development. Whether it should is a separate decision.
- Releases already published lack GOOSE publishing until the next release.
