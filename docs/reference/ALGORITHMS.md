# Core algorithms and invariants

This is a navigation-level specification. Source and tests remain executable truth;
ADRs explain why a compatibility boundary exists.

## Project open and migration

1. Bound and validate the ZIP and its single `project.json` entry.
2. Reject unsupported future formats and unsafe paths before extraction.
3. For formats 1-7, create and byte-verify a recovery archive before mutation.
4. Expand checksum-referenced SCL sources, rebuild derived native models, migrate
   mappings, validate the result, and atomically replace autosave.
5. Restore engineering data stopped. Session ID/revision are runtime metadata and
   never enter the archive.

Owners: `projects/archive.py`, `document.py`, `mapping_migration.py`, `service.py`.

## Runtime start

1. Assign deterministic loopback endpoints and preflight all listeners.
2. Build one native configuration per IED and spawn one worker per IED.
3. Require private capabilities `versioned_updates`, `atomic_control_feedback`,
   `control_selection_events` and `concen_sbow_compatibility`.
4. Push initial typed values, publish authoritative mapped states, then accept
   automation/manual/MMS activity.
5. A worker failure marks its IED failed; there is no silent restart.

## Switching commands and feedback

Standard DPC command meaning is `false = Open`, `true = Close`. Native feedback is
Dbpos `1 = Open`, `2 = Closed`. Studio commands resolve a semantic action, send the
Boolean command through the independent probe, then update only explicit feedback
mappings. Externally received native controls emit `control_applied`; a dedicated
queue mirrors that semantic action to every mapped node outside the worker reader
thread. Accepted no-op feedback still republishes the authoritative node state.

Enhanced SBO model 4 normally reserves on `SBOw` and moves only on a matching
`Oper`. Selection lifecycle comes from the native stack. The explicit
`concen_inverted_sbow` exception applies only to DPC model 4 with exact origin
`CONCEN`: successful `SBOw` is queued, Boolean-inverted and applied immediately.
This is a workaround, not IEC semantics. See the current handover before changing it.

## Trip Relay

Trip command and state polarities are independent. The default is Operate command
`0`, Reset command `1`, Operated feedback `1`, Reset feedback `0`. External SPC and
Studio operations first derive the semantic action from command polarity, then the
feedback value from state polarity. Raw `ctlVal` is never copied directly to Trip
feedback.

## Equipment state projection

Direct mappings expose their typed value. One Boolean switch state maps false/true
to Dbpos Open/Closed. Two contacts map `(open, closed)` as `(1,0)=Open`,
`(0,1)=Closed`, `(1,1)=Intermediate`, `(0,0)=Invalid`. SSE state events and
`GET /api/v1/equipment/states` use the same backend evaluator.

## Tab and event synchronization

The backend owns one session ID and monotonic project revision. Diagram writes carry
the revision and tab ID; mismatches return conflict. `project_changed` makes other
tabs cancel pending saves and reload. SSE IDs are monotonic in a bounded buffer; an
expired cursor yields `resync`, after which the frontend reloads project and equipment
state snapshots.

## Automation cadence

A 100 ms monotonic deadline drives sources and scenarios. At most five overdue
cycles are retained; larger backlogs emit skipped-tick evidence. Pause freezes
automation only, while MMS and manual actions remain active.

## Evidence journals

Runtime events are compact JSONL split at 10 MiB. Segment/lifecycle indexes append
facts instead of rewriting manifests; no automatic pruning occurs. Engineering
history is compact, segmented at 200 entries and SHA-256 chained. Repository
verification checks its schema, chain and Git line removals. Both are queried by
read-only tools documented in [history/INDEX.md](../history/INDEX.md).
