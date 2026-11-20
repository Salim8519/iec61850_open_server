---
title: "Link symbols to IED signals"
description: "Connect each SLD symbol to the IED signal it shows, such as a breaker position or a measurement, using the properties panel and the signal picker."
section: sld
order: 3
video: point-panels
---

A symbol on the drawing is only a picture until you link it to an IED. Once linked, it shows the
live value of that signal, and it can send commands to the IED.

## Open the properties of a symbol

Click the symbol. Its properties open on the right. The panel has one section for each kind of link:

- **... comes from**: the signal that shows the symbol's state. For a breaker, the heading is
  **Position comes from**.
- **Command goes to**: the IED control that the symbol sends when you operate it.
- **Signals on this relay**: the status signals of a relay symbol.
- **Measurements in this list**: the measurements shown by a measurement list.
- **Points in this panel**: the points of a point panel, see below.
- **How the value changes**: a generator or a recording that moves the value. See the Test section.

If the symbol still needs a link, the panel says **Connect this symbol to a measurement first.** and
shows a status such as **Needs a link**.

## Pick a signal

Every link uses the same signal picker.

1. Choose the **Device**: the IED, such as `Feeder01`.
2. Type part of a name in the search box, for example `XCBR1` or `Pos`. The list narrows as you type.
3. Choose the signal in the list. It is shown with its full path, such as `Feeder01CTRL/XCBR1.Pos`.
4. The picker chooses the point type that the symbol needs, and says why. Change the **Point type**
   only when you know you need another kind.

For a breaker, set **Position comes from** to `Feeder01CTRL/XCBR1.Pos`. Set **Command goes to** to the
breaker's control in the same IED, so that Open and Close reach it.

## Point panels

A **point panel** is one card that holds the points of one IED. Use it when a drawing would otherwise
need many small symbols.

1. Press **Add**, then choose **Point panel** under **Protection & control**.
2. Open the panel's properties. Under **Points in this panel**, add points by type (status, double
   point, value or command) and by object, such as `XCBR1`.
3. Press **Run**. The panel shows the live value of each point. You can change a value or send a
   command from the panel, as you can from a symbol.

## Undo a wrong link

Press **Undo** if you linked the wrong signal. Each change to the drawing can be undone, one step at a
time.
