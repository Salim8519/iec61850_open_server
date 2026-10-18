# Known issues and delivery status

Implementation is in progress. The existing Docker demo is separate from Studio.
Native reads, enhanced controls, GI and live data-change reports have local
independent-client evidence. This is not third-party-client certification.

Outstanding release gates:

- Browser visual/workflow acceptance.
- Report-first mapping, saved Signal Test controls, bulk deletion and display-only
  switch motion are implemented and the updated portable build is now serving the
  live workspace. Visual acceptance remains pending. See change 0017 in the index.
- Additional SCL schema revisions, attribute arrays, per-index instance overrides,
  substation topology import/export, multi-source SCD merging, indexed/setting-group
  export, additional bit/octet encodings and explicit simulation-address export.
- Native CFG array correction is promoted to the normal worker and portable build.
  Top-level SCL DO `count` is not valid under the bundled XSD; schema-valid SCL
  array checks use nested SDO arrays. Array-of-array behavior and per-index
  SCL overrides remain separate open work.
- Interlocking is advisory only (switching hints, ADR-0043); Studio does not simulate
  CILO/EnaOpn/EnaCls or block commands. Physical switching-delay/failure behavior is open.
- Typed DPC/SPC loopback behavior is independently probed. Third-party relay/client
  interoperability, vendor-specific control checks and trip-reset workflow visual
  acceptance remain open. Enhanced-SBO selection, repeated-selection rejection,
  matching operation and selection lifecycle events have local independent-probe
  evidence. UI operations drive explicitly mapped simulated Dbpos, Boolean or
  constructed-contact feedback after command acceptance.
- Every client gets standard IEC 61850 control; the CONCEN profile was removed
  (ADR-0069, change 0099) after a complete capture showed the GE concentrator
  sending standard `SBOw`+`Oper`. GE UR FlexLogic is not simulated: HMIs such as
  GE Viewer gate each command on PROCEED virtual outputs (`Master/GGIO1.IndNNN`,
  e.g. CB close Ind008), which the project must drive with latching push buttons,
  and push-button latches reset when the IEDs restart.
- Direct signal assignment, configurable Trip polarity, Push Button modes and
  analog Source properties are implemented. External MMS control follows the same
  mapped feedback path as Studio operation. Format-7 Trip relays store independent
  command and state polarity; migration defaults are Operate=0 / Reset=1 and
  Operated=1 / Reset=0. Keyed live rendering and reconnect hydration preserve
  authoritative canvas state. Tabs now share a visible backend session/revision,
  synchronize project mutations and reject stale diagram writes. Browser visual
  acceptance remains open.
- Mapping suggestions, quality editor and broader quality/time coverage.
  Explicit report/dataset filtering and saved independent signal testing are now
  implemented; automatic mapping suggestions remain separate work.
- Guided LAN setup (change 0049) is implemented and tested with fakes; applying it
  on a real adapter with the UAC prompt still needs one recorded owner run.
- Portable clean-machine packaging, tray lifecycle and packaged recovery workflow
  acceptance. Source startup recovery and cross-process locking have tests.
- Concurrent 20-IED/50,000-attribute load and report recovery benchmarks.
- Broader malformed-input and concurrent lifecycle coverage.

Loopback listeners are the default; lab mode adds adapter addresses (0049). Original SCL download is not an edited-model
export. A separate staged scalar export now validates and re-imports its output;
it retains all IED declarations in the original document and leaves the project
unchanged. Its browser workflow is not visually verified yet.
Timed set/check sequences persist and run; unknown scenario actions are rejected.
GOOSE publish/listen (0050) and read-only live acquisition (0051) have evidence on
this PC only: a VMware virtual adapter and a Studio stand-in device. Physical
network, third-party relays/subscribers, Sampled Values, R-GOOSE, commands to real
IEDs and electrical solving remain open. Npcap must be installed by the user.
Station-level SLD import, signal flow, station health, Sampled Values listening and
test mode are built (changes 0058-0062) with stand-in and VMware-adapter evidence only;
real vendor SCDs with CSWI LNodes, third-party relays and merging units are pending.
Studio cannot set the GOOSE simulation bit on simulated IEDs (TMODE-06, retired).
UI v2 (change 0054) awaits owner visual acceptance; checked at 1366×768 and 1920×1080.
Licensing (0069): no protected EXE/installer built yet (dry run and unit tests only); PayPal
and Mailgun tested against fakes, not live sandbox; release key/URL not configured; offline
PCs never learn about refunds or moves; unsigned builds trigger SmartScreen. The stale
2026-09-16 portable folder fails `test_portable.py` (no studio_live.exe) until rebuilt.
Desktop window (0073): windowed EXE, second-launch focus and the close prompt are not yet
verified in a built EXE. Without Edge/Chrome the default-browser tab has no exit control.
The close prompt uses the browser's own "Leave site?" wording.
