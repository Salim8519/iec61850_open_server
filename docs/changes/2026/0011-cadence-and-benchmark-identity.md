# 0011: Timing recovery and benchmark identity

Date: 2026-09-09. Status: tested; full load acceptance remains outstanding.

Fixed-deadline automation now preserves short scheduling delays instead of silently
discarding ticks. A five-cycle backlog bounds recovery; longer overruns report
original lateness and discarded-tick counts. Start/resume reset deadline debt.
Unexpected cycle errors visibly pause automation while the clock thread survives.
See [decision](../../decisions/ADR-0005-automation-cadence.md).

Benchmark runs record failure/late/skipped counts, runtime state, start/end source
and worker binary hashes, and files changed during measurement. Customer XML is
not included in provenance. An optional loopback port permits an independent
diagnostic run without interrupting a long test. End elapsed time now includes
waiting for the final active cycle to settle before computing throughput.

Evidence:

- Cadence tests cover early wake, exact deadline, transient delay, bounded long
  stalls with explicit skipped counts and resume without paused-time replay.
- Injected cycle failure confirms a visible paused state and surviving thread.
- Provenance test confirms source/native changes affect identity and customer CID
  contents are excluded. Existing native scenario pause/resume test still passes.
- Complete Python suite: 66 passed in 14.64 seconds. File length, architecture and
  documentation checks pass.
- Separate 30-second native run on port 18121: 20 IEDs, 50,000 attributes, startup
  3.875 seconds; 29,900 acknowledged updates, 996.6667/sec measured, zero failures,
  zero reported late cycles/skipped ticks, peak working set 287,195,136 bytes.
  Workers stopped and no captured source/binary changed during the run.
  Evidence: `.local/benchmarks/fce0b9a5523a4719a0d0895d3b7b2982/result.json`.
  This run precedes the small end-time accounting adjustment described above.
- Original one-hour run remains live on its original code. Interim observation:
  2,132.062 seconds, 2,108,400 applied updates, zero source/worker failures and
  299,384,832-byte aggregate working set. It does not verify the new scheduler.

Neither run proves full throughput, uniform wire timing, browser responsiveness
or report-loss acceptance. The complete release and hibernation gates remain open.
