# ADR-0005: Bounded deadline recovery with visible overruns

Status: accepted, 2026-09-09.

The first long load run showed scheduling gaps under concurrent builds. Resetting
the next deadline to now + 100ms on every delay discarded scheduled ticks without
an event. Preserve the original cadence across short stalls, retaining at most
five overdue cycles. Longer stalls explicitly report discarded ticks and original
lateness. Start and pause/resume reset deadlines so startup/paused time is not debt.

Recovered attempts evaluate the current active elapsed time, not historical values
with fabricated timestamps. Delivery can therefore burst after a stall; this is
not hard real-time behavior or proof of a uniform 100ms wire cadence. Unexpected
cycle errors pause automation and emit failure/runtime events, leaving MMS active.

Benchmarks count actual acknowledged updates, failures, late cycles and discarded
ticks. Start/end source and worker-binary hashes identify changed implementations;
customer XML is excluded. Record raw measured throughput, including sampling
boundaries. Do not label a short runtime check as the one-hour/UI/report-loss gate.
