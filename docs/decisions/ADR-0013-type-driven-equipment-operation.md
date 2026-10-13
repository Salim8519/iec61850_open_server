# ADR-0013: Type-driven equipment operation and selected feedback

Status: accepted, 2026-09-16.

Equipment operation is addressed by canvas node ID, not by a guessed IEC 61850
reference. The backend resolves that node's independently selected command and
state bindings against the imported SCL catalogue. CB, DS and ES require an
imported DPC control; Trip relay requires SPC. Logical-device, logical-node and
data-object names have no role in compatibility decisions.

The wire command retains the IEC 61850 control type. DPC `ctlVal` is Boolean:
false requests Open and true requests Close. Dbpos is the status representation,
where Open is 1 and Closed is 2. SPC remains Boolean; ADR-0014 makes its semantic
Operate/Reset or Press value explicit in equipment configuration. The native
worker remains the authority for applying the MMS command.

After the native command succeeds, the coordinator updates the state mapping
that the user explicitly selected:

- native Dbpos receives 1 for Open or 2 for Closed;
- a single Boolean switch point receives false or true;
- open/closed Boolean contacts receive true/false or false/true;
- a Trip relay Boolean receives the configured raw Operate or Reset value.

These writes use the normal acknowledged sample path. Timestamps, reports,
events, overrides and canvas state therefore follow the same rules as other
runtime values. If the selected state is the command object's own native status,
the duplicate write is skipped because the worker already applied it. A command
may remain intentionally state-less.

All selected feedback targets are validated before sending the command. If a
write still fails after command acceptance, the API and log distinguish
"command applied" from "simulated feedback failed"; it must not report the
entire operation as an unapplied command.

The signal picker lists runtime-editable imported points and does not require a
free-text path search. One-point switch mapping accepts Dbpos or Boolean and
stores the corresponding mode from the selected signal's imported type. Two
Boolean contacts remain an explicit choice. This keeps vendor models general
without silently inventing a relationship between similarly named objects.

This refines [ADR-0011](ADR-0011-typed-equipment-controls.md): state and command
bindings remain independent engineering choices, while a successful UI operation
now drives the chosen simulated feedback.
