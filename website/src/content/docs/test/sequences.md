---
title: "Timed sequences"
description: "Build a timed sequence that sets values or checks values at set times while the IEDs run, and start it by hand or automatically with Run."
section: test
order: 3
video: test-station
---

A **sequence** is a list of steps. Each step happens at a set time after the sequence starts. Use it
to repeat the same test every time, without pressing buttons at the right moment.

## Create a sequence

1. Click the **Test** place, then the **Sequences** tab.
2. Press **New sequence**.
3. Type a **Sequence name**, for example `Breaker trip test`.
4. Tick **Start automatically with Run** if the sequence must start each time you press Run.

## Add steps

For each step:

1. Type the **Time (seconds)** from the start, from 0 to 86400. Decimals are allowed, such as 2.5.
2. Choose the **Action**:
   - **Set value**: writes the value you enter into the signal.
   - **Check value**: looks at the signal at that time and does not change it.
3. Choose the **Device**, then search for the **Signal** in **Search references**. Use the
   **Point type** list to narrow the search.
4. Type the **Value**. For a true or false signal, type `true` or `false`. For a number, type a
   finite number.
5. Press **Add step**.

The steps appear in a table: **Seconds**, **Action**, **Reference**, **Value**. Press **Remove** on a
row to delete that step.

If a step is not valid, the form says why, for example **Choose a signal from the search results.**

## Run a sequence

1. Press **Run** at the top right. The IEDs start.
2. Press **Run sequence** on the sequence. Its steps start at the times you set.
3. The Test place shows **1 sequence running** while it plays.

**Delete** removes the whole sequence. Press **New sequence** to start another one.

You can edit a sequence only while the IEDs are stopped. The page says **Stop the simulation to edit
sequences.**

## Repeat a recorded test

To play a recording of real values instead of typed steps, use a CSV trace. See
[Play a CSV recording](/docs/test/csv-playback/).
