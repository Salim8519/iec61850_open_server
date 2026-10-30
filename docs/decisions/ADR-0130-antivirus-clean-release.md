# ADR-0130: Releases pass the antivirus; the lab program keeps its symbols

Accepted: 2026-10-31. Owner: "first time ever, the installer got quarantined by kaspersky, imagine
what our client would think", "virus total flagged us here too", "deploy 0.1.30 clean again".
Change [0204](../changes/2026/0204-antivirus-clean-release.md).

## Problem

Kaspersky Total Security 21.3 (bases 2026-10-30) quarantined both downloads of
`ScadaBench-Setup-0.1.30.exe`. A report-only scan (`avp.com SCAN /i0`) of a copy outside the
project folder named one object inside the installer, `scadabench_labnet.exe`, as
`VHO:Trojan.Win64.Crypt.gen`. That is a heuristic for packed or encrypted code, not a known threat.
The 0.1.29 program was not flagged, though it was built with the same flags; neither were the
other 16 programs, the portable exe or the 0.1.29 installer. The 0.1.30 source of the same program,
built five ways and scanned:

| `-ldflags` | Size | Kaspersky |
| --- | --- | --- |
| `-H windowsgui -s -w` (release) | 2.66 MB | VHO:Trojan.Win64.Crypt.gen |
| `-H windowsgui -s` | 2.66 MB | VHO:Trojan.Win64.Crypt.gen |
| `-H windowsgui -w` | 2.85 MB | VHO:Trojan.Win64.Crypt.gen |
| `-s -w` (console program) | 2.66 MB | clean |
| `-H windowsgui` (symbols and debug information kept) | 4.00 MB | clean |

Nothing in the release pipeline scanned what it published, and the Windows build PC excludes the
project folder from Kaspersky, so the build PC never noticed.

## Decision

1. `scadabench_labnet.exe` is built with `-H windowsgui` only: same program, same window behaviour,
   about 1.3 MB larger, and clean in Kaspersky.
2. `compile_go_release.py` ends with an antivirus check (`studio/tools/antivirus_gate.py`): every
   `.exe`/`.dll` of the installed folder, the portable exe and the installer are copied to a fresh
   `%TEMP%` folder and scanned report-only with Kaspersky (`avp.com`) and, when it is the active
   antivirus, Microsoft Defender. A file that real-time protection removes counts as a detection.
   On any finding the portable exe and installer are renamed `*.flagged`, so `website/deploy.cmd`
   cannot publish them, and the build fails with each file and verdict. With no scanner the build
   fails unless `--allow-no-scanner` is given.

## Compatibility

No product, protocol or data change. The other programs keep `-s -w`; the gate catches a future
heuristic hit on any of them. Code signing (already supported through `STUDIO_SIGN_PFX`) would
lower such heuristics further, but it needs a certificate the owner has not bought. One engine on
VirusTotal (Bkav Pro, `W32.Malware.*`) is outside what this PC can check.
