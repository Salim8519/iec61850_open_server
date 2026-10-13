# ADR-0009: Report-first mapping, saved tests and guarded display motion

Status: accepted, 2026-09-09.

The user chose CFG-style mapping workflow only, a saved signal test panel, visual
switch animation only, and an explicit all-signals fallback. CFG database import,
CFG control logic, new interlocks and physical switching-time changes are excluded.

Report identities include logical device, logical node and BR/RP scope. Dataset
membership resolves against exact imported references and functional constraints;
structured and supported indexed members expand to leaves. Unresolved members
remain visible as diagnostics. Selection never changes native report configuration.
Signal descriptions inherit SCL instance/type descriptions without inventing names.

One browser picker serves equipment and test controls. Report provenance is saved
with the chosen exact reference; explicit All signals preserves old mappings.
Integer/enum switching feedback requires explicit position values. Boolean and
double-point feedback use their defined display states. SVG contacts interpolate
for 300 ms after confirmed feedback, with reduced-motion support. Display motion
does not modify native operation timing, timestamps or report generation.

Saved test controls are independent of the drawing and contain labels, exact IED
references, report provenance and Boolean pulse settings. Pulses run in the backend
on monotonic time, including while automation is paused. Native per-attribute
revisions guard pulse return updates under the data-model lock. Sample writes and
MMS control feedback advance these revisions, including writes of the same value.
Later writes win. Stop/failure cancels pending pulses. No actions restart from disk.
Manual updates suspend automatic sources until an explicit runtime release.

Bulk deletion uses a fingerprint of the reviewed engineering state and selected
IDs, one verified recovery archive, one atomic save and rollback on failure.
Lifecycle and project locks serialize the confirmation against Run and editing.
Physical nodes/wires stay; affected mappings, sources, scenario steps and saved
test controls are removed together.

New project archives use format 3 to prevent older readers silently dropping panel
definitions. The compact XML source pool and resource limits remain unchanged.
Formats 1 and 2 migrate with an empty panel and existing unfiltered mappings.
Archive validation compares preserved engineering fields while ignoring newly
derived status-kind metadata and absent/null optional mapping fields.

Candidate native/UI/portable output directories allow testing while the user keeps
an older simulation running. Never overwrite a running engine or mix its frontend
with a newer backend. Promotion requires stopping workers and preserving autosave.
