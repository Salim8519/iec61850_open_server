# ADR-0083: Go release packaging (installer + portable exe on the Go engine)

Accepted: 2026-10-20. Owner: "make production pipeline for go, it gives exe installer and exe
portable too", then "go exe installer". Amends [ADR-0074](ADR-0074-go-engine.md) (Python stays
the default) for releases built by the new pipeline, and keeps
[ADR-0063](ADR-0063-official-installer-only.md) (one installer + one portable exe) unchanged.
Number reserved in [architecture](../architecture/go-engine/architecture.md) ("0083 Go launcher
and dual-engine packaging"); dual-engine packaging was dropped, see Alternatives.

## Problem

Releases (0.1.14) still ship the Python backend (PyInstaller + Nuitka, ~55 MB unpacked, Python
start ~1 s). The composed Go server matches Python (pytest 456/104/0 on both, golden replay 0
diffs, 2 h soak 0 errors; change 0138) and is 3x faster with ~20x less backend CPU, but it has no
packaged form: no app window, no single-instance hand-off, no `--self-test`, no portable mode.

## Decision

1. **New pipeline** `studio/tools/compile_go_release.py` (one command, owner-run). Output
   `studio/dist-go/`: `ScadaBench/` folder, `ScadaBench-Setup-<v>.exe`,
   `ScadaBench-Portable-<v>.exe`. The Python pipeline (`compile_protected.py`) stays as is and
   remains the fallback release.
2. **Same inputs as the Python release:** frontend build packed to `frontend.pak`, licensed native
   workers (`-DSTUDIO_ENFORCE_LICENSE=ON`, enforcement checked), `studio_goose.exe`,
   `studio_xsd.exe` (`tools/build_xsd_helper.py`, pinned libxml2 2.11.9), schemas, README,
   licence files, release keys from `configure_release.py`. The workers' VC++ runtime
   (`msvcp140`, `vcruntime140`, `vcruntime140_1`) is copied from the MSVC redistributable into
   `_internal/native` so a clean PC needs no extra install (owner: "one click and all running");
   Npcap stays a user install (its licence forbids bundling; workers delay-load it).
   GNU libiconv (LGPL-2.1, used by `studio_xsd.exe`) ships as the replaceable
   `libiconv-2.dll` with `COPYING.LIB`, a notice naming its source, and the unmodified source
   archive on `download.scadabench.com/source/` (uploaded by `website/deploy.cmd`).
8. **Main version.** `website/deploy.cmd` and `release-record.ps1` publish the Go release
   (`studio/dist-go`) when it holds the site's version, else the Python release.
3. **ScadaBench.exe is `cmd/studio_server`** built with `CGO_ENABLED=0 -trimpath`,
   `-H windowsgui -s -w`, `-X main.packaged=1`, `-X main.version=<v>` and the native integrity
   seal (`-X main.nativeSeal=<name=sha256,...>`, the Python `integrity.EXPECTED`).
4. **Same folder layout** as the Python install: `ScadaBench.exe` + `_internal/` (`native/`,
   `frontend.pak`, `schemas/`, `licenses/`, `README.md`). The installer (`installer.iss`, same
   AppId) removes an old `_internal` before copying, so a Go install cleanly replaces a Python one
   and back. Data and licence locations are unchanged.
5. **Same packaged behaviour** in Go (`internal/desktop`): Edge/Chrome `--app` window with its own
   profile, default browser fallback, stop when the window closes (profile hand-off rule),
   single instance per data folder via `Local\ScadaBench-Show-<sha256[:24]>`, focus of the
   recorded window, message boxes when there is no terminal, log file
   `data/logs/launcher-<date>.log`, terminal attach for command-line use, the
   SubstationStudio -> ScadaBench folder move, `--dev` refused, and `--self-test` (temporary data,
   bundled schema, independent loopback MMS read).
6. **Module signature:** the build signs SHA-256 of `ScadaBench.exe` with
   `native/release/module_signing.seed` into `ScadaBench.exe.sig`; the packaged engine sets
   `STUDIO_MODULE_PATH`/`STUDIO_MODULE_SIG` only when the signature exists (as Python did for the
   `.pyd`). Code signing (`STUDIO_SIGN_PFX`) runs before the module signature.
7. **Portable exe** (`cmd/scadabench_portable`, windowsgui): embeds the exact installed folder
   (same bytes, including `ScadaBench.exe.sig`), copies it once to
   `ScadaBench-data/app/<content hash>/` beside the portable exe (built under a temporary name and
   renamed; reused when complete) and runs that `ScadaBench.exe --data-dir ScadaBench-data`,
   passing arguments, terminal and exit code. Workers therefore keep a stable path per build
   (firewall rules, ADR-0063) and the installed and portable apps run identical binaries.

## Compatibility

- Same AppId, data folders, licence file and project formats: upgrade from and downgrade to a
  Python release keep projects and licence. A Go run may leave an autosave journal that only Go
  folds; the background snapshot normally writes `autosave.subsim` within seconds.
- Python portable copies used `ScadaBench-data/workers/<hash>/native`; the Go portable uses
  `ScadaBench-data/app/<hash>/` and leaves the old folder untouched.
- No Python fallback inside a Go release (choose the release instead). Npcap still not bundled.

## Alternatives

- Dual-engine package (Go + Python behind `SCADABENCH_ENGINE`): about 3x the size and two
  backends to protect; rejected by the owner's request for a Go exe.
- Go portable running in-process with an embedded payload: the exe cannot carry its own module
  signature, and installed and portable binaries would differ; rejected.
