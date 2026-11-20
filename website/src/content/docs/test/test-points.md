---
title: "Test points"
description: "Save the alarms, statuses and measurements you change most as test points, then change them live on the Test page while the IEDs run."
section: test
order: 1
video: simulation
---

The **Test** place is for changing values on purpose. A **test point** is one signal you want to
change during a test: a breaker position, an alarm, a measurement. Test points are saved with the
project, so they are still there the next time you open it.

## Add a test point

Add test points while the IEDs are stopped.

1. Click the **Test** place. The **Test points** tab is open.
2. Press **Add test point**.
3. Search for the signal. Only signals you can change are listed.
4. Choose the signal and press **Add**.

If the IEDs run, the button is grey, with the tooltip **Stop the run to add test points**.

## Change a value while the IEDs run

1. Press **Run** at the top right.
2. Each test point shows its current value. Change it with the control on its row.
3. The change goes to the simulated IED at once. Clients connected to that IED see the new value.

A value you cannot change is read only. Test buttons never change a real device.

## Pulse a status

A status that is True or False can be pulsed. A pulse sets the value for a short time, then sets it
back.

1. While the IEDs are stopped, open **Pulse settings** on the test point.
2. Choose the **Pulse value** (True or False) and the **Pulse length (ms)**, from 100 to 60000.
3. Run the IEDs. Press **Pulse True** or **Pulse False** on the row.

## Generators drive a value

If a generator is moving a value, the row says **Generator**. Your changes are replaced by the
generator until you press **Resume generator** on that row. See [Generators](/docs/test/generators/).

## Order, remove and filter

- Use the **up** and **down** arrows to change the order. Use the remove button (trash icon) to delete a test point.
  Both are available only while the IEDs are stopped.
- When you have many test points, the **Filter test points** bar appears. Type part of a name to
  narrow the list, or press **Clear** to see all of them.
