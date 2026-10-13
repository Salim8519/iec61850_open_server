# 0032: Open switch start positions and connector-free push button

Date: 2026-09-17. User-requested customization of equipment start behaviour
and symbol drawing.

CB, DS and ES equipment now declares a persisted `initial_position` of `open`
(default) or `closed` in the project diagram. At run start the coordinator
applies each mapped switch's start position to its exact state points through
the acknowledged sample writer, before the first binding publish, so a mapped
switch starts Open instead of the SCL-missing "Intermediate" zero value. The
start position intentionally takes precedence over SCL-specified initial
positions for mapped switching equipment; the SCL `<Val>` data itself is still
retained in the imported model and editor.

Operating equipment through the studio now records the resulting position as
the node's start position, so the state a user leaves a switch in is restored
on the next run. An Inspector "Start position" select lets the user choose
Open or Closed while stopped. The push button symbol lost its two vertical
connector stubs.

Evidence:

- 90 unit tests pass, including a new `test_initial_state.py` covering the
  default contract value, per-mapping-mode start application and skip rules.
- 17 integration and 21 native loopback protocol tests pass.
- `studio/tools/verify.py` passes file-length, architecture, documentation and
  history checks. Frontend TypeScript and Vite production build pass.

Browser visual acceptance and an independent third-party MMS observation of
the persisted restart behaviour remain release gates.
