# ADR-0015: Atomic native controls and shared mapped feedback

Status: accepted, 2026-09-16.

Native DPC and SPC controls apply their owned status and timestamp atomically in
the libiec61850 handler before returning success. The worker publishes the exact
Boolean `ctlVal`, control reference and resulting native status in a private event.
No artificial handler delay or shared pending target is retained. This prevents a
later command from replacing the value of an earlier accepted command.

The coordinator indexes diagram nodes by exact IED ID and imported control object.
It consumes native control events on a dedicated thread, resolves each node's
stored polarity and feedback mapping, and uses the normal sample writer for mapped
state points or constructed contacts. It avoids redundant native-status writes
when semantic state already matches, but overrides an SPC status mapped as Trip
feedback when explicit state polarity differs from command polarity. Commands from
an external MMS client and commands initiated through Studio follow one authoritative
feedback path without blocking the worker reader.

State and command remain independent and optional. One command may be shared by
multiple diagram nodes, each with zero or one state construction. No IED name,
logical node name or vendor reference pattern is inferred.

The worker advertises `atomic_control_feedback`; the coordinator rejects an older
binary rather than silently running incompatible control semantics. Rebuilding the
normal native worker is therefore required after this source change.

This refines [ADR-0013](ADR-0013-type-driven-equipment-operation.md): accepted MMS
events, rather than only the initiating HTTP request, are now the trigger for
mapped feedback.
