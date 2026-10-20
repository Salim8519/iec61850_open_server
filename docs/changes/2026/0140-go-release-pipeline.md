# 0140: Go release pipeline: installer + portable exe on the Go engine

Date: 2026-10-20. Owner: "make production pipeline for go, it gives exe installer and exe portable
too", "go exe installer i mean", "users should never suffer, one click and all running".
Decision: [ADR-0083](../../decisions/ADR-0083-go-release-packaging.md).

## Change

- `studio/tools/compile_go_release.py` (+ `go_release_steps.py`, `go_release_resources.py`,
  `build_xsd_helper.py`): one command builds `studio/dist-go/ScadaBench/`,
  `ScadaBench-Setup-<v>.exe` and `ScadaBench-Portable-<v>.exe`. Frontend pak, licensed workers
  and the installer step are the Python release's own functions (`compile_protected.py`).
- `cmd/studio_server`: packaged window mode (`windowed.go`), native seal and module signature
  (`packaged.go`), `--self-test` (`selfcheck.go`), `_internal` resource folder (`paths.go`),
  `--dev` refused in the packaged build, second start focuses the running window.
- `internal/desktop` (lane D, merged fc45c23): app window, show event, focus, message boxes,
  launcher log, terminal attach, legacy folder move; event names equal Python's (7 vectors).
- `internal/selftest` (lane S, merged 02edbd8): Python's self-test JSON byte for byte.
- `cmd/scadabench_portable`: portable launcher embedding the installed folder.
- `studio_xsd.exe` build (lane X, merged 76fbe61) and `tools/build_xsd_helper.py`.
- `internal/app`: `Options.NativeSeal` -> licensing `ExpectedFiles`.
- `installer.iss`: `[InstallDelete] {app}\_internal` before copying (Go <-> Python upgrades).
- Workers' VC++ runtime DLLs shipped in `_internal/native` (found while building: the Go
  folder had none; the workers import MSVCP140/VCRUNTIME140).
- GNU libiconv (LGPL-2.1) built as `libiconv-2.dll` instead of static (`xsd/cmake/deps`),
  notice `packaging/libiconv-NOTICE.txt`, source archive uploaded by `website/deploy.cmd` to
  `download.scadabench.com/source/`; parity re-run with the DLL: 809/809 (corpus 559, edge 250).
- Main version: owner "deploy the go version to the cloud, it will be the main version", "dont
  delete python". `deploy.cmd` and `release-record.ps1` take `studio/dist-go` when it holds the
  site's version, else `dist-protected`; Python code and pipeline unchanged. Version 0.1.15
  (`site.ts`, `version.py`), changelog entry.

## Evidence

Build `compile_go_release.py --version 0.1.15`: exit 0. `ScadaBench-Setup-0.1.15.exe` 13.3 MB
(Python 0.1.14: 24.6 MB), `ScadaBench-Portable-0.1.15.exe` 29.8 MB, installed folder 27 MB
(`ScadaBench.exe` 12.4 MB + `.sig`; 8 workers incl. `studio_goose`, `studio_xsd`; 3 VC++ DLLs).

| Check (built release, scratch data folders) | Result |
|---|---|
| `ScadaBench.exe --self-test` (licence enforced, sealed workers, module signature, real `studio_xsd`) | ok, same JSON as Python, 3.9 s first run |
| One worker byte changed / `ScadaBench.exe` byte changed | refused: "ScadaBench program files were changed or damaged. Reinstall ScadaBench." |
| Portable `--self-test` from an empty folder | ok; first start 7-10.6 s (copy + antivirus scan), next 1.5 s |
| Double-click start (no terminal) | Edge app window "ScadaBench" in 3.4 s; stable 43 s |
| Second start | exits 0 in 0.21 s, running instance kept |
| App window closed | backend gone in 0.13 s, 0 leftover processes |

- The owner's first portable start failed: "could not prepare ... rename ... Access is denied"
  (antivirus holding the freshly copied programs). The portable now retries the rename for up to
  30 s; rebuilt and passed from an empty folder.
- Lane X (`studio_xsd.exe`): 907/907 verdicts identical to lxml (corpus 559, suite 58, edge 250,
  bench 40); `STUDIO_ENGINE=go` pytest 456/104/0 with the real helper. Log:
  [speed-x](../../architecture/go-engine/streams/speed-x.md).
- Lane S (`--self-test`): Go JSON equals Python `check()` byte for byte on the same workers.
- Lane D (`internal/desktop`): 22 tests; event names equal CPython 3.12 for 7 paths.

## Limitations

- Not code-signed (no certificate): SmartScreen warns on first start, as for 0.1.14.
- Npcap still a user install for GOOSE/SV (licence).
- Licence texts of libxml2, zlib and libiconv ship in `_internal/licenses`; libiconv is a
  replaceable DLL with its source published (see Change).
- The Go engine does not yet pass `ExpectedBuild` to the XSD client (a stale helper is not
  refused by build id; the version pin 2.11.9 is still enforced).
- Clean-machine test (no VS, no VC++ redistributable) not run.
- Installer not run on this laptop by the integrator: the owner's installed 0.1.14 shares the
  AppId, so running it upgrades the owner's copy (owner's choice).
