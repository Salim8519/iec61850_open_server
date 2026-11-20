---
title: "Operate and test equipment"
description: "Use Test to force a value in a simulated IED without sending a command, and Operate to send a real IEC 61850 command such as Open or Close, with Select before Operate."
section: sld
order: 4
video: commands
---

While IEDs run, the drawing is live. You can then do two different things with a symbol. They look
alike, but they do different jobs.

| Button | What it does | Is a command sent? |
| --- | --- | --- |
| **Test** | Changes the value in the simulated IED, as if the plant moved by itself. | No. No command is sent. |
| **Operate** | Sends the command that the symbol is linked to, such as Open or Close. | Yes. A real IEC 61850 command. |

## Test a symbol

1. Press **Run**.
2. Right-click the symbol. A menu titled **Test** opens, with the test buttons for this symbol.
3. Press a test button. The simulated IED changes at once, and the symbol follows.

You can also open the symbol's properties and find the **Test** section there. The menu and the
section both say **forced change, no command**.

Test buttons only work on simulated IEDs. A real IED is never changed by a test button.

## Operate a breaker

A breaker needs a **Command goes to** link first. See
[Link symbols to IED signals](/docs/sld/link-to-signals/).

1. Press **Run**.
2. Click the breaker and find **Operate** in its properties. It is tagged **real MMS command**.
3. Press **Open** or **Close**. The command goes to the IED. The breaker moves only when the IED
   accepts the command.

If the symbol has no command link, the panel says **Add a command link to operate it like SCADA.**

## Select before Operate

Some IEDs accept a command in two steps: first Select, then Operate. ScadaBench reads this from the
IED's configuration, and shows the time the IED allows.

- **Off (normal)**: one press selects and operates, the way a station HMI works.
- **On (step by step)**: for testing. The first press only selects. Then press **Operate** or
  **Cancel** before the time runs out. A countdown shows how long you have.

Turn the step-by-step option on under **Show Select before Operate step by step**, in
**Command options**. It applies to the whole drawing and to point panels.

## Command options

The **Command options** panel sits under the Operate buttons. Most tests leave it off.

- **Test command**: marks the command as a test. A device in test mode acts on it. A device in
  normal mode ignores it.
- **Interlock check** and **Synchro check**: ask the device to check interlocking or synchronism
  before it moves.
- **Originator category**: who sends the command. **Station control** is usual for a local SCADA.
- **Originator name**: up to 64 characters that the device logs as the sender.

The options are sent with the next command and remembered for this symbol.

## Push buttons, trip relays and tap changers

- **Push button**: while the IEDs run, a left-click presses it. Its panel shows **Press** or
  **Toggle**, depending on the button type.
- **Trip relay**: while the IEDs run, its panel shows **Reset** and **Operate** buttons.
- **Tap changer**: its panel shows **Raise** and **Lower** buttons. Raise and Lower send one step
  command each.
