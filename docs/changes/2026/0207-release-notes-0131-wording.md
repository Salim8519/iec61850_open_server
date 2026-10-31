# 0207: 0.1.31 release notes rewritten for customers

Owner: "this is professional software not noob laptop guy trying out stuff". The first 0.1.31
changelog entry described the owner's own test session (a Wi-Fi card losing its connection, the
antivirus product and its verdict by name). Its title is also the `notes` line of the signed
release record, so every installed app's update notice showed it.

## Change

| Area | Files |
| --- | --- |
| Changelog | `website/src/pages/changelog.astro`: title "More reliable and faster network setup"; entries ordered New, Faster, Improved, Fixed, Upgrading; no vendor or verdict names, no test-session story; the Repair fix for a running service is listed |

No product code, installer or portable changed; the release files and their hashes stay those of
change 0206.

## Evidence

- `npm run build`: 11 pages; the built changelog shows the new 0.1.31 entry.
- `website/deploy.cmd`: Pages deployment f0dc2e69, the same installer and portable uploaded again,
  "Saved release 0.1.31 from 2026-11-01".
- Live `https://scadabench.com/changelog/` shows the new entry; none of "Kaspersky", "PDM:Trojan",
  "lost Wi-Fi" is on the page. The signed record at `license.scadabench.com/api/v1/release` reads
  `notes: "More reliable and faster network setup"` with hashes f5847187 / 562fb53b.
- `/download/ScadaBench-Setup.exe` and `/download/ScadaBench-Portable.exe` still resolve to the
  0.1.31 files (19,556,057 and 53,095,936 bytes).

## Limitations

- Apps that already fetched the 0.1.31 record keep the old line until their next daily check.
