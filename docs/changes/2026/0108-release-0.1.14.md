# 0108: Release 0.1.14 published (installer and portable)

Date: 2026-10-19. Owner request: "make version 0.1.14, deploy to internet".

## Contents (since the published 0.1.13)

- [0105](0105-ux-delete-picker-loading-goose-updates.md) easier IED delete and multi-delete,
  pickers that keep their place, loading indicators, GOOSE corner mark, quiet update check (ADR-0071).
- [0106](0106-partial-run-states.md) stopped, partly running and fully running never look alike.

## Change

- `website/src/data/site.ts`: `VERSION` 0.1.13 → 0.1.14; `studio/backend/studio_app/version.py`
  `0.1.14+dev` (the test keeps both equal).
- `website/src/pages/changelog.astro`: 0.1.14 entry in plain words; the 0.1.13 heading is now literal.
- Built with `compile_protected.py --version 0.1.14`; published with `website/deploy.cmd`.
- `license-server/deploy/owner.ps1 -Action release`: the record is now sent base64-encoded on the
  ssh command line. Windows PowerShell put a UTF-8 BOM in front of the piped JSON, and the server
  refused it ("release record must be JSON with version, date, notes, page and installer").

## Evidence

| Check | Result |
| --- | --- |
| Build log | native enforcement on; leak check "no studio_app source, bytecode, symbols or loose UI files"; unsigned |
| `ScadaBench-Setup-0.1.14.exe` | 24,597,843 B, SHA-256 `c6e9683f…fb852b` |
| `ScadaBench-Portable-0.1.14.exe` | 29,469,986 B, SHA-256 `5b559901…769fde0` |
| `--self-test` (portable in an empty folder, and the folder build) | both ok (MMS 42.5, arrays, export, recovery, deletion, saved panel and pulse) |
| Deploy | installer on Pages (deployment 66609afb), portable uploaded to R2 |
| Live site | `/download/` links both 0.1.14 files; `/changelog/` shows 0.1.14; both downloaded files match the SHA-256 above |
| BOM cause | the same record piped from PowerShell to a local `license-server release` failed the same way; the bytes start with `EF BB BF` |
| `GET https://license.scadabench.com/api/v1/release` | before publishing: 404 `no_release`. After the owner said yes, `owner.ps1 -Action release` saved 0.1.14; the answer is 200, kid `768dc76e`, `Cache-Control: public, max-age=300`, with an ETag |
| App `UpdateService.check` against the live server | with current 0.1.13: signature accepted, `available: true`, latest 0.1.14 with the hashes above; with current 0.1.14: `available: false` |

## Limitations

- The automatic release step in `deploy.cmd` failed on the BOM. The record was published afterwards,
  once the owner said yes. Apps 0.1.13 and older have no update check; only 0.1.14 and later hear
  about new versions.
- The server still rejects a BOM itself. A parser that tolerates it waits for the next license-server
  change, because another session is working in `license-server/`.
- Unsigned; SmartScreen warns. Not installed on a clean PC by this change.
- Pages holds only the current installer, so `/download/ScadaBench-Setup-0.1.13.exe` now answers 404.
  The unversioned link redirects to 0.1.14.
