# 0146: SNMP devices: simulate and read SNMP v1/v2c/v3 devices in Tools

Date: 2026-10-20. Owner request and decisions in [ADR-0092](../../decisions/ADR-0092-snmp-devices.md).

## Change

- New Go worker `studio_snmp.exe` (`studio/engine/cmd/studio_snmp`, stdlib only): SNMP v1, v2c
  and v3 (USM: MD5, SHA-1, SHA-224/256/384/512; DES, AES-128, AES-192/256 in both key styles,
  Blumenthal and Reeder/Cisco), agent (GET, GETNEXT, GETBULK, SET, traps and informs) and
  manager (get, set, walk, 1 s re-read, trap/inform receiver incl. v3).
- Tools > **SNMP devices** with two roles:
  - **Simulate devices**: network switch (8 ports, IF-MIB) and GPS time server (NTPv4-MIB)
    profiles, empty device, import from snmpwalk output or ScadaBench CSV. Each device has its own
    IP and port, v1/v2c/v3 users, editable values (fixed, counts up time, counts up), events that
    send traps (port link down/up, GPS sync lost/back), request log, CSV export.
  - **Read devices**: connect with v1/v2c/v3, walk all, show values with MIB names and meanings,
    re-read every 1 s (changeable), write a value, save what was read as a simulated device,
    receive traps, MIB library (30 built-in IETF/IANA modules; user can add and remove MIBs and
    bring removed built-ins back).
- Port clash explained in plain words with a suggested free port; no adapter or firewall change.
- Project section `snmp` (devices, reader, users incl. passwords as the owner chose): Go engine
  only, kept on restore, not part of the station snapshot, saving it does not bump the revision.
- Routes `/api/v1/tools/snmp/...` (Go only; see [API](../../reference/API.md)); push topics for
  status, requests, traps and values; the Tools card and Activity show what SNMP is doing.
- Release: `compile_go_release.py` builds `studio_snmp.exe` and ships it with the workers, plus
  `licenses/snmp-mibs/snmp-mibs-NOTICE.txt`. `run_dev_engine.py` rebuilds it when stale.
- Independent check `studio/tools/snmp_interop.py` (pysnmp, own venv; not in verify.py).

## Evidence

- Go tests: `snmp/proto` (BER, USM keys incl. RFC 3414 A.3 vectors, encrypt/decrypt for every
  protocol), `snmp/manager` (agent and manager round trip v1/v2c/v3, traps and informs),
  `snmp/profiles` (every profile OID resolves in the MIB library with the right type),
  `snmp/mib`, `snmp/snmpfile`, `snmp/service` (end to end with the real exe), and the whole
  engine suite (`app` route tests exclude only the Go-only SNMP routes).
- Independent client, pysnmp 7.1.21: **65/65** (pysnmp manager reads/walks/sets the ScadaBench
  agent in v1, v2c and v3 with every auth x priv pair; v1/v2c/v3 traps and informs reach a pysnmp
  receiver; the ScadaBench manager walks and sets a pysnmp agent in v2c and v3 AES-256 both styles).
- Frontend 250/250 (new `snmpTool.test.ts`), `tsc` clean.
- Browser at 1366x768 on a scratch Go server (loopback 127.66.40.1:20161, receiver 20162):
  added a switch, started it, read it in v3 (168 values, names and meanings), sent "Port 3 link
  down" and received `IF-MIB::linkDown` with `ifOperStatus.3 = 2 · down`; request log, MIB
  library; no panel scrolls sideways (side panes fixed with a `minmax(0, 1fr)` column).
- `verify.py`: file lengths, architecture, documentation, history, specifications and GOOSE
  pass; the Go step was stopped by antivirus refusing to start `gateway.test.exe` (Access is
  denied) on two runs (`settings.test.exe` on another); both packages pass when compiled and run
  separately.

## Limitations

- Python engine: no SNMP (routes answer 404; the section is ignored there).
- Simulated devices use 127.x.x.x addresses or addresses the PC already has; lab-network
  aliases for SNMP devices are not created yet.
- v3 users all have full read/write (no VACM views); a device answers one context (empty).
- Not in the SLD yet (owner chose Tools now, SLD later). No SNMP over TCP or TLS/DTLS.
- MIB notice wording follows the IETF Trust licence; it has not had a legal review.
