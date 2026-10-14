# ADR-0039: Test mode, behaviour and simulation

Accepted: 2026-10-14. Specification: [Test mode](../specs/test-mode.md).

IEC 61850 Ed2 testing relies on Mod/Beh per logical device, on `LPHD.Sim`, and on
simulation-flagged GOOSE/SV. Engineers need to see that state across the station, and to
put Studio's simulated IEDs into the same modes as real relays during a test.

**Decision**
- **Real IEDs:** `LLN0.Mod/Beh` and `LPHD.Sim` are read through the existing read-only
  live session (ADR-0037) and shown as text badges on device rows, in health rows and on
  bound diagram equipment. Studio never writes them; the backend refuses such writes.
- **Simulated IEDs:**
  - Each logical device gets an on / on-blocked / test / test-blocked / off choice, plus
    `LPHD.Sim`.
  - `Beh` is derived from `Mod` per IEC 61850-7-4.
  - The settings are saved as an optional per-IED `test_mode` (no format change). They
    are written at Run, and immediately when changed while running, through the normal
    sample writer.
- **Simulation notice:** simulation-flagged GOOSE and SV are listed in Station health as
  "Simulation traffic present", with each source.
- **Retired: "Mark GOOSE as simulation" (TMODE-06).** libiec61850 1.6.1 has no API to set
  the simulation bit on server-published GOOSE, and vendor sources stay unmodified
  (ADR-0006).
- **Native fix:** the worker now writes integers with the setter of the declared type.
  `MmsValue_setInt64`/`setUint32` silently ignore attributes narrower than their width,
  so INT8/16/32, Enum and INT8U/16U values were never written. Out-of-range values are now
  refused.

**Compatibility**
- Additive for project data.
- The integer fix changes behaviour: writes that silently did nothing before now take
  effect, or fail visibly when out of range.
- Native programs must be rebuilt.
