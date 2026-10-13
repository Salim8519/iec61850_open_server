# 0020: Typed switch controls, constructed state and trip relay

Date: 2026-09-16. Implements
[ADR-0011](../../decisions/ADR-0011-typed-equipment-controls.md).

Added a distinct latching Trip relay drawing item with one SPC command: Operate
sends true and Reset sends false. CB, DS and ES command pickers now accept only
imported DPC control objects. State and command mappings are independent, so
command-only equipment and cross-IED feedback are valid.

Switch state can use native Dbpos, one Boolean, or two arbitrary Boolean contacts.
The coordinator constructs one authoritative canvas value and publishes it as an
`equipment_state` event. The frontend renders that value and does not reproduce
the contact truth table. Constructed state never creates or rewrites MMS nodes.

SCL import now exposes a control only when the CDC, `stVal`, and Boolean
`Oper.ctlVal` are mutually consistent. The native worker checks the same contract.
DPC false/true commands update feedback to Dbpos Open/Closed; SPC true/false
updates Boolean feedback and remains latched until the reset command.

Project format 4 stores typed nested mappings. Legacy Boolean switch bindings are
migrated to the fixed one-Boolean interpretation; native Dbpos is retained.
Unsupported integer/enum switch mappings and incompatible controls are removed
with a visible notice only after preserving and verifying the exact input archive.

Evidence recorded during implementation:

- Native Release build completed for `studio_ied.exe` and `studio_probe.exe`.
- Seven local independent-client protocol tests passed for DPC Open/Close, latched
  SPC Operate/Reset, report delivery, stale guards and initial equipment state.
- Twenty-four focused backend/unit/integration tests passed after the mapping
  contract update; pure tests cover all four two-contact states and missing input.
- Frontend TypeScript/Vite production build passed.
- The final maintained suite passed all 103 tests, including two rebuilt portable
  package checks and the independent loopback protocol tests.

Clean-machine and browser visual acceptance remain release gates. Local loopback
evidence is not third-party interoperability certification.
