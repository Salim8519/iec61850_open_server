---
title: "Compare a real device with its SCD"
description: "See where a real IED differs from its SCD file, and check every IED at once with the Health check and the Signal flow views."
section: devices
order: 4
---

A real device should match its configuration file. When it does not, the test result is wrong, so
check it first.

## Compare one device

1. Connect the real IED: choose **Monitor real device** and press **Run**.
2. Wait until the IED shows **Live**.
3. Open the **Compare with SCD** tab.
4. ScadaBench reads the device model and lists every point where the device and the SCD file
   disagree. Typical differences are a data point with another name, a different type or a missing
   report block.

If the list is empty, the device has exactly the data points, types, data sets and report blocks that
the SCD describes. If the list is long, the device probably runs another configuration. Ask the owner
of the device which file is current, then import that file again.

The comparison runs only while the device is live. Until then, the tab says what it is waiting for.

## Check the whole station

Under **Whole station** in the list on the left, choose:

- **Health check**: answers "Does the real station match the SCD?" It gives one result per IED. It
  shows whether ScadaBench can reach each device on port 102, and the other checks in the list.
- **Signal flow**: shows who sends GOOSE and reports to whom, according to the SCD. The same view
  is also on each IED's **Signal flow** tab.

Use the Health check before a test day. Fix anything marked as a problem, then run again.
