# ADR-0063: One official installer; portable = one exe that runs directly

Accepted: 2026-10-18. Owner: "portable not allowed anymore, only official installer; portable
only without installer is allowed", then "exe not zip, portable exe direct, and official
installer only". This replaces the self-extractor and zip parts of
[ADR-0060](ADR-0060-portable-version.md). The sealed-build rules of ADR-0060 stay.

**Decision**

- **One installer.** `ScadaBench-Setup-<version>.exe` (Inno Setup, `packaging/installer.iss`) is
  the only installer.
- **No portable installer, no zip.** The self-extractor (`packaging/portable.iss`) and the zip
  tool (`tools/make_portable.py`, `packaging/PORTABLE.txt`) are removed.
- **The portable form is one exe that runs ScadaBench.** `ScadaBench-Portable-<version>.exe` is
  a PyInstaller one-file exe made by the same `studio_protected.spec` analysis as the installed
  folder, so both hold exactly the same sealed files (compiled `.pyd` and its signature, packed
  UI, licensed workers). Double-clicking it starts ScadaBench; there is no wizard and nothing is
  installed in Windows.
- **Everything stays beside the exe.** Projects, logs and the license live in
  `ScadaBench-data` next to the exe (`launcher/portable_runtime.py`).
- **Workers run from a stable path.** A one-file exe unpacks into a new random temp folder on
  every start. Windows firewall rules name the program path, so the workers are copied once to
  `ScadaBench-data/workers/<content hash>/native` and run from there. The same build keeps the
  same path (no new firewall rule or UAC prompt per start), and a new build gets its own folder.

- **Hosting.** The exe is over the 25 MiB Cloudflare Pages file limit. It is served from the R2
  bucket `scadabench-downloads` at `download.scadabench.com`. `website/deploy.cmd` uploads it,
  and old portable links redirect to it.

**Compatibility**

- Copies unpacked by the 0.1.6–0.1.8 portable installers keep working with their `data` folder.
- The one-file exe starts more slowly than the installed app, because it unpacks about 55 MB
  on each start.
- Antivirus programs check one-file exes more strictly; the exe is not code-signed yet.
