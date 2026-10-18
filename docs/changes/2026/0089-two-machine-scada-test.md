# 0089: Two-machine SCADA test, third-party interoperability, 20 fixes and the gateway firewall

Date: 2026-10-18. Decision: [ADR-0062](../../decisions/ADR-0062-gateway-firewall.md).
Specs: [SCADA master](../../specs/scada-master.md), [Station gateway](../../specs/gateway.md) (GW-11).

Owner request: test ScadaBench between two machines (host and a VMware Windows VM on VMnet8),
against third-party IEC 104, DNP3 and Modbus tools, in every role. Fix every bug at its root,
with a test.

## Setup

- **Host:** source app on 127.0.0.1:18766 with a temporary data folder. Host address on VMnet8:
  192.168.98.1.
- **VM:** 192.168.98.130, isolated (no internet). Portable 0.1.7, built with
  `compile_protected` and licensed by the owner.
- **Third-party tools on the VM:**
  - FreyrSCADA IEC 104 Server and Client Simulators 21.06.022;
  - FreyrSCADA DNP3 Outstation and Client-Master Simulators 21.06.022;
  - proconX diagslave 3.5 and modpoll 3.16.
- Only the sample IED (Feeder01) and simulator test points were used, with no customer data.

## Interoperability observed (only what was seen)

| Direction | Protocol | Result |
| --- | --- | --- |
| ScadaBench master (host) → diagslave (VM) | Modbus TCP | Find registers, reads, FC6 write confirmed by modpoll, live changes, link drop and auto-reconnect |
| ScadaBench master (host) → FreyrSCADA server (VM) | IEC 104 | GI, counter interrogation, C_DC direct, select refusal worded, unknown IOA, link drop and reconnect with re-GI |
| ScadaBench master (host) → FreyrSCADA outstation (VM) | DNP3 | Integrity poll incl. counter and frozen counter, CROB, analog output, refusals worded, link drop and reconnect |
| FreyrSCADA 104 client (VM) → ScadaBench gateway (host) → Feeder01 (MMS) | IEC 104 | GI with 11 points, C_CS, C_DC direct and SBO, spontaneous M_DP_TB_1, unknown IOA refused (COT 47) |
| FreyrSCADA DNP3 master (VM) → ScadaBench outstation (host) | DNP3 | Integrity poll, CROB closes the breaker through MMS |
| modpoll (VM) → ScadaBench Modbus server (host) | Modbus TCP | FC2, FC4 and float reads, FC15 coil write opens the breaker, exception 2 where nothing is mapped |
| FreyrSCADA 104 client (VM) → ScadaBench converter (host) → FreyrSCADA DNP3 outstation (VM) | 104 → DNP3 | 7 points relayed with the right values; C_SC becomes a CROB latch, accepted |
| ScadaBench master (host) ↔ ScadaBench 0.1.7 gateway (VM) | 104, DNP3, Modbus | GI and SBO C_DC; integrity poll, SBO CROB and unsolicited DBI; Find registers, reads and coil write |
| ScadaBench 0.1.7 master (VM) → ScadaBench gateway (host) | 104, DNP3, Modbus | Values invalid while Run is off, good after; SBO C_DC closes the breaker; CROB opens it with an unsolicited DBI; Modbus Find registers, reads and coil write |
| Host IED Explorer and Find IEDs → VM simulated IED on the VM lab network | MMS | Browse, reads, polling, operate (direct-with-enhanced-security); scan of 254 addresses in 2.3 s finds and identifies it |
| IEDExplorer 0.80a (VM) → ScadaBench simulated IED (192.168.98.21) | MMS | Identify, model, variables, named variable lists, XCBR1 values including the host's operate |
| Independent 104 client → host gateway → VM IED as "Real device" | 104 → MMS | Switch off: negative confirmation, nothing sent; switch on: confirmed, terminated, closed position reported back |
| IEC-Test 01.04 (OST) → ScadaBench 0.1.7 gateway, on the same VM (127.0.0.1:2404) | IEC 104 | STARTDT/TESTFR; GI answered with ACT_CON, M_SP/M_DP/M_ME_NC/M_ME_NB (COT 20) and ACT_TERM; C_DC select-before-operate close and open on IOA 50002 confirmed and terminated, with spontaneous M_DP_TB_1 on IOA 20001 (2, then 1); wrong common address answered COT 46 and an unsupported TI 111 answered COT 44, both negative |
| Siemens IEC 61850 Browser 5.1.20 → ScadaBench 0.1.7 simulated IED, on the same VM (192.168.98.21:102) | MMS | Connect and browse (two GetNameList requests returned error 0x6a17 and the scan continued); read XCBR1.Pos (stVal, q, t) and GGIO1.AnIn1.mag.f; buffered reporting on LLN0$Events delivered a dataChange report for XCBR1.Pos.stVal; its "Control Value" stayed disabled without an SCL file, so no control was sent from it |

## Bugs found and fixed (each has a test)

1. **Packaging.** `compile_protected.py` failed on an Inno Setup file lock caused by the antivirus.
   It now retries ISCC up to 8 times (`test_protected_packaging.py`).
2. **Modbus wording and form.**
   - The interval hint talked about minutes and "all values".
   - "Select before operate" was offered for Modbus.
   - A write answered "Done: write: written".

   Modbus now has its own hint, has no select, and a write says "the device confirmed the write"
   (`scadaTerms.test.ts`).
3. **Link lost.** Rows stayed "good" while the link was down. They now read
   "not current (link lost)" (`test_master_rules.py`).
4. **Interrogation wording.** A failed Modbus read was called "Interrogation failed". The message
   now uses each protocol's word.
5. **Stale rows after switching protocol.**
   - Rows from the old device stayed in the table. Points now belong to one device
     (protocol, host, port) (`test_master_device_table.py`).
   - A command draft survived a protocol switch. It is now dropped when the protocol changes.
6. **IEC 104 counters never read.** The 104 master never sent C_CI, so integrated totals were
   missing. It now runs counter interrogation after the GI ends
   (`test_master_counters.py`, against an independent raw-socket station).
7. **Negative select worded as "activation confirmed".** It now says it was refused and
   suggests unticking Select before operate.
8. **DNP3 frozen counters were dropped silently**, and the DNP3 "Counter" was called
   "Integrated total". Both are fixed.
9. **DNP3 refusal wording.** "Refused: not supported (success)" now names the step, and says
   "no such control point" where that applies.
10. **The gateway on "All network adapters" was unreachable** because of Windows Firewall Block
    rules (major). Fixed by ADR-0062 with a firewall card and network-helper items
    (`test_gateway_firewall.py`).
11. **Converter notes.**
    - IEC 104 address 0 was described as "(0 was taken)"; it now says "(IEC 104 has no address 0)".
    - Converter notes were prefixed with "Skipped:"; the prefix is gone.
12. **DNP3 state bits read as quality.** DNP3 binary and double-bit state bits were read as
    "reference error", which made the converter mark open breakers invalid. Quality words are
    now per DNP3 object type.
13. **C_CI in the traffic log.** C_CI_NA_1 and the time-tagged commands showed as "other" in the
    traffic log; they are now named.
14. **DNP3 confirmation wording.** The confirmation said "address 1" for a DNP3 output; it now
    says "index 1" (`scadaTerms.test.ts`).
15. **Find registers estimate.** The page promised "up to about 5052" requests in seconds; a real
    sweep took about 5,300 requests and 9 minutes. The estimate now follows the worker's split
    rule and states minutes (`probeCost.test.ts`).
16. **Coil writes.** A coil value was a free number box; it is now ON or OFF.
17. **Host hint.** "Accept this PC as its master" appeared for Modbus too; it is now neutral.
18. **Clean-up card.** It said "from an earlier session" right after this session's apply
    (`guideCopy.test.ts`).
19. **Simulated IED identity.** It reported the old name "Substation Studio / Laboratory
    simulator" to every client; it is now "ScadaBench / Simulated IED" (`test_explorer_wire.py`).
20. **Real-IED refusal wording.** The gateway log said real IEDs are "never" commanded, but the
    owner's switch decides. It now says the switch is off (`test_gateway_wire.py`).

## Evidence

- `python studio/tools/verify.py`: PASS.
- Tests:
  - 380 unit and integration tests pass;
  - protocol: 64 pass, 7 skipped (optional third-party tools);
  - frontend: 119 pass;
  - `tsc` is clean.
- Detailed two-machine notes (T1–T10) were taken during the run. The table above lists only
  what was observed.

## Limitations and open items

- **Host follow-up, done 2026-10-18 with the owner's UAC approvals:**
  - **Gateway firewall clean-up:** both ScadaBench-Gateway rules removed. Windows' own rule for
    the source-build gateway is enabled again. Windows had meanwhile rewritten it from Block to
    Allow, because the owner answered its "allow access?" dialog. A first attempt was refused
    ("administrator permission was not given"); nothing changed, as designed.
  - **Host lab network on VMnet8:** Feeder01 on 192.168.98.21:102.
    - From the VM, ScadaBench 0.1.7 Find IEDs found and identified the host's IED in 2.3 s as
      "ScadaBench / Simulated IED".
    - IED Explorer browsed it, read XCBR1 and operated Close, and the host read it back as closed.
    - IEDExplorer 0.80a read the model and named variable lists ("Init end").
    - Clean-up restored VMnet8 to 192.168.98.1 only, with no ScadaBench firewall rules left.
  - **Fixes found during this follow-up, each with a test:**
    - Clean-up refused to re-enable Windows' rule once its action was no longer Block. Restore
      now only checks that the rule is still the gateway's inbound rule
      (`test_gateway_firewall.py`).
    - The clean-up list took 10.6 s to appear (one PowerShell per item) and showed nothing
      meanwhile. Windows Firewall is now read once per preview (about 5 s), with a "Checking
      Windows…" line.
    - The card now says "Will be switched on again" for Windows' rule.
    - While the UAC prompt is open, the card says it is waiting for Windows, and the button
      cannot be pressed twice (`guideCopy.test.ts`).
    - After a command, IED Explorer showed "now reads on (closed)" above a table that still said
      "off (open)". It now reads the data object again (`guideCopy.test.ts`).
  - **Incident:** while driving the VM console, a mis-mapped click restarted the test VM. Nothing
    was lost (Run was stopped, the project autosaves), and the owner signed in again.
- **Not yet tested:** real IEDs (no physical relay was available).
- **VM 0.1.7 is older than these fixes.** Its own screens still show bugs 3, 5, 12, 16 and 19
  (they were seen there, and fixed in source). A new portable build is needed to carry the fixes.
- **Modbus Find registers on a sparse map is slow by design.** It probes at most 10 requests
  per second, so 0–5010 on the gateway took several minutes. Servers that accept partly mapped
  blocks give ranges wider than the mapped registers.
- **Protocol details:**
  - IEC 104 time tags are UTC;
  - DNP3 binary output status stays RESTART until the first command;
  - the DNP3 command log names the peer "master" (opendnp3 does not expose the address).
