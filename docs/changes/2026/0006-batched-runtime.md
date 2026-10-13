# Change 0006: measured runtime throughput

Date: 2026-09-09. Synthetic fixture only; no customer data involved.

The new fixture has exactly 20 IEDs and 50,000 scalar attributes on
127.51.0.2 through 127.51.0.21, port 18120. Five measurement sources per IED
at 10 Hz request 1,000 aggregate value updates/sec through the real coordinator.

## Baseline and correction

Initial 20-second result: startup 3.812 seconds, 265 updates/sec, peak aggregate
working set 277,389,312 bytes, zero source/worker failures. This missed the target.
Evidence: .local/benchmarks/f988cd86b50444009f0c787957cc61d9/result.json.

Replaced per-point IPC with bounded per-IED batches and concurrent independent
IED dispatch. Manual and generator writes serialize per IED. A fixed-deadline
clock avoids adding processing time to every 100ms period. Pause/Stop synchronize
with active automation cycles. Logs keep an open buffered segment instead of
opening a file per event; control/runtime events and close flush the buffer.

Repeated 20-second result: startup 3.062 seconds, exactly 20,000 acknowledged
generator updates (1,000/sec), peak 278,130,688 bytes, zero failures, workers stopped.
Evidence: .local/benchmarks/57dd48f3d35e49df93d214a21390d8e7/result.json.
Hardware: AMD Ryzen 9 5900HS, 8 physical cores, 16 logical processors and
25,132,564,480 physical RAM bytes. RAM differs from the stated 16GB fixture.

## Logging and current verification

Application-owned event segments are bounded to approximately 1GB, plus the
active segment. Rotation prunes oldest recognized segments only, preserves
unknown files and active segments, and records pruning in manifests. Pruned
historical segments cannot be recovered unless separately backed up.
The safety and flush tests pass. Full regression suite: 32 tests pass.

One-hour soak started; evidence directory:
.local/benchmarks/a73e3e48b2684c86821db0ec55d33ec2.
Do not claim the soak passed until result.json records final completion.
The benchmark explicitly does not verify UI responsiveness or MMS report loss.
Those remain separate acceptance requirements even if the one-hour runtime passes.

Interim soak observation at 1,277 seconds: 1,266,200 updates, zero reported source/
worker failures, aggregate working set 294,363,136 bytes. A scheduling gap appeared
while concurrent builds/tests ran. Therefore sustained 1,000/sec acceptance is not
proven by this run; retain the complete result rather than hiding missed cycles.

Final result: run ended after a large timing interruption and stopped workers.
See [0015](0015-portable-promotion-and-stale-backend.md). This soak did not pass
the uninterrupted one-hour requirement; it has not been silently replaced.
