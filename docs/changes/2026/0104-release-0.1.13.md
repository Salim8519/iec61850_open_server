# 0104: Release 0.1.13 published (installer and portable)

Date: 2026-10-19. Owner request: the protected installer and portable 0.1.13 are built; put them on
the server for everyone to download.

## Contents (since the published 0.1.10)

- [0095](0095-explorer-reconnect.md) IED Explorer reconnect fix.
- [0097](0097-clean-sld-test-buttons-run-one.md) clean SLD, equipment test buttons, run or connect one IED.
- [0098](0098-publish-wording-one-page-network.md) role wording, reachability, one-page network setup.
- [0099](0099-standard-commands-lab-operate.md) standard IEC 61850 commands, Operate reaches lab-mode IEDs.
- [0102](0102-goose-visibility-and-timing.md) GOOSE visibility and retransmission timing.
- Test builds 0.1.11/0.1.12 were never published; 0.1.13 was retested host ↔ VM in
  [0103](0103-goose-retest-0.1.13.md).

## Change

- `website/src/data/site.ts`: `VERSION` 0.1.10 → 0.1.13 (download links and changelog title).
- `website/src/pages/changelog.astro`: 0.1.13 entry in plain words; the 0.1.10 heading is now literal.
- Published with `website/deploy.cmd`, which ships the files of the `site.ts` version (0103).

## Evidence

| Check | Result |
| --- | --- |
| `ScadaBench-Setup-0.1.13.exe` | 24,558,570 B, SHA-256 `59dd1fb5…7f1f8a` (built 2026-10-19 02:16) |
| `ScadaBench-Portable-0.1.13.exe` | 29,413,157 B, SHA-256 `59bd35ff…b97824` |
| Portable `--self-test` in an empty folder | ok (MMS 42.5, arrays, export, recovery, deletion, saved panel and pulse) |
| Deploy | installer on Pages (deployment 737ea480), portable uploaded to R2 |
| Live site | `/download/` links both 0.1.13 files; `/changelog/` shows 0.1.13; unversioned setup and portable links 302 to 0.1.13; both downloaded files match the SHA-256 above |

## Limitations

- Unsigned; SmartScreen warns. Not installed on a clean PC by this change (0103 used it on host and VM).
- Pages holds only the current installer, so the old `/download/ScadaBench-Setup-0.1.10.exe` now
  answers 404 (the unversioned link redirects to 0.1.13); old portable names redirect to 0.1.13.
- Open finding from 0103: reachability is shown from the project, not from Windows, so lost lab
  aliases are noticed only when Run fails.
