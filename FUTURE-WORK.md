# Future work (owner-approved, not started)

Found on 2026-10-23/24 by the vendor-file test round: 181 public and local SCL files (ABB, Siemens,
GE, SEL, Schneider, NR/NARI, Hitachi, Efacec and others) pushed through import, and 24 vendor IEDs
run and checked with an independent MMS client. The owner chose to fix the import and runtime bugs
now (changes 0185 and 0186) and to keep the items below as planned future changes. Raw evidence and
reproductions are in the session scratchpad folders `bulk-static` and `runtime-test` (local only;
some files may be customer projects and must never be published).

## 1. Memory use far over budget (important)

**Problem.** The Go engine's budgets (`docs/architecture/go-engine/README.md`) are about 80 MB for
6 IEDs / 90k points and 200 MB for 20 IEDs x 36k points. Measured on master e496dd5e:

| Data | Measured | Budget |
|---|---|---|
| 4 IEDs, 82k points, fresh server | 510-534 MB | ~80 MB |
| 9 IEDs, 176k points, after import / after browsing / idle after reopen | 618 MB / 1.06 GB / 749 MB | ~200 MB |
| 19 IEDs, 256k points (30 x SEL station) | 970 MB / 1.37 GB / 1.16 GB | ~200 MB |
| 20 IEDs, ~340k points, right after Run / steady / after Stop | 2.4 GB / 1.0-1.73 GB / 1.73 GB | 200 MB |

That is about 3-9 KB per point, worse than the 2.8 KB per point recorded for the old Python
engine, and memory is not given back when the run stops. Cold open of a 13 MB SCL project took
3.07 s and of the 19-IED copy 3.6 s (budget 3 s). A 20-IED Run takes 2.0-2.9 s (budget 1-2 s).

**Likely causes (to confirm with a heap profile first).** Every signal is held as an ordered map
(`studio/engine/internal/pyjson/object.go`) instead of a compact struct; uncapped global caches
(`scl/data_classes.go`, `scl/controls_extended_read.go`) keep whole parsed SCL documents; run-time
state may keep per-run copies after Stop.

**Plan.** Profile (heap and allocations) on the 20-IED vendor project; compact signal storage behind
the existing API (answers must stay byte-identical); bounded, size-aware caches; release run state on
Stop; add a memory benchmark gate (import, browse, Run, Stop) to `verify.py` so it cannot regress.
Needs its own ADR, because it touches the core signal model.

## 2. Attribute arrays (`DA count`) are refused

**Problem.** 11 IEDs (Schneider PM8000/ION meters, WAGO DSCH, BINOM, Schneider Easergy P5) are
refused with "needs array-attribute support" (`studio/engine/internal/scl/builder_attr.go`),
so the whole IED cannot be used even though only a few attributes are arrays (harmonics, histogram
values, multi-value settings).

**Plan.** Model arrays in the Go model builder and the native worker configuration (libiec61850
supports array attributes), expose each element as its own signal (`ref(0)`, `ref(1)` ...), keep
reports, datasets and GOOSE members that point into arrays correct, and prove it with an independent
MMS client reading the array. Needs an ADR (model and protocol change). An interim option, if wanted
earlier: import the IED and leave only the array attributes out, with a warning naming each one.

## 3. IEDs with several access points (NR / NARI layout)

**Problem.** NR and NARI relays (for example PCS-921S, PCS-9705, NSR-3641; also the owner's
FBMR-E02) put MMS on access point S1, GOOSE on G1 and Sampled Values on M1 by design. Import
refuses any IED with more than one server ("select an access point first"), but no screen lets the
user choose (`scl/importer.go`, `scl/preview.go`).

**Plan.** Simulate the MMS server of S1, keep the GOOSE and SV control blocks and addresses from G1
and M1 (they are the same IED), and say clearly in the preview which access point supplies what.
Needs an ADR and protocol evidence that GOOSE published from the G1 definitions is identical.

## Also noted (smaller, not scheduled)

- No way to simulate an interlock or synchro-check refusal on SPC/DPC breakers and disconnectors
  (`runtime/typed_checks.go` refuses them; CILO.EnaOpn/EnaCls is not used).
- Deadbands (`db`, `zeroDb`) are not modelled: a 10 Hz source sends 10 reports per second.
- Commands ignore Beh and the Test flag (IEC 61850-7-4 Annex A). Kept on purpose by the owner on
  2026-10-24 (ADR-0104); revisit only if the owner asks.
