# 0227 Gateway Start after Stop: Modbus Stop no longer hangs on a connected master, listen failures name their cause; a late Lab add is taken off by this copy

Date: 2026-11-20. Owner: "do all of them, once i wake, i should see all ready and no coding left,
just testing". Lead: two bugs found in the night's trials, to fix properly: "Gateway Start right
after Stop fails with port in use" (6 of 200 in trial N3 on the code before change 0222) and "a Lab
address add whose answer was lost, finished later by the service, is journaled `failed` and stays
on the card until a Windows restart" (limitation of change 0222).
Related: [0222](0222-lab-addresses-owned-and-kept.md), [locks results](../../testing/speed-trials-locks-2026-11-20.md),
[ADR-0141](../../decisions/ADR-0141-lab-addresses-owned-and-kept.md) (compatibility note added, no
new decision). Layers: native gateway (`studio_gateway`), Go engine (`labsetup`, `netcfg` comment).

## Finding: what refused the Start after a Stop

- **Not reproducible on today's code.** `studio/tools/trials/t0227_gateway_restart.py`: IEC 104
  and Modbus TCP, on the gateway's own Lab address 198.18.27.50 and on 127.0.0.1, each Start right
  after the last Stop, 200 cycles each. Plain: **0 of 800** refused. With a TCP client connected
  before every Stop and one IED starting and stopping on the Lab card meanwhile (as N3 did): IEC
  104 **0 of 400** refused. Right after every Stop answer Windows listed **no listener** on the
  port and the old `studio_gateway` process **no longer existed** (2 000 Stops): Stop already
  waits for the program to exit (`worker.Process.Close`), so there is no release race to wait for.
- **Why the six were refused before 0222.** lib60870's IEC 104 listener sets `SO_REUSEADDR`, and
  Windows does not refuse a bind for `TIME_WAIT` connections even without it (probe: listen,
  connect, close either side first, listen again on the same port: accepted in all four cases).
  So neither the old listener nor `TIME_WAIT` can refuse an IEC 104 bind; the address not being
  on the PC at that moment can. The same N3 run (`n3n5-n3-start-base-20261120-112232.json`) shows
  the IED loop refused five times with "MMS bind failed: 198.18.24.10" and one loop with the
  gateway address missing "not by us": other agents' pre-0222 engines took this engine's Lab
  addresses off (the cross-removal finding fixed in 0222). The native message said "Another
  program may use this port" for every bind failure, so a missing address read as a busy port.
- **A real Stop bug the loop found instead: Modbus TCP Stop with a connected master.** On Windows
  `shutdown()` does not wake a `recv()` another thread waits in, so a connected master that sent
  nothing held the gateway's Stop until its next request or the 60 s idle timeout. The app gave
  up after 10 s, then ended the program: **Stop took 16.0 s and answered "Native halt request
  timed out" in 19 of 20 cycles** (`--protocols modbus --client --cycles 10`, both listens; the
  20th client did not connect in time). A master polling every second made Stop wait up to that
  second.

## Change

- **Modbus Stop wakes every connection** (`native/gateway/modbus_server.cpp`): Stop ends the
  acceptor first, then shuts down and closes each master's socket (taken out of the list, so the
  connection thread does not close it again) and joins the threads. Closing wakes the waiting
  `recv()`; ending the acceptor first means no new socket can reuse a closed one's handle.
- **A failed listen names its cause** (`native/gateway/listen_error.cpp`/`.hpp`, new;
  `server_104.cpp`, `modbus_server.cpp`): "Could not listen on A:P. Another program uses this
  port. Close that program or choose another port." (WSAEADDRINUSE); "...: A is not an address of
  this PC now. If it is the gateway's own address on the ScadaBench Lab card, press Start again;
  otherwise choose another network." (WSAEADDRNOTAVAIL; Start puts the own address back);
  "...: Windows does not let programs use this port (another program holds it for itself, or
  Windows keeps it reserved). Choose another port." (WSAEACCES); otherwise the old sentence with
  the Windows error number. lib60870 keeps its bind error to itself, so after a refused IEC 104
  start the gateway binds a probe socket the way lib60870 does and reads the error.
- **Late Lab adds are settled by the service's own list** (`labsetup/lost.go`): after a lost
  answer the card is read as before; when the service answers that read, the result is final
  (it changes the card one request at a time, so its answer comes after the lost request). When
  it does not answer either (still busy) and Windows does not show the address yet, the add is
  journaled `requested` (it counts as this copy's, ADR-0141 item 1) instead of `failed`, and a
  lost removal of an address Windows no longer shows is journaled `failed` (not done) instead of
  `removed`. The next Release (any Stop, project change, exit, launch) then takes off an address
  the service put on late, or journals `skipped` for one that never came. Run and Start still
  fail with the same reason. `netcfg/journal.go` comment updated.
- Specs: gateway GW-20, GW-21; lab-network NET-25. ADR-0141 compatibility note.
- Trial harness: `studio/tools/trials/t0227_gateway_restart.py`, `t0227_mms_master_restart.py`.

## Evidence

- Gateway loop, trial native build (`studio/native/build/trial-0227`), client before every Stop
  and the IED loop running (355 IED start/stops): **0 of 800** refused, **0 Stop failures**, no
  listener and no old process after any Stop. Stop median (worst): IEC 104 own 0.087 s (0.473),
  This PC 0.051 s (0.552); Modbus own **0.057 s (0.383)**, This PC **0.025 s (0.819)** instead of
  16.0 s. Start median 0.18-0.50 s; Start worst 3.3-10.5 s in rounds where the IED loop held the
  Lab-card lock (one IED start timed out under the PC's load then; other agents ran too).
- IED and master, today's code (`t0227_mms_master_restart.py`): one IED on the Lab card started
  right after its stop **0 of 200** refused (start 0.315 s median, 1.168 worst); Run/Stop of 8
  IEDs **0 of 50** (0.916 s, 6.307); SCADA master Connect/Disconnect toward a second engine's
  gateway on 198.18.27.50, IEC 104 and Modbus, from All networks and from its own address
  198.18.27.51: **0 of 800** refused (Connect 0.06-0.35 s median). The pattern does not show for
  MMS ports (libiec61850 also sets `SO_REUSEADDR`) or the master (it connects from port 0), so
  nothing changed there.
- Protocol tests `studio/tests/protocol/test_gateway_wire.py` with `STUDIO_NATIVE_BIN` = the trial
  build: 10 passed; with today's Release build the three new tests (four cases) fail.
  Full protocol suite with the trial build: 94 passed, 9 skipped, 2 failed that fail the same way with
  today's Release build and touch no changed code (`test_adapter_listing` compares the Lab card's
  addresses while other agents change them; `test_master_wire` DNP3 is timing-sensitive and passed
  on rerun with both builds).
- Go: `lost_late_test.go` 3 tests fail before (journal `[failed]`, address left on the card) and
  pass after; `go test ./internal/labsetup/ ./internal/netcfg/`, `go vet` and `GOOS=windows go vet`
  (also `-tags trial` for labsetup) pass. `verify.py`: every section PASS (exit 0).

## Limitations

- The six refusals of trial N3 were not reproduced: their cause (another, pre-0222 engine taking
  the address off between Start's add and the listen) is inferred from the same run's other
  faults, not watched. An older ScadaBench (0.1.34, installed) still takes off every Lab address
  it does not use; the gateway then now says the address is not on this PC and Start puts it back.
- lib60870's `SO_REUSEADDR` lets a second IEC 104 listener bind a port another program already
  listens on (Windows shares it); not changed here (vendor HAL, pinned).
- The late-add settlement is unit-tested; a service that loses an answer cannot be produced on
  demand with the installed service (it answers in milliseconds).
- The lead rebuilds `studio/native/build/Release` (`studio_gateway`); until then the app uses the
  old gateway program.
