# ADR-0030: Layer-2 GOOSE with user-installed Npcap

Accepted: 2026-10-14. Owner decision: Npcap is installed by the user (free licence);
Studio detects and guides, never bundles. Specification: [GOOSE](../specs/goose.md).
Partly replaced by [ADR-0073](ADR-0073-goose-engine-go.md): GOOSE now runs inside the app in every
mode without Npcap; Npcap is needed only to share GOOSE with the network or hear real devices.

- Every native build compiles libiec61850 GOOSE. `cmake/goose_options.cmake`
  requires the pinned Npcap SDK 1.15 (`tools/fetch_npcap_sdk.py`, SHA-256 checked)
  and stops otherwise; `cmake/goose_support.cmake` injects the unmodified upstream
  GOOSE and win32 Ethernet sources into the vendor targets (ADR-0006 pattern; the
  vendor tree is untouched).
- `wpcap.dll`/`Packet.dll` are delay-loaded. `common/npcap.cpp` loads them from
  `System32\Npcap` only when present and maps an adapter GUID to the pcap index the
  vendor HAL expects. Without Npcap every executable still starts.
- The worker emits `GC`/`PA` configuration only for `GSEControl` blocks with a
  matching SCD `GSE` address; anything else is an import warning. Publishing is
  enabled on a background thread because opening a pcap adapter can take seconds.
  Required capability `goose_aware` rejects stale workers.
- Vendor HAL code prints to the console; the IPC layer now writes frames to a
  private duplicate of stdout and redirects descriptor 1 to stderr.
- `studio_live` receives GOOSE (one receiver per adapter), supervises
  receiving/timeout/conf-rev/dataset/commissioning problems and lists unknown
  GOOSE on the wire. Simulated subscribers get only LGOS supervision writes.

Compatibility: GOOSE is active only in lab mode with an adapter and Npcap.
Loopback projects behave exactly as before. Sampled Values and R-GOOSE remain off.
