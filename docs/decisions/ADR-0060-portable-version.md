# ADR-0060: Portable version from the same sealed build

Accepted: 2026-10-17. Owner request: "make portable and installer versions too".

**Decision**
- The portable version is the protected app folder from `compile_protected.py`, never the
  unprotected preview from `build_portable.py`. It has the same licence checks, sealed workers
  and leak check as the installer.
- **Portable mode is the existing rule.** A `data` folder next to `ScadaBench.exe` keeps
  projects and settings in that folder (`Settings.packaged_data_directory`). The license stays
  per machine.
- `studio/tools/make_portable.py` makes two forms:
  - `ScadaBench-<v>-Portable.zip`: the folder with an empty `data/` and `PORTABLE.txt`, for
    USB sticks and copying by hand;
  - `ScadaBench-Portable-<v>.exe`: an Inno Setup self-extractor (`packaging/portable.iss`)
    that unpacks the same folder where the user chooses. It adds `data/` and writes no
    registry key, shortcut or uninstaller. Choosing the same folder again updates it and keeps
    the projects.

  The tool refuses a folder that holds user data or source files. It retries Inno Setup
  because antivirus scanners briefly lock new setup files.
- **The website offers the self-extractor.** The zip (27.1 MiB for 0.1.6) is over Cloudflare
  Pages' 25 MiB file limit; the LZMA self-extractor (22.6 MiB) is not. `deploy.cmd` publishes
  it as `/download/ScadaBench-Portable.exe`.

**Compatibility**
- No app behaviour change. Installed copies are unaffected. A portable copy and an installed
  copy can live side by side, each with its own data.
- Both forms are unsigned, like the installer (SmartScreen "More info → Run anyway").
