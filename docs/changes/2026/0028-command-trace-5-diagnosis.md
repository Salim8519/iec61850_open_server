# 0028: Command trace 5 breaker diagnosis

Date: 2026-09-16. Follow-up evidence for
[ADR-0018](../../decisions/ADR-0018-concen-sbow-compatibility.md).

The user identified `commmand_traces_5.pcapng` as an Open attempt that left CB-1
Closed. Packet and runtime evidence agree that transport and MMS acceptance worked:

- Capture SHA-256 is
  `544C0C8591B354A19B6CF4AECA809C9E14499A3F9A88976A2F659C7B560B0ABD`.
- Frame 994 at 23:10:40.370307 AST writes Boolean false to
  `C03_Y1105System/Bkr0XCBR1$CO$Pos$SBOw` with origin `CONCEN`.
- Frame 997 at 23:10:40.370728 is the successful MMS write response.
- No matching `Oper` appears in the 15.289-second capture.
- Run `28f4855e31e84f73b09e14c2caafe624`, event lines 194-199, records the
  explicit profile adapting false to true, executing Close, changing zero mapped
  points and republishing Dbpos 2 (Closed).

The enabled `concen_inverted_sbow` profile therefore caused the Open failure; this
was not a concentrator connection error. The earlier trace was identified as a
Close attempt but used the same origin, object, trigger and false value. Those two
stated intentions cannot be distinguished from the available wire data. A
current-state toggle would guess and could reverse a repeated/no-op command.

Expected correction is client-side standard enhanced SBO: false for Open, true for
Close, followed by matching `Oper`. If client correction is impossible, a new
explicit policy needs its own failure-mode decision, format migration and
independent two-direction evidence. The current inversion profile remains stored
on CB-1; this diagnosis did not silently mutate the project.

The capture stayed local. The current handover records the autosave, recovery,
binary and log identities needed for continuation.
