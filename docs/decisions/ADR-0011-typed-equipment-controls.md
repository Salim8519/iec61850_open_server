# ADR-0011: Typed equipment state and controls

Status: accepted, 2026-09-16.

CB, DS and ES use independent state and command bindings. A command binding is an
imported DPC control object. Its MMS `Oper.ctlVal` is Boolean: false requests Open
and true requests Close. The corresponding DPC `stVal` is Dbpos, represented on
the wire as a two-bit coded enum. The application normalizes its four values as
0 Intermediate, 1 Open, 2 Closed and 3 Invalid/Bad. It does not send 01 or 10 as
the control value.

This distinction follows the bundled libiec61850 model mapper and is exposed by
the library's
[control-value type API](https://support.mz-automation.de/doc/libiec61850/c/latest/group__IEC61850__CLIENT__CONTROL.html).
The library's
[Dbpos API](https://support.mz-automation.de/doc/libiec61850/c/latest/group__DBPOS.html)
defines the four feedback states and converts between Dbpos and an MMS bit string.
The worker rejects controls whose declared CDC and feedback type disagree.

Switch display state supports three explicit mappings:

- native Dbpos, passed through unchanged;
- one Boolean, fixed as false to Open and true to Closed;
- separate open and closed Boolean contacts, from any imported IED/reference.

For two contacts, open/closed values `00`, `10`, `01`, `11` become Dbpos values
0, 1, 2, 3 respectively. Missing or non-Boolean contact values remain unknown.
Constructed values belong to the coordinator and canvas only: they do not add
virtual MMS data objects or change imported CID/SCD wire types. There are no
hard-coded logical-device, logical-node or vendor naming assumptions.

Trip relay is a distinct equipment kind. It binds an optional Boolean state and
an independent SPC control. Its polarity is explicit; ADR-0014 changes the new-node
default to Operate=false and Reset=true while retaining an inverse option. The
simulator does not auto-reset. Protection relay remains a separate drawing type.

Project format 4 replaces flat equipment fields with `state_mapping` and
`control_binding`. Format 1-3 Boolean switch mappings migrate to the one-Boolean
mode and native Dbpos mappings migrate directly. Incompatible legacy switch state
or control mappings are removed only after the original archive is copied to a
byte-verified, ZIP-verified recovery file. The UI displays the migration loss and
backup location. Older applications reject format 4 instead of silently dropping
the new structure.
