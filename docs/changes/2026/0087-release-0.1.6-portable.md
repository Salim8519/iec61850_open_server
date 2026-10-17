# 0087: Release 0.1.6 with installer and portable version, website publish

Date: 2026-10-17. Decision: [ADR-0060](../../decisions/ADR-0060-portable-version.md).
Contents: changes [0086](0086-separate-run-running-dots.md) (separate Run, running dots, page
commands, taller point list) plus the portable version.

## Change

- `compile_protected.py --version 0.1.6`: the leak check is clean. Kaspersky blocked the
  installer step again; the manual Inno Setup retry succeeded on the 4th try.
- New `studio/tools/make_portable.py`, `packaging/portable.iss`, `packaging/PORTABLE.txt` and
  `tests/unit/test_portable_zip.py`.
- Website:
  - version 0.1.6 and a 0.1.6 changelog entry;
  - the features text no longer says the gateway starts the IEDs;
  - the download page has a "Portable version" button and note;
  - `deploy.cmd` also publishes the newest portable exe.

## Evidence

| Artifact | Size | SHA-256 |
| --- | --- | --- |
| `ScadaBench-Setup-0.1.6.exe` | 23,680,875 B | `54da4410…c3ed1762` |
| `ScadaBench-Portable-0.1.6.exe` | 23,681,127 B | `cebef935…b5f048cf` |
| `ScadaBench-0.1.6-Portable.zip` (local only, over the Pages limit) | 28,384,841 B | `425a68f5…6f4ab612` |

- **Portable exe, silent unpack to `%TEMP%\sbp`:**
  - exit 0; the folder holds `ScadaBench.exe`, `_internal`, an empty `data` folder and
    `PORTABLE.txt`;
  - no uninstall registry key;
  - the longest path is 113 characters after the base folder.

  A first try in a 150-character scratch folder failed on the Windows 260-character path
  limit. That came from the test folder, not the package.
- **The unpacked portable app:**
  - `--self-test` ok;
  - started without `--data-dir`, it wrote `launcher.json` into its own `data` folder;
  - health 0.1.6, and the owner's license is accepted.
- **End to end with the independent 104 master:**
  - gateway Start leaves Run "stopped";
  - after Run, a general interrogation returned 11 objects, all good;
  - select/execute was confirmed (ACT_CON ×2, then ACT_TERM), and the position came back
    spontaneously as 2;
  - the log shows "is now closed (ON)", and a page command returned ok.

  The installed 0.1.5 app held 127.50.0.2:102, and 0.1.6 refused Run with a clear message;
  the test IED was moved to port 10102.
- **Live:** `/`, `/features/`, `/faq/`, `/changelog/` and `/download/` all return 200. The
  download page shows Version 0.1.6 and the "Portable version" button. Both live downloads
  match the SHA-256 values above.

## Limitations

- Unsigned (SmartScreen). The owner's installed copy was not updated by this release.
- The portable zip is not on the website (over 25 MiB). It is in `studio/dist-protected` for
  direct sharing.
