# Change 0008: CSV measurement playback

Date: 2026-09-09.

Source configuration now accepts CSV traces with explicit time/value columns,
hold/linear interpolation and repeat/hold-final behavior. The UI previews columns
and five rows before activation. Reference search now reaches measurements beyond
the first page of an IED, including numeric integer measurements.

Input contract: UTF-8 comma-separated text with unique headers, at most 5MB and
100,000 rows. Time starts at zero seconds, strictly increases, and stays within
24 hours. Numeric values must fit the target type. Linear interpolation is only
available for floating-point measurements. No formulas or unit conversions run.

Original CSV text and resolved samples persist in .subsim; restore reparses the
text instead of trusting stored samples. Browser project snapshots and source
events omit raw CSV/sample arrays, preventing large traces in logs or routine
responses. Traces evaluate at the same 10Hz automation cadence and acknowledge
native updates. They are not a precision waveform or Sampled Values publisher.

Applying a source during a run restarts its local source time at zero. Pause freezes
playback. The automation clock now excludes IED startup time. Run restarts saved
traces from zero; manual overrides suspend them until the source is reapplied.

Unit/integration coverage includes interpolation, endpoints, repetition, activation
time, malformed/duplicate/nonfinite samples, integer constraints, preview and project
round-trip. Native CSV playback and pause are checked with an independent MMS read.
Visual browser verification and the full release acceptance remain outstanding.
