# ADR-0014: Semantic equipment actions and direct mapping selection

Status: accepted, 2026-09-16.

Selecting the final signal in a mapping picker commits it immediately. Mapping
pickers do not add Copy, Use or Clear buttons around that selection. A
`Not assigned` option removes state, while `No command binding` removes command.
State and command remain independent and optional for CB, DS, ES and Trip relay.

Equipment APIs accept semantic actions so the backend, not the browser, resolves
their IEC 61850 Boolean value. Switch Open/Close retains false/true. Trip relay
stores an explicit polarity; new nodes default to Operate=false and Reset=true,
with the inverse available. Format 4 projects migrate to the old
Operate=true/Reset=false polarity to preserve their prior behavior.

One Push Button kind has two exclusive behaviors. Momentary SPC sends one
configured Boolean command per press and performs no opposite wire command.
Latching status toggles one explicitly mapped writable Boolean through the normal
sample writer, so reports, timestamps, overrides and logs remain authoritative.
Changing behavior clears the incompatible mapping.

The Source drawing item represents an analog measurement source. Its state
binding accepts numeric MX points by imported type and FC. Its inspector reuses
the existing manual, constant, ramp, sine, step, noise and CSV source services.
No browser-side generator or reference-name rule is introduced.

This decision refines the fixed Trip relay polarity in
[ADR-0011](ADR-0011-typed-equipment-controls.md) and the raw Boolean operation
interface in [ADR-0013](ADR-0013-type-driven-equipment-operation.md).

