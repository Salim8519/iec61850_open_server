# ADR-0149: Devices commands to a real IED: every class, one click or step by step, sent in place

Accepted: 2026-11-23. Owner, 2026-11-23, for 0.1.37, after testing 0.1.36 on two live GE BCUs:
"add select before operate, or operate directly thing", "when user sends command in the pop up
command dont change his scrolled position", "add search command", "allow sending command direct
without need to open command", "make sure the scadabench can handle many kinds of commands in ied
mode". Change [0239](../changes/2026/0239-devices-command-steps.md). Amends ADR-0138.

## Problem

On the live relays 0.1.36 could monitor but its Devices commands stopped short:

- Only SPC/DPC On/Off and Close/Open existed; INC, ENC, APC, BAC, BSC and ISC of the CID were not offered.
- A select-before-operate command was always one press; there was no way to Select, see the
  reservation and then Operate or Cancel, as on the SLD (change 0173).
- The device's refusal reached the person as raw protocol words ("object-value-invalid",
  "type-inconsistent (blocked by interlocking)"). A relay in Test mode refused every command
  because Devices always sent Test = false, and nothing said why.
- The pop-up swapped its list for the question, so every send jumped back to the top; it had no
  search for a relay with ~300 commands, and a Data points row could not send.

## Decision

1. **Engine (Go).** `POST /live/{key}/command {action: select|operate|cancel, ref, value, options,
   confirm, selected}` (`realcmd.Service.Command`). It finds the control in the IED's `controls`
   (SPC/DPC) or its typed list from the stored SCL (`scl.CachedExtendedControls`), refuses in words
   exactly as ADR-0138 does (Allow commands off, not Live, not in the CID, not confirmed; Cancel
   needs no confirmation), turns the value into the typed ctlVal (`typedcmd.Wire`: ENC labels,
   ISC Raise/Lower from the read position) and sends the step through the shared `studio_control`
   typed session with the device's own control model. `selected: true` (step by step) makes an
   Operate of an SBO control require its Select; otherwise Operate selects and operates in one go.
   A Select answers the Operate window (`typedcmd.OperateWindow`, moved from runtime unchanged).
   `/live/{key}/operate` stays for compatibility.
2. **Plain-English refusals.** `realcmd.Explain` names what every AddCause and the common MMS
   errors mean and what to try; on a refusal the engine reads the node's `Beh.stVal` over the live
   session and says when the relay is in Test mode (tick Test command), when a test command meets
   a relay not in Test mode, or when the function is Off. The device's own answer follows the words.
3. **Events.** `real_command_requested` (with action and test), `real_command_selected`,
   `real_command_applied`, `real_command_canceled`, `real_command_failed`, `real_command_observed`
   share one correlation id. The feedback is observed for SPC/DPC/INC/ENC/ISC values; set points
   and Tcmd steps report where the feedback went.
4. **Card.** The typed session keeps one association per (local card, device) and binds the chosen
   card (`live.local`, change 0233) like the 0.1.36 operate path. `studio_control` says
   `typed_local`; the client refuses a card-bound request to a worker without it.
5. **Screen.** The Commands pop-up lists every command with a search (reference or class) and the
   shared Command options (Test, interlock check, synchro check, originator, and the app-wide "Show
   Select before Operate step by step" switch of change 0173). Defaults are what 0.1.36 sent
   (interlock check on, station control) with Test following the relay's Test mode until changed;
   they are kept per IED for the session. Each command asks in its own row and answers there, so the
   list never jumps. The `.Oper.ctlVal` row of a command in Data points gets the same buttons while
   the IED is Live with Allow commands on. Inputs reuse the point panel's number and list boxes,
   StepButtons and SboCountdown.

## Compatibility

Projects are unchanged. The command path for real IEDs moves from the Tools `operate` lane to the
typed lane of the same `studio_control` process; the default options keep 0.1.36's wire content
(orCat 2, interlock check on, orIdent ScadaBench). The worker and engine ship together; the new
capability guards a mixed build. SLD and Test still refuse commands to real IEDs; the gateway rule of
ADR-0138 is unchanged.
