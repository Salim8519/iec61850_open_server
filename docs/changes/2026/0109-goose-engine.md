# 0109: GOOSE engine in Go, GOOSE inside the app by default, one switch to share

Date: 2026-10-19. Decision: [ADR-0073](../../decisions/ADR-0073-goose-engine-go.md).
Spec: [GOOSE](../../specs/goose.md) GOOSE-05, 11, 17–27; [signal layers](../../specs/signal-layers.md) LAYER-21.

## Owner request

"Why doesn't GOOSE work on loopback?" Then: zero setup. GOOSE stays inside the software by default.
One click shares it with the outside network, with a high-risk warning when the IEDs use real
addresses. Smart protection is always on. A conflict asks the user to stop the IED or continue.
The engine is written in Go ("easiest to debug… ram low, cpu fast, builds fast").

## Change

- **Go engine** `studio/goose_engine` → `studio_goose.exe` (standard library, no cgo, 2.6 MB).
  - Packages: `frame` (parse), `guard` (gates, conflicts, rate, outbound checks), `bus` (loopback
    UDP), `uplink` + `npcap` (one Npcap handle, loaded from `System32\Npcap` only), `engine`, `ipc`.
  - Build: `studio/tools/build_goose_engine.py`. The dev launcher rebuilds it when stale,
    `verify.py` runs `go test`, and `studio.spec` packages it.
- **Native.**
  - `studio/native/hal`: dispatcher (`udp:` → engine, numeric → vendor Npcap HAL renamed
    `Pcap_*`). The vendor tree is untouched.
  - `studio_ied` publishes only to `udp:` (`goose_bus` capability).
  - `studio_live` decodes from a UDP inbox (`goose_inbox`), adds `origin`/`src_mac`, and never
    starts an adapter receiver.
- **Backend.**
  - `acquisition/goose_engine.py` and `goose_share.py`, `domain/goose_reach.py`,
    `projects/goose_reach.py`, `api/goose_share.py`.
  - Network keys `goose_reach`, `goose_uplink_guid`, `goose_share_guid`, kept on a mode change.
  - The supervisor gets each IED's bus before launch.
  - One-time notice for lab projects saved before this change.
  - Npcap wording: needed only to share or to hear real devices.
- **Frontend.**
  - *Where GOOSE goes* card in Settings › Network & GOOSE: inside/share radios, adapter line,
    Advanced picker.
  - Share warning (alertdialog, Cancel focused, stronger for high risk).
  - Conflict prompt bottom-right: *Keep it stopped outside* / *Continue anyway* → *Yes, keep
    sending*.
  - Top-bar share icon while running.
  - Traffic status pill with `from the network` rows.
  - Removed every "Needs lab network / works on the lab network only" GOOSE wording (SLD sheet,
    Devices, hints).
  - Tools subtitle shortened so it fits at 1366 px.
- **Fixed during the browser check.** Run with sharing froze ("Native uplink request timed
  out"):
  - Cause: `GooseEngine` held its state lock while waiting for the engine's reply, and the reader
    thread needed the same lock to deliver the `goose_share` event sent before that reply.
  - Fix: requests and state now use separate locks.
  - Regression test: `test_events_arriving_before_a_reply_never_deadlock`. It fails on the old
    single-lock design and passes on the new one.

## Evidence

| Check | Result |
| --- | --- |
| Scratch feasibility (before the plan) | loopback UDP bus 3000/3000, p50 0.4–0.9 ms; libiec61850 over Npcap 500/500; bridge tests 300/300, zero loops; `NOCAPTURE_LOCAL` hides only its own injections |
| `go test ./...` (frame, guard, limits, ipc, engine, 7 engine tests) | pass; `go vet` clean |
| Engine smoke on VMnet1 with real Npcap | sharing after 3.1 s; conflict caught; own frames never heard back |
| `pytest studio/tests/unit studio/tests/integration` | 458 passed |
| `pytest studio/tests/protocol` with `STUDIO_GOOSE_ADAPTER` = VMnet1 | 88 passed, including `test_goose_inside` (no adapter), inputs/consistency/runtime through the engine, and `test_goose_share_wire` + `test_goose_conflict_wire` against the independent `npcap_peer` (3 consecutive runs stable) |
| 10-IED latency benchmark (`test_goose_engine_latency`) | 3000/3000 delivered; p50 0.074 ms, p99 0.146 ms, max 0.6 ms (target p99 ≤3 ms) |
| `node --test studio/tests/frontend` / `tsc --noEmit` | 174 pass / clean |
| Dev browser, 1366×768, owner's dev project (6 IEDs, *This PC only*) | Inside: all 6 GOOSE blocks *Receiving* with `goose_input` events, no Npcap or adapter used. Share on VMnet1 (Advanced): warning names the adapter; *Shares at Run* while stopped; Run → *Checking* ~3 s → *Sharing*; top bar and subtitles unclipped. Peer injected C06_Y130's gocbRef from 00-30-A7-10-20-30 → prompt shown; *Continue anyway* asks again; *Keep it stopped outside* → choice `stop`, others keep sharing. Project network restored to inside/automatic afterwards |

## Limitations

- Sharing was proven on a VMware host-only adapter with an independent Npcap peer on the same PC,
  not through a physical switch, a host↔VM pair or a third-party relay. A host↔VM retest (as in
  0103) is still due.
- No Wireshark capture was recorded; the independent peer stands in for it.
- Switch port security, Wi-Fi drivers that drop foreign source MACs, and a second ScadaBench
  running the same project (reported as a conflict) are listed in known issues.
- Go is needed to build from source. No EXE was built (owner rule).
