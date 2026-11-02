# 0211: Listen-check messages name what Windows answered; the lab card's DNS step may fail

Owner: a friend's laptop showed "the card's settings could not be set: ... Set-DnsClient ...
HRESULT 0x80041010" and "C06_BCU cannot listen on 127.50.0.2:102 because another program is using
it ([WinError 10013] ...)" with no usable log; "fix ... without generalizing the issue to all",
"make sure the app is not behaving different than now". Decision:
[ADR-0134](../../decisions/ADR-0134-clear-listen-and-lab-card-failures.md).

## Change

| Area | Files |
| --- | --- |
| Port holder lookup (read only) | `studio/engine/internal/portholder/` (new: `portholder.go`, `portholder_windows.go`, `portholder_other.go`, tests) |
| Listen-check message and run-log event | `engine/internal/runtime/listen_problem.go` (new), `endpoints.go` (`bindTest` returns the raw error), `supervisor.go` (`prepareNetwork` emits `listen_check_failed`), `listen_problem_test.go` |
| Lab card DNS step | `engine/internal/labnet/card_script.go` (new: script with the DNS step in `try/catch`, `dnsNote`), `install_windows.go` (`Installed.Note`), `cmd/scadabench_labnet/main.go` (note added to the result message), `card_script_test.go`, `card_script_windows_test.go` |

## Evidence (owner's PC)

- Holder matrix (Python listener on port 102 in each socket mode, both binds tried on
  127.50.0.2:102, then a client connect): the preflight refuses exactly where the IED's own bind
  fails or another program would take its clients; see the ADR table. The check was kept.
- The real preflight against a real holder:
  - 0.0.0.0 exclusive: "C06_BCU cannot listen on 127.50.0.2:102 because python.exe (process 63920)
    holds port 102 on every address of this PC and lets no other program use it ([WinError 10013]
    ...)", event with holder pid, program and address 0.0.0.0;
  - 0.0.0.0 default: passes (as before);
  - 127.50.0.2 default or reuse: "... because python.exe (process N) is using it ([WinError 10048]
    ...)";
  - no holder: passes;
  - lab mode, address not on the PC: "... because this address is not usable on this PC right now
    ([WinError 10049] ...). If it is on a network card, check that the card is enabled and its
    cable is plugged in."
- With no holder found, the 10048 message is identical to the old one (test).
- Windows PowerShell runs the card script's DNS guard with a failing `Set-DnsClient` (an interface
  that does not exist): the reason comes out, the next step runs and the script succeeds; an
  unguarded failing step still stops it. No card was changed by the test.
- `go test ./...` and `python studio/tools/verify.py`: see the ledger entry.

## Limitations

- The friend's PC was not tested; the 0x80041010 class error was not reproduced here (the guard was
  tested with another `Set-DnsClient` failure).
- A holder running as another user or a protected service may show as "the program with process id
  N" (Windows does not give its name without administrator rights); Windows' own listeners show as
  "Windows (System)".
- `netcfg.Preflight` (not used by Run) keeps the old Python-identical text.
