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

**Quick tests (2026-10-24, master 12094ec2, 20-IED vendor project, `studio_server.exe`).** Private
memory of the server; the 20 native IED programs add about 0.5 GB (25 MB each) while running.

| Setting | After open | Running | After Stop | Run start | Steady CPU |
|---|---|---|---|---|---|
| Today (Go default, GOGC=100) | 1.0 GB | 2.2-2.4 GB | 2.3 GB | 3.3-3.9 s | under 1 % of a core |
| GOGC=50 | 0.81-0.88 GB | 1.1-2.0 GB | same as running | 3.5-3.7 s | same |
| GOGC=25 | 0.75 GB | 0.98 GB | 0.98 GB | 4.3 s | about 8 % of a core (costs CPU) |
| GOMEMLIMIT=512MiB | 0.6 GB | 0.8 GB | 0.8 GB | 8.1 s | about 10 cores: never use a limit below the live data |
| Give memory back after Stop (`debug.FreeOSMemory`) | 1.0 GB | 2.2 GB | 0.9 GB | 3.3 s | same |
| GOGC=50 + give back after Stop | 0.81 GB | 1.1 GB | 0.85 GB | 3.6 s | same |

The GC trace shows the real (live) data: about 0.5 GB after open (the point storage), a short peak of
1.35 GB live while Run starts, and 0.67 GB after Stop. So Stop already frees the run; the memory is
only not returned to Windows. Quick wins with no behaviour change: GOGC=50 set inside the engine (same
in development and the package) and give memory back after Stop. Then find what Run start builds all at
once (the 1.35 GB peak), then compact point storage. Owner decision: the website keeps "4 GB RAM".

**Done in change 0189 (ADR-0116, 2026-10-24):** compact objects, shared enum lists, one SCL read per
file at Run start, lookups limited to what a request names, memory returned after one-off jobs. Same
20-IED project: after open 1,028 -> 370 MB, peak 1,595 -> 761 MB, after Stop 1,595 -> 557 MB, server CPU
-65 %, reports and GOOSE per second unchanged. Still open: a typed point store (about 1 KB per point
today), the native IED programs (about 25 MB each) and a memory gate in `verify.py`.

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

**SCADA view (checked 2026-10-24).** In the 12 vendor files with arrays (harmonics `MHAI.HA/HPhV/HPPV`,
schedules `DSCH.SchdAbsTm`, curves `crvPts`), no report or GOOSE dataset uses an array point, except
one demo file (`PDIF0.TmASt.crvPts`, FC DC). A SCADA that imports the SCD needs the IED (breakers,
measurements, reports), not the arrays. So the interim option is enough for SCADA testing; an IED whose
dataset uses an array point should stay refused with a clear reason, because the SCADA expects the
exact dataset.

## 3. IEDs with several access points (NR / NARI layout): done in change 0190

Built 2026-10-24 ([ADR-0117](docs/decisions/ADR-0117-station-access-point.md)): S1, the access point
with the IP address or the reports, is simulated exactly; G1 (GOOSE) and M1 (SV) are named in the
preview and not simulated.

## Also noted (smaller)

- Deadbands (`db`, `zeroDb`): done in change 0190 ([ADR-0118](docs/decisions/ADR-0118-scd-deadbands.md)), off
  by default with a per-IED "Use SCD deadbands" switch; SCADA writes follow the SCD valKind.
- Interlock or synchro-check refusal simulation (CILO.EnaOpn/EnaCls): **not part of ScadaBench**
  (owner, 2026-10-24: not in the next 10 versions at least). Do not plan or propose it.
- Commands ignore Beh and the Test flag (IEC 61850-7-4 Annex A). Kept on purpose by the owner on
  2026-10-24 (ADR-0104); revisit only if the owner asks.
