# 0049: Guided lab network binding and project format 11

Date: 2026-10-14. Owner request: plan and build StationScout-comparable item 3,
"bind simulated IEDs to a real network adapter", beginner friendly with hints.
Decision: [ADR-0029](../../decisions/ADR-0029-lab-network-and-format-11.md).
Specification: [lab network](../../specs/lab-network.md).

Cause: simulated IEDs could only listen on 127.50.0.x; `networking/loopback.py`
rejected every other address, so real devices and other PCs could not reach them.

Change:
- Format 11 project settings (`domain/network_settings.py`, `projects/ied_settings.py`)
  with a recovery-archived migration. Format 9 archives are accepted again.
- `networking/`: adapter listing (`adapters.py`, `local_addresses.py`), address
  planning with conflict hints (`address_plan.py`, `arp_probe.py`), fingerprinted
  change plan (`change_plan.py`), append-only journal (`journal.py`), UAC launch
  (`elevation.py`), allow-listed helper (`helper/`), workflow service (`service.py`)
  and mode-aware Run preflight (`endpoints.py`).
- Routes in `api/network.py`; admin API also checks the caller's remote address.
- Network tab with a four-step wizard, endpoint table and clean-up card.

Evidence:
- Unit/integration: `test_format_11_migration`, `test_endpoints`, `test_address_plan`,
  `test_change_plan`, `test_network_journal`, `test_network_helper`,
  `test_network_api` (Windows calls replaced by fakes). Full Python suite: 208 pass.
- Browser (isolated data folder): wizard listed this PC's nine adapters with kind,
  status and internet badges; the SCD strategy proposed 10.0.0.2/24 and 10.0.0.3/24
  for the sample SCD on the VMware VMnet1 adapter. A first check took ~20 s because
  off-link addresses were ARP-probed; only on-link addresses are probed now, and
  off-link rows say they could not be checked instead of claiming "free".

Limitations: Apply was not executed on this PC during development because it
changes real Windows settings behind a UAC prompt; the owner should run it once on
a spare or virtual adapter and record the result. Firewall rules persist until
Clean up. IPv6 and VLAN configuration are out of scope.
