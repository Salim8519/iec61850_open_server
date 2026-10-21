# Acceptance and evidence

Required milestones:
1. Native Windows IED: import, browse, change value, report and real MMS control.
2. Browser engineering: equipment, mappings, save/reopen and SCL round-trip.
3. Automatic/manual sources, scenarios, interlocks and multiple IEDs.
4. Portable packaging, lab LAN cleanup, load and clean-machine verification.

Benchmark: 20 IEDs, 50,000 data attributes, startup under 30 seconds on 8-core/16GB,
1,000 aggregate updates/sec for one hour, below 4GB memory, responsive UI, and no
application-caused event loss within configured report-buffer limits.

Record actual hardware, fixtures, client versions and evidence with results.
Independent MMS client checks are distinct from browser tests.
No certification, hard real-time or vendor-protection-equivalence claims.

Recorded real-machine runs:
- [VM uses the host PC's license, real VMware VM (2026-10-22)](vm-host-real-vm-2026-10-22.md),
  change [0156](../changes/2026/0156-activate-once-vm-host-key.md), follow-up
  [0157](../changes/2026/0157-vm-box-and-guide.md).

Final user-requested action: hibernate this laptop only after the complete goal
and all required acceptance checks are genuinely finished. Do not hibernate for
a partial milestone or while blocked on a check needing user input.
