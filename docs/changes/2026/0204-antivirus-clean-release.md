# 0204: Kaspersky quarantined the 0.1.30 installer; release rebuilt clean with an antivirus gate

Owner: "first time ever, the installer got quarantined by kaspersky, imagine what our client would
think", "virus total flagged us here too" (Bkav Pro `W32.Malware.8A1A93B8`), "deploy 0.1.30 clean
again later". Decision: [ADR-0130](../../decisions/ADR-0130-antivirus-clean-release.md).

## Diagnosis (owner's PC, Kaspersky Total Security 21.3, bases 2026-10-30 11:44)

- Both browser downloads of `ScadaBench-Setup-0.1.30.exe` (sha 322016ce…, the published file) went
  to Kaspersky's quarantine at 00:28 and 00:30 on 2026-10-31 (quarantine store entries of
  16,916,690 and 16,916,708 bytes). The Kaspersky event log holds no detections, and the report
  and quarantine files cannot be read without administrator rights.
- `avp.com SCAN /i0` of a copy in %TEMP% (the project folder is excluded from scanning on this
  PC): `setup.exe//data0028 suspicion VHO:Trojan.Win64.Crypt.gen`, 100 other objects ok. The
  same scan of the 17 programs of the installed folder named only
  `_internal/native/scadabench_labnet.exe`. Clean: the 0.1.29 installer, the installed 0.1.29
  `scadabench_labnet.exe`, and the portable exes for 0.1.29 and 0.1.30.
- The 0.1.30 lab program source, built five ways: flagged with `-H windowsgui -s -w`, `-s` alone,
  `-w` alone; clean as a console program (`-s -w`) and as `-H windowsgui` with symbols and debug
  information kept. A stripped Go GUI program sits on the edge of this heuristic; the 0.1.30 code
  change (ADR-0128) moved it over.
- Microsoft Defender is not running on this PC (Kaspersky is the antivirus), so it was not used.

## Change

- `studio/tools/build_labnet.py`: `-ldflags "-H windowsgui"` (no `-s -w`).
- `studio/tools/antivirus_gate.py` (new): copies programs to a fresh %TEMP% folder, scans them
  report-only with Kaspersky and, when active, Defender; a file removed by real-time protection
  counts as a detection. `avp.com` writes no report when the report file name holds a hyphen.
- `studio/tools/compile_go_release.py`: final step "Antivirus check" on the installed folder, the
  portable exe and the installer; findings rename both outputs `*.flagged` and fail the build;
  `--allow-no-scanner` for a PC without an antivirus.
- `studio/tests/unit/test_antivirus_gate.py`: report parsing, no-scanner rule, lab build flags.

## Evidence

- `test_antivirus_gate.py` 4/4. The gate on the published 0.1.30 build: 17 programs scanned, 2
  findings (`scadabench_labnet.exe` and `setup.exe//data0028`, both VHO:Trojan.Win64.Crypt.gen),
  exit 1; on the new lab program: 0 findings.
- Release 0.1.30 rebuilt (owner: "deploy 0.1.30 clean again"): final step "18 programs scanned
  with Kaspersky, 0 findings". Installer 5aa417af (18,150,963 bytes), portable 90ea1f7b
  (49,066,496 bytes), `scadabench_labnet.exe` e5d0408d; portable self-test ok.
- Deployed with `website/deploy.cmd`: Pages, R2 portable and signed release record 0.1.30. The live
  `/download/ScadaBench-Setup.exe`, `/download/ScadaBench-Portable.exe` and the R2 portable link
  serve exactly these files; a Kaspersky scan of the downloaded copies in %TEMP%: 0 detections,
  files left in place.

## Limitations

- The version stays 0.1.30 by the owner's choice: a PC that installed the first 0.1.30 build is
  not offered this one. Kaspersky quarantined that installer before it ran here, so it is unlikely
  to be installed anywhere.
- One PC, one antivirus, one bases date. Heuristics change with new bases; the gate rechecks every
  release, but files already published are not rescanned. Defender was not available to test the
  gate's Defender path (it is not running on this PC).
- VirusTotal's Bkav Pro verdict was not rechecked: the files were not uploaded to VirusTotal.
- The programs are unsigned; a code-signing certificate would reduce heuristic verdicts further.
