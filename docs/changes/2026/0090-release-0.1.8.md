# 0090: Release 0.1.8, branded installer, versioned downloads, website publish

Date: 2026-10-18. Contents: changes [0088](0088-scada-master-converter.md) (SCADA master and
protocol converter) and [0089](0089-two-machine-scada-test.md) (two-machine test, gateway
firewall, fixes). 0.1.7 was only a VM test build and was never published.

## Change

- **Branded installer.** New `studio/tools/build_installer_art.py` draws the welcome panel (logo,
  name, tagline, protocols) and the small header icon with Edge in headless mode, then writes the
  BMP sizes Inno Setup uses for 100–250 % display scaling into `studio/branding/installer/`.
  - `installer.iss` and `portable.iss` use them, with `DisableWelcomePage=no`. Inno Setup 6
    skips the welcome page by default, so the picture would otherwise only show on the last page.
- **Builds outside the scanned folder.** The antivirus locked every icon update in
  `studio/dist-protected`: 20 portable attempts failed, and one installer attempt deleted the
  setup it was replacing. `compile_protected.py` and `make_portable.py` now let Inno Setup write
  into a temp folder and move the finished exe in. The fix is tested in
  `test_protected_packaging.py`; with it, the portable built on the first try.
- **Versioned downloads.**
  - `website/src/data/site.ts` builds the links from the version
    (`/download/ScadaBench-Setup-0.1.8.exe`, `/download/ScadaBench-Portable-0.1.8.exe`).
  - `deploy.cmd` publishes the files under their own names. It stops if the download page links
    a different version, and writes `_redirects` so the old unversioned links go to the new files.
- **Website text.**
  - Version 0.1.8 and a 0.1.8 changelog entry. It names only the tools that were observed in
    0089.
  - The gateway feature text now covers the master, the converter and the firewall step.
- **Build note.** The Astro build needs Node 22.22 or newer. `deploy.cmd` works with the Node in
  `%LOCALAPPDATA%\pi-node`; the Node in `C:\Program Files\nodejs` (22.11) is too old.

## Evidence

| Artifact | Size | SHA-256 |
| --- | --- | --- |
| `ScadaBench-Setup-0.1.8.exe` | 24,387,160 B | `13778450…6107f80d` |
| `ScadaBench-Portable-0.1.8.exe` | 24,387,606 B | `5df36bd3…5ad29477` |
| `ScadaBench-0.1.8-Portable.zip` (local only, over the Pages limit) | 29,009,578 B | `b7b0f8ff…6551ae07` |

- **Release build checks.**
  - Leak check: no `studio_app` source, bytecode, symbols or loose UI files.
  - The sealed version is 0.1.8, and `--self-test` reports ok.
- **Portable.**
  - Silent unpack: exit 0, the folder holds `ScadaBench.exe`, `_internal`, `data` and
    `PORTABLE.txt`, and there is no uninstall key.
  - Its `--self-test` reports ok. Started from the unpacked folder, it wrote `launcher.json` into
    its own `data` folder.
- **Installer.** The welcome page shows the ScadaBench panel and "This will install ScadaBench
  version 0.1.8", with the brand icon on the following pages. It was closed without installing.
- **Live site.**
  - `/`, `/features/`, `/changelog/` and `/download/` all return 200.
  - The download page shows Version 0.1.8 and links both versioned files, and both live files
    match the SHA-256 values above.
  - The old unversioned links return 302 to the 0.1.8 files.
- **Tests.** The packaging tests pass (14).

## Limitations

- Unsigned, so Windows SmartScreen warns. No code-signing certificate yet.
- Not yet installed on a clean PC. The owner's installed copy was not updated by this release.
- Antivirus locks still slow the PyInstaller resource step. It retries by itself; that is not
  changed here.
