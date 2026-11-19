# 0220 Fix plan Phase A and owner-decided Phase B items

Date: 2026-11-19. Owner: "find real ones, we can't afford sending weird product to
customers", "do the plan"; VM sharing "we allow VMs, but never PCs"; "server backup should be
manual"; "uninstall should fix firewall with option".
Plan: [fix plan from the 2026-11-19 review](../../testing/fix-plan-2026-11-19.md).
Decision: [ADR-0140](../../decisions/ADR-0140-fix-plan-safety-decisions.md).
Layers: frontend, Go engine, GOOSE engine, native gateway and worker, license server, installer.

## Change

Each item has a test that failed before the fix and passes after, unless marked "not built".

- **Drawing and save (A1, A2, A15-A17).** A project read keeps a drawing edit that is not saved
  yet (`state/diagramSave.ts`); a failed save keeps the drawing, says "Not saved" and tries
  again on the next edit or Save; only a revision conflict reads the engine's drawing back.
  A render error shows "Something went wrong. Your IEDs are still running." with Reload
  (`app/ErrorBoundary.ts`). Busy is a count, so buttons stay off until the last action ends.
  The real-IED connection form resets only when the IED or its saved values change.
- **Project file (A3-A5).** The autosave is fsynced before it replaces the old one and before
  the journal is trimmed. A journal that cannot be read is moved to `recovery\` and the app
  starts with the last autosave and the existing recovery notice; nothing is deleted. Save
  returns the error of the write that covered it.
- **Network (A6).** Stop, launch, exit and Clean up remove only addresses the lab journal says
  ScadaBench added (never-source, manual, not on a DHCP card); the service refuses removals on
  DHCP cards; a journaled address gone after a restart is settled, not removed.
- **Real devices and hangs (A7, A8, A10, A11, A18, A19).** Gateway Stop refuses every waiting
  command ("the gateway stopped before this command ran"); a full queue refuses. An operate
  timeout kills the control worker and reports "did not answer in time; result unknown — check
  the position before trying again" (`real_command_unknown`); a second operate of the same
  reference while one runs is refused. Pipe writes to workers have the request deadline, so a
  frozen worker cannot hang Stop or Quit. A failed halt leaves the gateway stopped, not
  "running". Read errors mark real-IED values stale. A failed add of an IED to a running station
  stops and forgets it; workers are published under the lock only after their model started.
- **GOOSE (A9).** `frame.Parse` checks the goosePdu tag byte's length; both receive paths count a
  panic and carry on; frames after Close are dropped. Fuzzed for 30 s (3.7 million inputs).
- **Native, not built (A12-A14, A20).** Modbus client sockets close after 60 s without a request
  and finished connection threads are joined; overlapping Modbus registers across kinds refuse
  the start, naming both points; the command bridge cancels waiting commands on stop and drops
  late answers; a worker whose start fails after creating the server cleans up like `stop()`.
- **License server (A21-A23, B-2).** Lemon orders are accepted only when subtotal and currency
  match the current or a signed quoted price (as PayPal); Lemon test-mode and PayPal sandbox
  orders get trial keys; a refund that arrives before its order is remembered; duplicate refunds
  write one line; the manual backup also saves `sales.jsonl`, `admin-account.json`,
  `release.json`, and a new `restore --from FILE` command writes them into an empty folder.
- **Licence sharing (B-1).** A virtual switch is recognised by its adapter description, never
  by a name the user can change; a computer asks a host and accepts a host proof only when its
  SMBIOS firmware names a hypervisor.
- **Uninstall (B-3).** When the network journal exists the uninstaller asks "Also remove the
  firewall rules ScadaBench added and switch Windows' own rules back on?" (Yes by default); Yes
  runs `scadabench_labnet.exe firewall` with one administrator prompt, which undoes only
  journaled firewall items; a refusal or failure is shown.

## Evidence

- Frontend: `tsc --noEmit` clean; 500 of 500 frontend tests pass (487 before).
- Go engine: `go vet` and `GOOS=windows go vet` clean on every touched package; `go test -race`
  passes for gateway/service, realcmd, native, tools/worker, tools/control, acquisition;
  autosave, labnet, labsetup and licensing pass. The 9 netcfg failures on Linux are the same on
  the previous commit (Windows paths).
- GOOSE engine and license server: `go vet`, `go test ./...` pass.
- File lengths and documentation checks pass.

## Limitations

- Nothing was run on Windows. The C++ (A12-A14, A20) and the installer change were never
  compiled. `internal/runtime` fake-worker tests and the uplink tests need Windows.
- Manual gateway Send still runs directly (routing it through the queue would stop sends while
  the gateway is stopped).
- A VMware guest set to `SMBIOS.reflectHost = TRUE` looks physical and cannot use the host's
  licence. The native guard still accepts `vm_proof.json` without its own VM check.
- Lemon: the dashboard price and currency must equal the server's price, or real orders get
  no key. A tax-inclusive store accepts subtotal or subtotal plus tax.
- Uninstall with the lab card shows two administrator prompts.
- Laptop checks to run: [fix plan, laptop checklist](../../testing/fix-plan-2026-11-19.md).
