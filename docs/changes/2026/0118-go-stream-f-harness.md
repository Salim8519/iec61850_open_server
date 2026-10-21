# 0118: Go stream F merged: parity harness, golden replay, SCL corpus, benchmarks, Go licensing/updates/network

Date: 2026-10-20. Workstream F of [ADR-0076](../../decisions/ADR-0076-go-only-development-and-parallel-workstreams.md)
(P0 harness, P5 platform). Branch `go-f-harness` (ad3d2a6, 6c30ac3) merged into `master` (d2126da). Stream
log: [f-harness-platform](../../architecture/go-engine/streams/f-harness-platform.md) (proposed harness ADR
text in its section 8, to be numbered when the Go server is wired).

## Change

- `studio/tests/conftest.py`: `STUDIO_ENGINE=python|python-http|go`; the default `python` runs every
  test as before; other values load `tests/parity/plugin.py`, which replaces `create_app` with a remote
  backend process (Job Object, leftover `studio_*.exe` fails the test, natives hashed).
- `studio/tests/parity/`: seams and callback protocol, golden replay (`record|golden|dual|selfdiff`,
  recorder refuses non-public uploads and IED names), four demo-station transcripts, SCL corpus
  generator and 182 public/synthetic cases (shared with stream B).
- `studio/tools/benchmark_engines.py`, `studio/tools/engine_bench/`: S1-S10 over HTTP only, launcher
  without the dev watcher, IEDs on 127.66.x.y:18700, RAM/CPU per process each second, provenance.
- Go: `internal/licensing` (certificates, Ed25519, policy, time anchor, gate), `internal/updates`
  (ADR-0071 signed release check, `update_check.json`), `internal/netcfg` (IP Helper adapter list,
  plans/fingerprints, firewall rule names, journal, helper validation, preview/apply with fakes).

## Evidence

- Stream: full suite 446 passed / 104 skipped under both `python` and `python-http` (worktree without
  native build) with identical skip lists; harness self-tests 9/9 in both modes; replay self-diff with
  hash seeds 0 and 1 and replay against goldens: 0 differences (one real Python non-determinism,
  `/health` `modes.ieds` order by random ids, normalised with its reason); corpus self-diff 182/182.
- Go vectors from the frozen Python: 24 license scenarios identical (answers and files), update check
  bytes identical, network plans/journal/rule names identical; read-only adapter listing 8/8 identical.
  No test changes adapters, addresses or firewall rules or asks for elevation.
- Python baseline S1+S9 (5000 points/IED, 60 s idle): 1 IED ready 1.59 s, run 0.13 s, stop 0.02 s,
  backend 74 MB / 0.47 % CPU; 2 IEDs 1.86 s, 0.24 s, 0.03 s, 81 MB / 0.29 %.
- Integrator on `master`: only owned paths; private-data scan clean (only a fake test machine id);
  `go vet` clean; netcfg, licensing, pytext, updates pass; Python unit+integration 457 + 3 passed
  (same as before the merge); `tests/parity` 9 passed; `verify.py`: see ledger line.

## Limitations

- Native-dependent tests not yet proven under `python-http`; benchmarks S2-S5, S7, S10 written but not
  run (RAM limit); S6 (Npcap rig) and S8 (gateway/firewall) refuse to run on this laptop by design.
- Not built: runtime golden scenarios (operate, `wait_event`), IPC tap, route coverage report, UI tour.
- `/license`, `/app/update`, `/network` HTTP handlers and the dev network-helper launch not ported.
- Public-file hashes depend on `core.autocrlf=true`; re-run `parity.replay.manifest --write` elsewhere.
- Needs from stream A for `STUDIO_ENGINE=go`: parity build of `studio_server` answering the
  `/api/v1/_parity/*` table and reading the seams schema; benchmark output line with the session URL.
