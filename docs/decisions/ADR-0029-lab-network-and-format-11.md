# ADR-0029: Guided lab network, elevated helper and project format 11

Accepted: 2026-10-14. Owner request: bind simulated IEDs to a real network adapter
(StationScout-comparable item 3) without surprising changes to Windows.
Specification: [lab network](../specs/lab-network.md).

- Format 11 adds `project.network = {mode, adapter_guid, adapter_label}` and per IED
  `role`, `live`, `lab_ip`, `goose_publish`. Older projects migrate to loopback
  mode, role `simulated`, publishing on; a verified recovery archive is written
  first, as for every migration. Format 9 archives are now accepted (they were
  wrongly rejected by the compact-format check).
- `lab_ip` is separate from the loopback `ip`, so switching back to "This PC only"
  needs no address rewrite. SCL addresses are never rewritten.
- The wizard previews an exact change list with a fingerprint. Apply re-plans and
  refuses a changed plan (409). Only the elevated helper changes Windows: it runs
  the same validation code in development (`networking/helper/run.py`) and in the
  frozen build (`SubstationStudio.exe --network-helper`), checks the request hash
  and an allow-list, and then uses `New-NetIPAddress -SkipAsSource -PolicyStore
  ActiveStore` plus one inbound TCP rule in group `Substation Studio` scoped to the
  owned addresses, the MMS ports and `studio_ied.exe`.
- An append-only `data/network/journal.jsonl` records applied, removed and failed
  items. Clean-up removes only outstanding owned items; leftovers found later are
  reported, never removed silently.
- The admin API now also rejects non-loopback remote addresses, not only foreign
  `Host` headers, because lab mode puts LAN addresses on the PC.
- Real-device addresses may be loopback (another simulator on this PC); lab
  addresses may not.

Compatibility: older Studio builds cannot open format-11 projects. Waitress still
binds 127.0.0.1. Lab firewall rules persist until clean-up; aliases vanish on reboot.
