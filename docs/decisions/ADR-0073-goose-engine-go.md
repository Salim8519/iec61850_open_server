# ADR-0073: GOOSE engine in Go, GOOSE inside the app by default

Accepted: 2026-10-19. Change: [0109](../changes/2026/0109-goose-engine.md). Spec: [GOOSE](../specs/goose.md)
(GOOSE-05, 11, 17–27). Partly replaces [ADR-0030](ADR-0030-goose-with-user-installed-npcap.md)
(GOOSE only in lab mode, Npcap needed for any GOOSE) and the lab-only wording of
[ADR-0066](ADR-0066-goose-inputs-and-mms-access-events.md).

**Problem**

GOOSE is a layer-2 Ethernet message (EtherType 0x88B8) and has no IP, so the private
`127.50.x.x` addresses of *This PC only* mode cannot carry it. Until now:
- GOOSE ran only in lab mode, through an adapter, with Npcap installed.
- In lab mode every simulated GOOSE frame also went out to the real network, with no warning and
  no way to keep it on the PC.

The owner asked for zero setup:
- Simulated IEDs exchange GOOSE inside the app by default.
- One switch shares it with the outside network, after a clear warning (high risk when the IEDs
  use real addresses on that adapter).
- Smart protection is always on while sharing.

**Decision**

1. **A second process, written in Go.** `studio_goose.exe` (source: `studio/goose_engine`, Go 1.24+,
   standard library only, no cgo) owns every wire decision: where a frame goes, conflicts, sharing,
   rate limits. Go was chosen because the owner develops with AI help and wants code that is easy
   to read and debug, low on memory, quick to build and fast. Go is already used in the repository
   (`license-server/`). It talks to the backend over the same 4-byte length + JSON channel as the
   native workers (`WorkerProcess` unchanged). `hello` returns capability `goose_engine_v1`; the
   backend refuses a missing or older binary.
2. **Workers never get an adapter.** A small Ethernet HAL in `studio/native/hal` routes interface
   ids that start with `udp:` to a send-only loopback backend toward the engine (`udp:PORT:MAC12`).
   Numeric ids still use the unmodified vendor Npcap HAL, compiled under `Pcap_*` names, so
   Sampled Values listening is unchanged. The vendor tree stays untouched (ADR-0006).
3. **Engine → decoder.** The engine forwards valid frames to a UDP inbox in `studio_live`, with a
   1-byte origin (`app` / `wire`). `studio_live` keeps decoding with libiec61850
   (`GooseReceiver_handleMessage`, no receiver thread) and adds `origin` and `src_mac` to `goose_rx`.
   Each simulated IED gets its own locally administered MAC per run, `02:<random 2 bytes>:<index>`.
4. **Inside (default)** works in every network mode, with no Npcap and no adapter. When Npcap is
   present the engine also listens on the auto-picked adapter (one handle: promiscuous,
   `NOCAPTURE_LOCAL`) so real devices are heard. It cannot transmit unless sharing is on.
5. **Share** needs `confirm: true` and the adapter GUID the user saw (`PUT /goose/share`).
   - The adapter is auto-picked: the lab adapter in lab mode, otherwise a connected wired adapter
     preferring the default route. Wi-Fi, VPN, virtual and loopback adapters are never auto-picked.
     Advanced can choose one.
   - Sharing pauses (it never follows) when the resolved adapter changes.
   - Before sending, the engine listens 3 s.
   - Only registered simulated frames go out: multicast, ≤1518 bytes, ≤1000 per second per IED.
6. **Smart guard, always on.**
   - Frames that come back with our MAC are dropped (reflections).
   - Network frames are never sent back to the network.
   - If a real device sends a simulated IED's gocbRef, that IED's outward copy is *held*, the
     real copy is never delivered inside, and the user chooses *Keep it stopped outside* or
     *Continue anyway* (confirmed). The choice lasts for the Run.
7. **Settings** live in the project network block, with no format bump (ADR-0038):
   `goose_reach` (`inside` | `outside`), `goose_uplink_guid` (Advanced) and `goose_share_guid` (the
   confirmed adapter). A mode change keeps them. Lab projects saved before this change open with a
   one-time notice: sharing is never switched on automatically.
8. **Build.**
   - `studio/tools/build_goose_engine.py` tests and builds the engine into the native folder.
   - The development launcher rebuilds it when a Go source is newer, so dev never runs a stale
     engine.
   - The PyInstaller spec ships it next to `studio_live.exe`.
   - `verify.py` runs `go test`.

**Compatibility**

- New runtime process and language. The cost is about 10 MB of RAM and one extra loopback hop.
  The measured latency for 10 IEDs is p99 0.15 ms.
- Go is needed to build from source, but not to run the app.
- Lab users lose automatic outward GOOSE; the one-time notice explains how to switch it back on.
- The `studio_ied` capability `goose_bus` and the `studio_live` capability `goose_inbox` are now
  required, so older native binaries are refused.
- Switch port security may limit how many source MACs one port accepts (known issue).
