# 0061: Sampled Values listening

Date: 2026-10-14. Roadmap feature D1. Specification: [Sampled Values](../../specs/sampled-values.md).
Decision: [ADR-0038](../../decisions/ADR-0038-sampled-values-listening.md).

## Change

**Native**
- `live/sv_traffic.*` handles subscribe, supervise and report, decimated to 2 Hz.
- `live/sv_header.*` is the passive header reader.
- `live/sv_math.hpp` provides RMS/phasor over complete cycles and `smpCnt` loss with
  wrap.
- `main.cpp` adds `sv_start`/`sv_stop`.
- `cmake/sv_support.cmake` and the test-only `testtools/sv_test_publisher.cpp`.

**Backend**
- `scl/communication.py` reads SMV addresses; `scl/sv_controls.py` reads
  `SampledValueControl` and recognises the 9-2LE layout.
- `domain/sv_supervision.py` holds the expectations and the problems.
- `acquisition/sv_monitor.py` starts with Run.
- `GET /sv`.
- `network.nominal_hz`, with safe merging of partial network updates.

**Frontend:** the Settings → Network → Sampled Values card:
- 50/60 Hz switch;
- stream table (status, rate against the SCD, lost samples, clock);
- per-channel RMS, fundamental and angle;
- not-supervised and unknown streams.

## Evidence

On the VMware lab adapter, with the new build:
- **Protocol tests:** 36 passing, including 4 SV wire tests (values, decimation, loss
  across wrap, simulate bit and unknown stream).
- **Values:** 1 A / 63.5 V → 1.0000 A and 63.5003 V, angles 0°/−120°/+120°.
- **Load:** 4 streams at 16,000 frames/s use 15.3% of one core, with no loss.
- **Python unit and integration:** 236 passing. **Frontend:** 67 passing.
- **Browser:** the stream from `open_substation.scd` (IED3_SMV) is listed. Switching to
  60 Hz changes the expected rate from 4000/s to 4800/s.

**Bugs found and fixed while testing**
- The test publisher crashed: `SVPublisher_ASDU_setSmpSynch` was called before
  `setupComplete`, and Npcap wasn't loaded first.
- RMS was off by about 1% when a cycle had a lost sample. Such cycles are now dropped.

## Test tolerance

The test publisher paces itself with Windows sleeps and ran at 3,891 samples/s once
under full-suite load. Its wire tests accept 3,600–4,400/s. The 2 % rate rule is
checked exactly in the unit tests.

## Limitations

- No real merging unit was tested.
- No waveform view.
- Station health doesn't include SV yet.
- The normal `build/Release` binaries still need the rebuild in `studio/native/README.md`.
