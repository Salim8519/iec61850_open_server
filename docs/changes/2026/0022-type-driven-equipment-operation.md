# 0022: Type-driven equipment operation and mapped feedback

Date: 2026-09-16. Implements
[ADR-0013](../../decisions/ADR-0013-type-driven-equipment-operation.md).

The equipment inspector now sends Open, Close, Operate and Reset through a
node-aware API. The backend resolves the node's imported DPC or SPC control,
sends the real MMS command, then updates the separately selected state mapping
through the acknowledged sample writer. Native Dbpos, one Boolean, two Boolean
contacts and Trip relay Boolean feedback are supported without reference-name
rules. Command-only equipment remains valid.

The prior failure was reproduced with a CB whose command was mapped to an XCBR
control while its display state was mapped to a different XSWI status. MMS
accepted the command and changed only the control's own native status, so the
drawing correctly continued to display the unchanged selected state. The new
operation path updates that selected simulation state only after MMS acceptance.

The state picker now shows runtime-editable imported signals directly; the
confusing free-text "Find signal name or path" field was removed. For one-point
switch state, the selected SCL type automatically chooses native Dbpos or the
fixed Boolean-to-Dbpos mapping. Command selection continues to filter imported
control objects by CDC, independent of vendor naming.

Evidence:

- Frontend TypeScript and Vite production build passed.
- Ten focused tests passed for generic CDC selection, all feedback modes,
  editable-signal filtering and the node-aware API.
- The native protocol regression used separate XCBR command and XSWI Dbpos state
  objects. Independent `studio_probe` reads observed distinct Closed and Open
  values, while runtime reads and equipment-state events recorded 2 then 1.
- The complete maintained Python suite passed all 107 tests, including native
  loopback protocol and portable packaging coverage.
- Maintained file-length validation passed.

Final browser workflow acceptance remains pending. Local loopback evidence is
not third-party interoperability certification.
