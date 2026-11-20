---
title: "See GOOSE messages on the SLD"
description: "Add IED components to the single-line diagram, show GOOSE lines between them, open an IED to read its messages, and change a value to send a GOOSE message."
section: sld
order: 5
video: goose-sld
---

GOOSE is a fast message that IEDs send each other, for example to trip a breaker. The SLD can show
who sends each GOOSE message and who receives it.

## Add the IEDs

Each IED that sends or receives GOOSE gets one **IED component** on the drawing.

1. Press **Add**, then **Add all GOOSE IEDs** in the **IEDs and GOOSE** group. One component appears
   for each IED that sends or receives GOOSE, in a row above the drawing.
2. Press **Undo** to remove them all in one step.
3. To add IEDs one at a time, press the button that shows how many IEDs are **not on the drawing**.
   Press **Add** next to an IED to add its component.

## Show the GOOSE lines

1. Press the **GOOSE messages** switch in the toolbar.
2. A line runs from the IED that sends a message, through the sender, to every IED that receives it.
3. Press the switch again to hide the lines.

The **Who is reading** switch works the same way for the SCADA and test programs that read each
simulated IED over MMS. It shows their IP addresses.

## Read an IED's messages

1. Click the IED component. Its sheet opens on the right.
2. **Messages it sends** lists each GOOSE message, value by value.
3. **Messages it receives** lists the messages it listens to.
4. Each message says **To** and the names of the IEDs that receive it. If none does, it says so.

The component itself shows a short summary, such as **Sends 2 messages · gets 1**.

## Send a GOOSE message

1. Press **Run** and wait until the IED shows **Simulated · running**.
2. In the IED sheet, change a value of a message.
3. The IED raises the message's state number once, and every receiver gets the new message.
4. The line on the drawing shows the message travelling from the sender to its receivers.

A real device is watch only. ScadaBench never sends GOOSE for a real device. Its component says
**Real device · watch only**.

## Messages that are not sent

Some GOOSE blocks in a file are switched off, have no data set or have no network address. The sheet
shows why, for example **Not sent (disabled in file)**. This is normal for spare inputs.

For more on GOOSE and the network, see the Tools section.
