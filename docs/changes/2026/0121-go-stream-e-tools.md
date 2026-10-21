# 0121: Go stream E merged: tools, IED Explorer, Find IEDs, traces, test panel, gateway/master/converter

Date: 2026-10-20. Workstream E of [ADR-0076](../../decisions/ADR-0076-go-only-development-and-parallel-workstreams.md)
(phase P4). Branch `go-e-tools` (b215c6d..da550c4, 9 commits) merged into `master` (6bb5c64). Stream
log: [e-tools](../../architecture/go-engine/streams/e-tools.md) (proposed ADR text for the
change-driven gateway value push). Python stays the engine; the routes are not wired yet.

## Change

- `internal/tools/` (`pytext`, `pycsv`, `pyoracle`, `nativetest`, `worker`, `numlog`, `control`,
  `memory`, `scan`, `explorer`, `trace`, `testpanel`): tool memory, Find IEDs range and scan, numbered
  logs, control client, IED Explorer, CSV traces and value rules, test panel validation.
- `internal/gateway` (domain rules) and `internal/gateway/service` (gateway, master, converter,
  probe at runtime). The gateway recomputes only points whose value changed (hub subscription),
  batches at most every 100 ms and 500 points, with a 10 s safety sweep; command confirmation waits
  on hub notifications instead of 100 ms polling (same 3 s deadline and texts).
- `internal/api/toolsapi`: tools, gateway, master, test-panel and traces/preview routes.

## Evidence

- Oracle parity with the frozen Python, byte for byte: memory 400 cases, scan plans 6000, traces
  ~10,000, plus pytext, pycsv, explorer, test panel and gateway/master/converter rules.
- Route parity: ~190 requests in three scenarios identical to Flask (status, body, Content-Type,
  Content-Disposition).
- Wire tests with real workers, none skipped: Explorer against `studio_ied` 127.65.0.32:20132 with
  `studio_live`, `studio_control` and a range scan; gateway over IEC 104 on 127.0.0.1:20404 with
  `studio_test_iec104_master` (GI, select/execute COT 7/7/10, spontaneous update, log entry);
  master runtime test over HTTP (port 20411).
- Gateway CPU at 5000 points (real `studio_gateway`, worker CPU excluded): idle Python 12.6-19.0 %
  of one core, Go 0.03 %; 1000 changes/s Python 14.7-19.1 %, Go 0.49-0.64 % (cycle count) or
  0.62 % (tick timer; 1.17-1.33 % in earlier runs before subtracting the harness).
- Integrator on `master`: only owned paths; private-data scan of 127 files: no match; `go vet`
  clean; tools, gateway, gateway/service and toolsapi pass (`go test -p 2 -trimpath`);
  `verify.py`: see ledger line.

## Limitations

- Not wired: stream C value hub and native framing (fakes plus real workers in tests), stream B
  project service (`IEDs`, `Gateway`, `SaveGateway`, `TestPanel`, `SaveTestPanel`), stream A
  composition (`toolsapi.Deps`, startup `Master.Autoconnect()` then `Manager.Autostart()`).
- Not ported: gateway firewall and network routes, `/signals/pulse`, `/signals/release-override`.
- Test binaries need `-trimpath` on this laptop (antivirus refuses some untrimmed ones).
