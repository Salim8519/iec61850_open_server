# Proposed spec: Find IEDs

> **Proposed.** Planned behaviour, not a capability. Agreed with the owner on 2026-10-17
> ([ADR-0054](../../decisions/ADR-0054-tools-place.md)).

## Purpose

Answer "what IEC 61850 devices are on this network?" in one press, with vendor, model and
firmware, so a beginner knows where to connect.

## Behaviour contract

- **Choose and confirm.** The user picks an adapter, and Studio proposes that adapter's own
  subnet.
  - Only /24 or smaller ranges are allowed.
  - A preview states the exact range and number of addresses. Nothing is sent before the
    user confirms.
- **Probe.** The backend opens TCP port 102 on each address with a bounded number of parallel
  attempts and a short timeout, using pure sockets (`tools/ied_scan.py`).
  - Progress is shown, and the scan can be stopped.
- **Identify.** For each open port, a new `studio_live` operation `identify` returns:
  - the MMS identity (vendor, model, revision);
  - the logical device names.

  Devices that refuse are listed as "port open, not identified" with the reason.
- **Results.** A table shows the address, vendor, model, revision, logical devices and the
  response time.
  - "Open in Explorer" opens the device in IED Explorer.
  - A hint explains how to add the device to the project as a real IED.
- **Safety.** No writes, no other ports and no broadcast storms. Only the confirmed range is
  scanned.

## Reserved rule IDs

| Prefix | Rules |
| --- | --- |
| `FIND` | preview before scan; /24 limit; bounded parallelism and timeout; stop works; identify fields or reason; open in Explorer |
