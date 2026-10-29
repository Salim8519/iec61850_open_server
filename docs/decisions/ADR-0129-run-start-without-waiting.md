# ADR-0129: Run start overlaps worker reads and computes mode points once

Accepted: 2026-10-30. Owner: "make sure the app is instant, yet doesnt eat cpu, ram, network,
windows overall" and "What about the speed of run and stop ... achieve insane speeds using power
of GOlang". Change [0202](../changes/2026/0202-lab-service-one-call.md); companion of
[ADR-0128](ADR-0128-lab-service-one-call.md).

## Problem

Timed step by step on the owner's project (7 IEDs, 5.5 MB) on a busy laptop, after the workers
already published GOOSE, Run still spent about 30 s before it answered:

| Step | Time | Cause |
| --- | --- | --- |
| Initial snapshot (`publishBindings`) | 14.6–18.9 s | 1,704 worker reads sent one at a time, about 11 ms per round trip |
| Behaviour sync (`applyTestModes`) | 5.0–10.9 s | `modePoints` sorted every reference of each model and ran two regular expressions on each, on every call |
| GOOSE input preparation | 2.3 s CPU | `BindInputs` rebuilt the subscriber's whole signal map for each subscription |

## Decision

1. **Overlapped snapshot reads.** The initial snapshot plans every read in the old order, then
   reads all IEDs in parallel; inside each IED's actor up to 64 reads are outstanding (the worker
   matches answers by request id and answers in order). The values are then stored and emitted
   in the old order, so the run log, the table and the SLD see exactly what reading one by one
   gave. A failed read stops the Run with the same first error in plan order.
2. **Mode points once per model.** Only references ending in `.Beh.stVal` or `.Sim.stVal` can
   match the patterns; those few are sorted and matched, once per model (`sync.Once`; a model's
   references never change during a run). Callers only read the result.
3. **One signal map per subscriber IED.** A `goosedom.Binder` keeps each subscriber's signal map
   for the rest of one preparation; `BindInputs` keeps its contract.

No new goroutine outlives the call, nothing polls, and memory grows only by the mode-point lists
of the running models (a few hundred strings).

## Compatibility

Same events, values and order; same errors. Worker protocol unchanged (it already allowed
several requests in flight). Python engine unchanged.
