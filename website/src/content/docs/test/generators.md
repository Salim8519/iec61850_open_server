---
title: "Generators"
description: "Make a measurement move by itself with a generator: a fixed value, a smooth wave, a ramp, steps, random noise, or a recorded CSV trace."
section: test
order: 2
video: test-station
---

A **generator** moves a measurement by itself while the project runs. For example, it swings a
current up and down, so your SCADA sees changing values without anyone touching the test.

A generator works only on measurements with a number value.

## Create a generator

1. Click the **Test** place, then the **Generators** tab.
2. Press **New generator**.
3. Choose the **Device**, the IED that holds the measurement.
4. Search for the measurement in **Search measurements**, then choose it from the **Measurement** list.
5. Choose **How the value changes**. The options are listed below.
6. Set the numbers for that behaviour.
7. Press **Save generator**.

The **Generators** table lists each generator with its behaviour. Press **Cancel** to leave the form
without saving.

## How the value changes

| Choice | What happens |
| --- | --- |
| **Set by hand** | The value only changes when you type a new one. |
| **Fixed value** | The value stays at the number you enter. |
| **Smooth wave** | The value swings up and down around the **Middle value**. |
| **Ramp** | The value climbs steadily, then starts again each cycle. |
| **Steps** | The value jumps between low and high each half cycle. |
| **Random noise** | The value moves randomly around the **Middle value**. |
| **Play a recording (CSV)** | The value follows a recorded time and value file. See [Play a CSV recording](/docs/test/csv-playback/). |

For the wave, ramp, steps and noise choices you also set:

- **Middle value** (or **Value** for a fixed value).
- **Swing (±)**: how far the value moves away from the middle.
- **Cycle time (seconds)**: how long one full cycle takes. It must be at least 0.1 second.

## Run and watch

1. Press **Run**. The generator starts with the project.
2. The **Value now** control shows the value that the generator sends.
3. If you change a value that a generator drives, the row shows **Resume generator**. Press it to hand
   control back to the generator.

If you choose **Set by hand**, the message says **You now set this value by hand.** Nothing moves by
itself in that case.
