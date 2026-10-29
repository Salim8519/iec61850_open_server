# 0201: Lab addresses follow Start and Stop

Owner: "in the scadabench lab, the ip will stick there even if u did not click start", then
"ok check and confirm" and "if all ok, do the fix, deploy version 0.1.29".
Decision: [ADR-0127](../../decisions/ADR-0127-lab-addresses-follow-the-run.md).

## Change

- `labsetup/release.go`: `Release` removes every card address no running IED and no gateway or
  master own address uses. `Use` stores only; `SetAddress` no longer adds or removes itself;
  `EnsureForRun` adds only the IEDs being started.
- `runtime`: `Options.LabRelease` is called after a full Stop, a partial Stop (`forget`) and a
  failed Run; `LabAddresses` gets the ids being started.
- `app`: release hook wired; `releaseLeftLab` at launch under the cycle lock. Exit already stops
  the run, which releases.
- Frontend: Ping hint "Press Start, then ping…", Use and edit hints, summary "Set: …" while stopped.
- Spec NET-17 updated.

## Evidence

- `go test ./internal/labsetup` (new: Use puts nothing on the card, stopped move touches nothing,
  addresses follow Start and Stop, gateway address stays); `go test ./...`: packages that failed
  under parallel load (Access is denied, native hello timeouts) pass when run alone;
  `netcfg` PowerShell live test timed out (20 s), unrelated. Frontend lab test 3/3; types clean.
- Real ScadaBench Lab card on the owner's PC, separate engine and scratch data folder, two
  template IEDs at 10.77.0.11 and 10.77.0.12:
  - Use: 104 ms, card empty. Start: both on the card, ping 10.77.0.11 answers.
  - Stop of one IED: only its address removed. Stop: card empty, ping no reply.
  - Move of a running IED to 10.77.0.21: old off, new on; 8.8 s, of which stop 2.7 s and start
    4.5 s of the IED itself (the same stop/start as before).
  - Independent client: TCP 102 connected on 10.77.0.12 and 10.77.0.21 while running.
  - Hard kill while running left both addresses; the next launch removed them.
  - Normal exit (Ctrl+Break) while running: exit in 4.1 s, code 0, card empty.

## Limitations

- MMS was checked at TCP level only; the IED workers and their MMS behaviour are unchanged.
- A Release that fails (service stopped) leaves the address until the next Stop, launch or restart.
- The gateway's and master's own addresses still stay on the card while chosen (ADR-0125).
