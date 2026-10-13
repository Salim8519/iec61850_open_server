# 0038: Color-coded property sections per domain

Date: 2026-09-20. Every equipment's properties now read in domain
colors: violet for IED signals, blue for command, teal for live
operate, amber for legacy objects. A flat `PropertySection` (tinted
divider, dotted title, domain badge) wraps state mapping, relay
signals, measurement generation, command mapping and live operation
for each kind; inner duplicate titles were removed. Full references
stay visible and wrapped. No logic, contract or runtime change.

Evidence:

- `npm run check` passes; 9 frontend node tests pass; backend unit
  suite: 106 pass; `verify.py` passes.

Limitations: visual acceptance pending.
