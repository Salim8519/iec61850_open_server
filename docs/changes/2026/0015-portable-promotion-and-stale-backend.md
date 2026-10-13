# 0015: Normal-build promotion, portable checks and stale-backend diagnosis

Date: 2026-09-09. Status: normal worker and portable preview rebuilt.

The old soak ended with workers stopped. Its final result is not acceptance:
requested 3,600 seconds, measured 37,438.687 seconds, 3,439,700 acknowledged updates,
91.8756/sec measured, zero reported source/worker failures, 299,900,928-byte peak
working set. A large timing gap occurred after approximately 3,484 seconds; its
cause is not established by this artifact. Evidence remains unchanged at
`.local/benchmarks/a73e3e48b2684c86821db0ec55d33ec2/result.json`.

Promoted the candidate correction to the normal native build after confirming no
workers remained. Rebuilt the portable folder with current source/UI/native code.
The strengthened packaged self-test checks schema-valid nested arrays through MMS,
edited SCL export, stopped startup recovery, IED removal and restoration of its
deletion backup. Packaging tests also verify duplicate-launcher lock refusal.

Conformance correction: SCL2007B4 does not permit `count` on a top-level LNodeType
DO. The prior synthetic top-level-array test demonstrates native CFG behavior,
not schema conformance. The packaged fixture now uses a schema-valid SDO array.
This distinction is recorded in ADR-0006; the original XSD is unmodified.

The user reported a deletion/export 404 after returning. Port 8765 was owned by
Python PID 16308, parent 39840, started at 02:03:04 on 2026-09-09, before these
routes were added. Updated on-disk frontend assets plus the old in-memory backend
are the likely mismatch. The Browser skill connection still fails before setup,
so requested the user save before restarting that server. No user IED was deleted
or live project replaced during diagnosis. Add explicit stale-backend guidance
for API/download 404s instead of displaying the generic Flask error.

Build logs: `.local/portable-build-recovery.log` and
`.local/portable-build-delete-fix.log`. Both report completed collection; artifact
tests, not that log alone, determine whether the new bundle works. This remains
a same-machine developer preview, not clean-machine or browser acceptance.

Verification after the final rebuild: 76 tests passed in 17.30 seconds using
normal native binaries and the newly frozen preview. The packaged deletion,
backup restoration, export, array read, stopped recovery and lock checks pass.
File length, architecture and documentation checks also pass.
