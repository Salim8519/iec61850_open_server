# 0100: Test build 0.1.12 and VM retest of change 0099

Date: 2026-10-19. Owner request: "make the new installer build to test", then "install the new
version in the vm" and run the tests again. Not published to the website.

## Build

- Commit `818bc91` (change 0099) on master, built with
  `studio/.venv/Scripts/python.exe studio/tools/compile_protected.py --version 0.1.12`.
  The first try used the system Python (no Flask) and stopped at `pack_frontend.py`.
- Version 0.1.12, not 0.1.11: an unrecorded `ScadaBench-Setup-0.1.11.exe` from 19:02 (before
  this fix) was already in `studio/dist-protected`; reusing the number would give two
  different builds one name.

| Check | Result |
| --- | --- |
| Native workers | license enforcement ON |
| Leak check | no studio_app source, bytecode, symbols or loose UI files |
| `ScadaBench-Setup-0.1.12.exe` | 24,561,980 B, SHA-256 `7860a56a…28ce2b` |
| `ScadaBench-Portable-0.1.12.exe` | 29,413,004 B, SHA-256 `fd5d4cbc…6b6a99` |
| One-folder app and portable (empty folder) `--self-test` | both ok; the backend requires the worker capability `standard_controls`, so an old worker would have been refused |
| Packaged app in lab mode (VMnet8 alias 192.168.98.27:18190, own API with its printed session) | loopback 127.50.0.2 → `error 5`; Operate Close `{"ok": true}` in 0.14 s, independent read `10`; Operate Open in 0.12 s, read `01` |

## VM retest (VMware VM 192.168.230.20, GE Viewer + Concentrator on the host)

- Installer copied through the RDP clipboard; SHA-256 in the VM matches. It upgraded the
  per-user install in place (no UAC); uninstall entry shows 0.1.12; `--self-test` ok in the VM.
- Start: "Project upgraded to format 15. Every command follows standard IEC 61850." No CONCEN
  notice, because CB-1 was already on standard and CB-2's command was left unmapped (0099).
- ScadaBench Operate on CB-1 in lab mode: "Close done", CB-1 Closed; "Open done", CB-1 Open;
  no "MMS control failed" (0.1.10 showed it for the same press).
- GE Viewer, PROCEED latched with `PUSH_BUTTON-2` (`Ind008`) and `PUSH_BUTTON-4` (`Ind012`):

| Command (capture `case-6-0112-retest.pcapng`, host VMnet8) | Wire | Result |
| --- | --- | --- |
| Y755 CLOSE | 23:59:09.25 `SBOw` → success; report `Pos` still `01` with selection true; 23:59:19.82 `Oper` ctlVal=true, orIdent `CONCEN` → success, CommandTermination+, report `Pos` `10` | Viewer "Command Sent OK", Y755 green (closed) |
| Y755 OPEN | 00:01:42.69 `SBOw` → success; `Pos` still `10`; 00:01:50.53 `Oper` ctlVal=false → success, CommandTermination+, report `Pos` `01` | Viewer "Command Sent OK", Y755 open |

ScadaBench's own Operate runs inside the VM (probe to its own lab address), so it is not on the
host capture; its result is the app message and the SLD position above.

## Limitations

- Unsigned; not published; not installed on a clean PC.
- Latching push buttons are not restored at run start (only CB/DS/ES start positions are), so
  the GE PROCEED buttons must be pressed again after every start.
- CB-2's command is still unmapped on the VM (its original target was CB-1's control).
- Captures stay in the working folder (customer model names); not committed.
