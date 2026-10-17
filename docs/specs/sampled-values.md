# Spec: Sampled Values listening

Decision: [ADR-0038](../decisions/ADR-0038-sampled-values-listening.md), which replaces the
"Sampled Values remain off" part of [ADR-0030](../decisions/ADR-0030-goose-with-user-installed-npcap.md).
Change: [0061](../changes/2026/0061-sampled-values.md).
Owner decision (2026-10-14): **listen only**. Studio never publishes SV.

## Purpose

On a process-bus station, the engineer can see every IEC 61850-9-2 Sampled Values
stream on the lab adapter and check it against the SCD:
- Is it there, at the right rate, synchronised, without lost samples?
- Does it carry plausible currents and voltages?

## Inputs (IEC 61850-6)

| Element | Used for |
| --- | --- |
| `SampledValueControl` (`name`, `datSet`, `smvID`, `confRev`, `smpRate`, `nofASDU`, `smpMod`, `multicast`) | Expected stream |
| `SmvOpts` | Which optional header fields are expected |
| `Communication/.../SMV` address (MAC, APPID, VLAN-ID, VLAN-PRIORITY) | Matching frames |
| `DataSet` of the SvCB | Channel names, order and type |
| `ExtRef` with `serviceType="SMV"` | Subscribers (shown in [Signal flow](signal-flow.md)) |

## Behaviour contract

**Native**
- libiec61850 owns frame decoding: build with `CONFIG_IEC61850_L2_SMV` on, using the
  pinned version, the same way GOOSE is built.
- The SV subscriber runs inside `studio_live`, on the same adapter and Npcap rules as
  GOOSE. It is receive-only, and no packaged executable contains SV publishing code.
- libiec61850 1.6.1 exposes neither unknown streams nor the Ed2 simulate bit, so a
  passive header reader (HAL Ethernet API, receive only) reads APPID, MAC addresses,
  VLAN, reserved1 bit 15 (simulate) and the first svID. It never sends anything.
- Per-frame work stays in native code. Python receives at most 2 events per second per
  stream: rate, counters, `smpSynch`, quality summary and computed values.
- Values are computed in native code from the decoded samples:
  - for 9-2LE style datasets (`INT32` + quality, current 1 mA and voltage 10 mV per
    count): RMS and the fundamental phasor (magnitude, angle) per channel over whole
    cycles; a cycle with a lost sample is dropped, never mixed into the next one;
  - for any other dataset layout: raw values only, labelled "not interpreted".
- A maximum of 16 streams is supervised. Beyond that, the extra streams are listed and
  marked as not supervised, never silently ignored.

**Supervision per expected stream**
- States: `waiting`, `receiving`, `problem` and `timeout`. Timeout means no frame for
  1 s.
- Problems:
  - `conf_rev_mismatch`
  - `dataset_mismatch`: the data size per sample differs from the recognised layout
  - `rate_mismatch`: measured samples per second against `smpRate`/`smpMod`
  - `not_synchronised`: `smpSynch` is 0
  - `sample_loss`: a `smpCnt` gap, counted and shown as a total and a rate
  - `invalid_quality`: any channel with invalid quality
  - `simulation`: the simulation bit is set (meaning and handling are in [Test mode](test-mode.md))
- Streams seen on the wire but not in the project are listed as *unknown*, with svID,
  APPID, MAC, source MAC, VLAN and the simulate bit.

**Interface** (Tools → GOOSE & Sampled Values traffic, beside GOOSE traffic; moved from Settings by ADR-0054)
- A stream table with state, rate, synchronisation, loss and quality. Every state has
  text, an icon and a fix hint.
- In the stream detail, a per-channel table shows RMS and phasor, with units and the
  nominal frequency used.
- The nominal frequency is a visible project setting:
  - one value, 50 Hz or 60 Hz;
  - the default is 50 Hz, shown with the hint "Change to 60 Hz for 60 Hz systems";
  - no frequency is ever guessed silently.
- Missing Npcap or no adapter: the same guidance as GOOSE.

## Out of scope

- Publishing SV, or merging-unit simulation.
- Waveform recording and COMTRADE export.
- PTP/IEEE 1588 accuracy analysis.
- R-SV (routed SV).
- IEC 61869-9 specific profiles beyond decoding.

## Rules

| ID | Rule | Enforced by |
| --- | --- | --- |
| SV-01 | No packaged executable contains SV publishing code; only the test-only `studio_sv_test_publisher` compiles the publisher. | `studio/tests/unit/test_native_build_config.py::test_sv_is_subscribe_only` |
| SV-02 | Expected streams come only from `SampledValueControl` + `SMV` address; a missing address is an import warning, not a guess. | `studio/tests/unit/test_sv_scl.py::test_control_without_address_is_diagnosed` |
| SV-03 | Python receives at most 2 events per second per stream, whatever the frame rate. | `studio/tests/protocol/test_sv_listen.py::test_event_rate_is_decimated` |
| SV-04 | `smpCnt` gaps are counted as sample loss, including across counter wrap. | `studio/tests/protocol/test_sv_listen.py::test_sample_loss_counts_gaps_and_wrap` |
| SV-05 | Measured rate is compared with `smpRate` and `smpMod`; a mismatch is `rate_mismatch`. | `studio/tests/unit/test_sv_supervision.py::test_rate_mismatch` |
| SV-06 | `smpSynch` 0 is shown as `not_synchronised`. | `studio/tests/unit/test_sv_supervision.py::test_unsynchronised_stream` |
| SV-07 | RMS and phasor are computed only for recognised 9-2LE layouts; others show raw values as "not interpreted". | `studio/tests/unit/test_sv_values.py::test_unknown_layout_is_not_interpreted` |
| SV-08 | The nominal frequency is an explicit, visible setting used for every computation. | `studio/tests/unit/test_sv_values.py::test_nominal_frequency_is_explicit` |
| SV-09 | Streams over the supervision limit are listed as not supervised. | `studio/tests/unit/test_sv_supervision.py::test_stream_limit_is_reported` |
| SV-10 | Every SV state and problem has a plain label, an icon and a fix hint. | `studio/tests/frontend/svStatus.test.ts::every sv state has text icon and hint` |
| SV-11 | A stream from an independent publisher process is received and supervised on the same adapter. | `studio/tests/protocol/test_sv_listen.py::test_independent_publisher_is_received` |
| SV-12 | The simulate bit and streams not in the project are seen by the passive header reader. | `studio/tests/protocol/test_sv_listen.py::test_simulate_bit_and_unknown_stream_are_seen` |
| SV-13 | Changing the network mode keeps the nominal frequency, and changing the frequency keeps the mode. | `studio/tests/unit/test_sv_values.py::test_frequency_survives_mode_changes` |

## Acceptance evidence

Recorded in change [0061](../changes/2026/0061-sampled-values.md), on the VMware lab
adapter with the test-only publisher (`studio_sv_test_publisher`, never packaged):
- 1 A / 63.5 V at 50 Hz, 80 samples per cycle, read back as 1.0000 A and 63.5003 V,
  with phase angles of 0°, −120° and +120°.
- About 4,000 samples/s measured. One sample dropped every 400 or 500 is counted
  exactly as loss, across the `smpCnt` wrap.
- The simulate bit (raw test frame) is detected, and a second stream not in the project
  is listed as unknown.
- 4 streams (16,000 frames/s) use 15.3% of one CPU core in `studio_live`, with no loss.
- Real merging-unit evidence is still required before any interoperability claim.

## Answered questions

1. A 2 Hz per-channel table is the scope; a live waveform view needs a recording design
   and stays out of scope. (Owner delegated the proposed answers, 2026-10-14.)
