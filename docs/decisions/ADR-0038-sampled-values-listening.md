# ADR-0038: Listen to Sampled Values (no publishing)

Accepted: 2026-10-14. This replaces "Sampled Values remain off" in
[ADR-0030](ADR-0030-goose-with-user-installed-npcap.md). Specification:
[Sampled Values](../specs/sampled-values.md). Owner decision: listen only.

**Decision**
- **Build:** `CONFIG_IEC61850_L2_SMV` is on. `cmake/sv_support.cmake` adds only the
  unmodified upstream `sv_subscriber.c` to the library. `sv_publisher.c` is compiled
  only into `studio_sv_test_publisher`, a test-only tool that is never packaged.
- **Listener:** `studio_live` gains `sv_start`/`sv_stop` (the read-only table, LIVE-01,
  still has no writes).
  - It decodes the expected streams with libiec61850, counts loss from `smpCnt` and
    computes RMS and phasor over whole cycles.
  - It reports at most twice a second per stream.
- **Passive header reader:** libiec61850 1.6.1 exposes neither unknown streams nor the
  Ed2 simulate bit. A passive reader on the HAL Ethernet API therefore parses only the
  frame header and the first svID. It never transmits.
- **Expectations:** these come from `SampledValueControl`, the SMV address and the
  project's nominal frequency.
  - The frequency is a new `network.nominal_hz` setting (50 or 60, default 50, shown in
    the UI). Older projects open with 50 Hz; the project format is unchanged.
  - A mode-only network change keeps the frequency, and a frequency change keeps the
    mode.
- **Interpretation:** values are interpreted only for recognised 9-2LE layouts
  (8 × INT32 + q, TCTR 1 mA, TVTR 10 mV); other layouts show raw numbers.

**Compatibility:** additive.
- IEDs gain an `sv` entry.
- The import warning "SV retained; publishing is not active" is removed.
- An older `studio_live` without `sv_start` fails only the SV listener, with a visible
  message.
- The native programs must be rebuilt.
