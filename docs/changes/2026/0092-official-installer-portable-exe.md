# 0092: Official installer plus a portable exe that runs directly (0.1.9 build)

Date: 2026-10-18. Decision: [ADR-0063](../../decisions/ADR-0063-official-installer-only.md).

Owner request: "only official installer; portable only without installer", then "exe not
zip, portable exe direct, and official installer only".

## Changes

- **Removed:**
  - the portable self-extractor (`packaging/portable.iss`);
  - the zip tool (`tools/make_portable.py`, `packaging/PORTABLE.txt`, `test_portable_zip.py`).
- **One analysis, two outputs.** `packaging/studio_protected.spec` builds both the installed
  folder and a one-file `ScadaBench-Portable.exe` from the same files.
  `compile_protected.py` names the exe `ScadaBench-Portable-<version>.exe` and signs it when
  signing is set up. The leak check covers both outputs, because they share one analysis and
  one PYZ.
- **`launcher/portable_runtime.py`, used by `Settings.defaults()`.** The one-file exe:
  - keeps its data in `ScadaBench-data` beside it;
  - copies its workers once to `ScadaBench-data/workers/<content hash>/native` and runs them
    from there, so firewall rules keep one program path across starts.
- **Website.**
  - The download page offers the installer and "Portable (no install)" again.
  - The changelog has a 0.1.9 entry.
- **Unchanged:** the installer, its welcome-page art and the build outside the scanned folder.

## Evidence (0.1.9 build, this PC)

| Check | Result |
| --- | --- |
| Leak check | clean (no studio_app source/bytecode, symbols or loose UI) |
| `ScadaBench-Portable-0.1.9.exe` | 29,158,130 bytes, SHA-256 `aa19c1e8…cfdf624a` |
| `ScadaBench-Setup-0.1.9.exe` | 24,400,157 bytes, SHA-256 `24189d89…622e4167` |
| Portable `--self-test` in an empty folder | ok (signed module accepted by the workers, MMS 42.5, arrays, export, recovery) |
| Portable start on port 18767 | ready in 4 s; UI 200, session 200 |
| Portable runs an IED | `open_substation.scd` IED1_XCBR running on 127.50.0.2:102; `studio_ied.exe` runs from `ScadaBench-data\workers\530a52e70e46e49f\native` |
| Portable restart | project kept; same worker folder reused |
| Installed folder `--self-test` | ok |
| verify.py, unit + integration | PASS; 390 passed |

The first build put the workers in `ScadaBench-data/runtime`, where the app also writes IED
`.cfg` files. They were moved to `workers/`, and a test now guards this.

## Published (owner-approved hosting)

- **Hosting.** The portable exe is 27.8 MiB, over the 25 MiB Cloudflare Pages file limit. The
  IEC schema package is shipped unmodified under its licence, so it is not trimmed. With the
  owner's approval, the exe is served from the Cloudflare R2 bucket `scadabench-downloads` on
  the custom domain `download.scadabench.com`.
- **`deploy.cmd`:**
  - uploads the newest `ScadaBench-Portable-*.exe` there (attachment download);
  - checks that the download page links it;
  - redirects `/download/ScadaBench-Portable.exe` and older portable names to it.
- **Live check:**
  - The download page links `ScadaBench-Setup-0.1.9.exe` and the R2 portable URL.
  - The old links redirect with 302.
  - Both downloaded files match the build SHA-256 above.
  - Windows' own web client gets 200 with the full size.

## Limitations

- **Certificate revocation check.** Right after the certificate was issued, curl's strict
  (schannel) revocation check failed for `download.scadabench.com`. This PC already needs
  `--ssl-no-revoke` for curl.
- **Not tested yet:**
  - a clean PC, an antivirus other than this PC's, or code signing (unsigned);
  - a firewall/lab-mode run from the portable exe (no UAC run this time).
- **Slower start.** The one-file exe unpacks about 48 MB to `%TEMP%` on each start (4 s here).
