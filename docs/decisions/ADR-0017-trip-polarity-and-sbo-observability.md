# ADR-0017: Independent Trip polarity and enhanced-SBO observability

Status: accepted, 2026-09-16.

Trip command meaning and Trip feedback meaning are independent engineering
settings. `trip_command_polarity` maps semantic Operate/Reset actions to SPC
`ctlVal`; `trip_state_polarity` maps the accepted semantic action to the Boolean
state point. The default contract is Operate sends 0, Reset sends 1, Operated is
1 and Reset is 0. External SPC commands use the same two-stage translation.
Raw `ctlVal` is never treated as mapped Trip feedback merely because the command
and state refer to the same native SPC object.

Project format 7 replaces `trip_polarity` with the two explicit fields. Formats
1 through 5 migrate to the default command and state meanings. Format 6 preserves
its explicit command polarity and assigns Operated=1 / Reset=0. Opening any older
format first creates and verifies a byte-identical recovery archive. Older Studio
builds reject format 7 instead of silently combining the meanings again.

Imported CB/DS/ES DPC semantics remain false=Open and true=Close. Enhanced SBO
selection reserves the object but does not operate it. A matching `Oper` is still
required; repeated `SBOw` remains rejected by libiec61850 with Object-already-selected.
The explicit non-standard exception later requested for one concentrator is
isolated by [ADR-0018](ADR-0018-concen-sbow-compatibility.md).
The worker registers the stack selection-state callback and publishes selected,
operated, canceled, timed-out, failed and disconnected events with the imported
SBO timeout. The private contract advertises `control_selection_events`, and the
coordinator rejects a stale worker without that marker.

Accepted mapped commands force an authoritative equipment-state publication even
when feedback is unchanged. `GET /api/v1/equipment/states` provides the same state
projection for SSE connection and resynchronization. This avoids interpreting an
SBO reservation as movement and prevents no-op commands or lost event history from
leaving the canvas stale.

The laboratory SCL template uses the IEC Cancel structure without the Oper-only
Check field. This lets an independent client exercise cancellation without changing
the retained structure of imported customer SCL.

This refines [ADR-0015](ADR-0015-native-control-feedback.md) and supersedes the
single Trip polarity compatibility model in
[ADR-0016](ADR-0016-format-6-trip-and-live-canvas.md).
