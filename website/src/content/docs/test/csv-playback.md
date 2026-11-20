---
title: "Play a CSV recording"
description: "Load a CSV file with a time column and a value column, choose how values between samples and after the last sample behave, and play it on a measurement while the IEDs run."
section: test
order: 4
---

A **CSV recording** lets a measurement follow real values over time. Use it to replay a fault
record, a load curve or any other time and value log, exactly as it was recorded.

## Prepare the file

The file must be a UTF-8, comma-separated CSV file with column headers in the first row.

- A **time** column, in seconds from 0. Time must go up strictly, row by row.
- A **value** column with the numbers to play. Values use the units of the signal you play on. No unit
  conversion is done, so write the values in the signal's own units.

A file can hold up to 100,000 rows and 5 MB. ScadaBench picks the columns named `time` and `value` when
they exist; otherwise it uses the first two columns. You can choose the columns yourself.

## Load the recording

1. Open a generator with **How the value changes** set to **Play a recording (CSV)**. You can also
   use the same choice in the **How the value changes** section of an SLD symbol.
2. Press **CSV trace** and choose your file. ScadaBench reads its columns at once.
3. Choose the **Time column** and the **Value column** from the two lists.
4. Choose what happens **Between samples**:
   - **Hold previous value**: the value stays the same until the next sample. Use it for status-like
     data.
   - **Linear interpolation**: the value slides smoothly from one sample to the next. Use it for
     measurements.
5. Choose what happens **After last sample**:
   - **Hold final value**: the value stays at the last sample.
   - **Repeat trace**: the recording starts again from the beginning.
6. Press **Apply CSV playback**.

The message says **Trace saved in the project. Run or resume automation to play it.** The trace is
stored with the project, so you do not need the file again.

## Play it

1. Press **Run** at the top right.
2. The measurement follows the recording from the start.
3. Press **Stop** to end the test.

## If playback does not start

- Check that the generator or symbol is still set to **Play a recording (CSV)**.
- Check that the IEDs run. A stopped IED has no live value.
- Check that the file has numbers in the value column. Text in that column cannot be played.
