# ADR-0016: Reset-high legacy Trip migration and isolated live canvas values

Status: accepted, 2026-09-16.

An independent packet capture shows the MMS object as the `$Oper` member of the
imported SPC control, with a Boolean `ctlVal`. The client text suffix `-1` represents
value 1; it is not part of the object reference. Existing format-5
projects may retain Operate=1 / Reset=0 from the earlier Trip relay behavior, which
inverts the sender's Operate=0 / Reset=1 convention.

Project format 6 establishes a one-time compatibility boundary. Opening formats
1 through 5 first writes and verifies an exact recovery archive, then changes every
legacy Trip relay to Operate=0 / Reset=1. The migration notice names each relay whose
effective polarity changed and exposes the recovery path. Format-6 polarity remains
an explicit engineering choice, including Operate=1 / Reset=0. No IED or reference
name is used to infer polarity.

Continuous events previously recreated the controlled React Flow node array and
replaced the value context. React Flow could temporarily lose measured dimensions
and hide nodes while measuring again. Live equipment values now use a keyed external
store, so only the subscribed node receives its update. The canvas boundary, node
projection, callbacks and configuration remain stable across Event Log and signal
traffic. Every projected node has safe initial dimensions before measurement.

The selected equipment ID remains the only selection authority. Both React Flow's
selected flag and the Properties inspector derive from it; live values never alter
selection or persisted diagram data. Control feedback events display the exact
control reference, raw `ctlVal`, node-specific semantic action and whether mapped
feedback changed.

This refines [ADR-0014](ADR-0014-equipment-actions-and-source-editor.md) and
[ADR-0015](ADR-0015-native-control-feedback.md). HTTP and native-worker contracts
are unchanged. Older Studio builds reject format 6 rather than misreading it.
