# ADR-0018: Explicit origin-scoped CONCEN SBO adaptation

Status: accepted, 2026-09-16.

Standard enhanced Select-Before-Operate remains the default and still requires a
matching `Oper`. One observed concentrator identified itself as `CONCEN`, inverted
Close to Boolean false, sent only `SBOw`, and never sent `Oper`. The user explicitly
requested a simulator compatibility path after the packet-level diagnosis.

Control bindings now store `standard` or `concen_inverted_sbow`. The non-standard
profile is valid only for an imported control-model-4 DPC object and is visibly
warned in Equipment Properties. It never applies to SPC or Trip Relay controls.
The native worker also requires exact origin identifier `CONCEN`; other clients,
including Studio's bundled probe, retain false=Open and true=Close.

For a successful matching `SBOw`, the worker queues execution outside the MMS
selection callback, inverts the raw Boolean and applies DPC feedback. Queueing
avoids taking the native data-model lock recursively while the MMS write is still
being processed. Raw and executed values, actions, profile and trigger are emitted
as distinct evidence. A later matching `Oper` is accepted without applying the
movement twice.

The IEC selection itself remains owned by libiec61850. It can subsequently time
out, cancel or disconnect, and repeated `SBOw` can still report already selected;
the Event Log states when the compatibility operation was already applied. Studio
does not silently rewrite the imported control model or customer SCL.

Project format 8 adds the explicit binding profile. Formats 1 through 7 migrate to
`standard`, so merely opening an older project never enables non-standard control.
An exact verified recovery archive is created before migration.

Follow-up evidence on 2026-09-16: `commmand_traces_5.pcapng` used
`CONCEN` + `SBOw(false)` for an operator-requested Open. The profile correctly
performed its documented inversion to Close, so an already Closed breaker did not
move. The prior trace had identified the same wire tuple as a Close request. The
profile therefore cannot infer both stated intents. It remains explicit and must
not be promoted as a general concentrator solution; see change 0028 and the current
handover.

Refinement on 2026-09-17 (change 0031): an accepted command that does not change
the value no longer rewrites the native feedback attribute, its timestamp or the
revision counter; `control_applied` carries `changed=false` instead. Run-log
evidence shows the observed concentrator completing standard `SBOw`+`Oper`
sequences on the standard profile (Close), so the recommended operator setting
for this concentrator is `standard`, pending a fresh two-direction capture. Open
attempts still never produced an `Oper` in recorded runs; the decision on any
further policy remains open with the failure-mode requirement unchanged.
