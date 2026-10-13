# 0026: Independent Trip state and enhanced-SBO lifecycle

Date: 2026-09-16. Implements
[ADR-0017](../../decisions/ADR-0017-trip-polarity-and-sbo-observability.md).

Project format 7 separates Trip command polarity from Trip state polarity. The
default and migrated behavior is Operate sends 0, Reset sends 1, Operated state is
1 and Reset state is 0. Format-6 command polarity remains explicit while its new
state polarity is initialized operated-high. Every older archive is byte-preserved
and ZIP-verified before the migrated project is written.

Studio and external SPC commands now resolve the semantic action from command
polarity before deriving feedback from state polarity. This includes a Trip state
mapped back to the command object's own native status and every relay sharing the
same command. Accepted no-op commands force an equipment-state event, and the new
`GET /api/v1/equipment/states` snapshot rehydrates the canvas after SSE connection
or resynchronization.

The native worker publishes enhanced-SBO selection lifecycle events and advertises
a required capability marker. Event Log explains that selection is waiting for
`Oper` and shows the imported timeout. Selection never moves the breaker; repeated
`SBOw` remains rejected. The laboratory template now has the standard Cancel shape,
without the Oper-only Check field.

Evidence:

- All 118 maintained Python tests passed against both isolated candidate binaries
  and the rebuilt normal worker/probe path.
- All 18 protocol tests passed separately against the rebuilt normal binaries.
- Eleven frontend contract tests and the production TypeScript/Vite build passed.
- File-length, architecture and documentation verification passed.
- An isolated copy of the supplied C03 recovery project restored one IED and all
  seven diagram nodes; its migration backup was byte-identical.
- Its imported `CB-1` DPC control reported model 4 and a 30,000 ms timeout. An
  independent MMS probe observed selection without movement, add-cause 19 on a
  repeated `SBOw(false)`, and Open only after matching `Oper(false)`; native events
  were selected then operated.
- Generic enhanced-SBO evidence also covered cancel, timeout, failed operation and
  selecting-client disconnect events.

Supplied packet captures and customer engineering archives remained local and were
not added to project data. No packaged executable was built. Interactive browser
acceptance and third-party-client interoperability remain release gates.
