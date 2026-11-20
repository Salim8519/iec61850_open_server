---
title: "Your first run"
description: "Add the built-in sample IED Feeder01, press Run to start it, check that it answers on its own address, then press Stop."
section: getting-started
order: 3
---

This walk-through uses the built-in sample IED, **Feeder01**. You do not need a file. It takes
about five minutes.

## 1. Add the sample IED

1. Click the **Devices** place in the left menu.
2. In the empty project, press **Use a sample IED**.
3. **Feeder01** appears in the list on the left. Click it to see its details on the right.

If you already have IEDs in the project, press **Add a sample IED** below the list instead.

## 2. Press Run

1. Look at the top right of the window. Press **Run**.
2. The button changes to **Stop**, and the run state says **Running · 1 IED**.
3. The IED is now live. ScadaBench starts its server, its reports and its GOOSE messages.

While IEDs run, the drawing and the IED roles are locked. This keeps the test safe. Press **Stop**
to edit again.

## 3. Check the address

1. Click **Feeder01** in the list.
2. Under its name, the line that starts with **Simulate** shows its address and port. The port is
   **102**, the standard port that SCADA uses for IEC 61850.
3. Give this address to your SCADA, gateway or test tool. It connects to Feeder01 the same way it
   would connect to the real relay.

## 4. Look at the data

1. Click the **Data points** tab of Feeder01.
2. You see the values the IED publishes, such as the position of its breaker, `XCBR1.Pos`.
3. To change a value while the IED runs, go to the [Test](/docs/test/test-points/) place.

## 5. Stop

1. Press **Stop** in the top right.
2. Feeder01 stops answering. The button shows **Run** again.

Stop is the same for every IED. To stop only one IED, use the buttons in its Devices details, as
explained in [Simulate and run IEDs](/docs/devices/simulate-and-run/).

## Next

- [Import your own CID or SCD file](/docs/devices/import-cid-scd/)
- [Save your project](/docs/getting-started/saving-the-project/)
