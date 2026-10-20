# Built-in MIB modules

These are the standard MIB modules ScadaBench loads into every MIB library
(`mib.Standard()`, ADR-0092). Each file holds exactly one module, named after the module.
The files are imported engineering data: they are excluded from the 299-line limit in
`studio/tools/check_file_lengths.py` and must not be edited by hand.

## How the files were made

- RFC modules: downloaded from `https://www.rfc-editor.org/rfc/rfcNNNN.txt` on 2026-10-20.
  The text from `NAME DEFINITIONS ::= BEGIN` to the matching `END` was extracted, RFC page
  footers (`[Page N]`), form feeds and page headers (`RFC NNNN ... Month Year`) were removed,
  together with the blank page padding around them, and the common indentation was removed.
  Nothing else was changed.
- IANA modules: downloaded from `https://www.iana.org/assignments/...` on 2026-10-20 and
  stored as published (only surrounding text outside the module, if any, was dropped).
- `RFC-1212` and `RFC-1215` are not complete modules in their RFCs: the RFCs only publish
  the `IMPORTS` and the `OBJECT-TYPE` / `TRAP-TYPE` macro. As in other MIB distributions,
  the lines `NAME DEFINITIONS ::= BEGIN` and `END` were added around that text. In
  `RFC-1212` the import of `DisplayString FROM RFC1158-MIB` (an obsolete module that is not
  shipped, and not needed by the macro) is commented out.

## Modules and sources

| Module | Source |
| --- | --- |
| RFC1155-SMI | RFC 1155, https://www.rfc-editor.org/rfc/rfc1155.txt |
| RFC-1212 | RFC 1212 (macro only, see above), https://www.rfc-editor.org/rfc/rfc1212.txt |
| RFC1213-MIB | RFC 1213 (MIB-II), https://www.rfc-editor.org/rfc/rfc1213.txt |
| RFC-1215 | RFC 1215 (macro only, see above), https://www.rfc-editor.org/rfc/rfc1215.txt |
| UPS-MIB | RFC 1628, https://www.rfc-editor.org/rfc/rfc1628.txt |
| SYSAPPL-MIB | RFC 2287, https://www.rfc-editor.org/rfc/rfc2287.txt (needed by NTPv4-MIB) |
| SNMPv2-SMI | RFC 2578, https://www.rfc-editor.org/rfc/rfc2578.txt |
| SNMPv2-TC | RFC 2579, https://www.rfc-editor.org/rfc/rfc2579.txt |
| SNMPv2-CONF | RFC 2580, https://www.rfc-editor.org/rfc/rfc2580.txt |
| HOST-RESOURCES-MIB | RFC 2790, https://www.rfc-editor.org/rfc/rfc2790.txt |
| HOST-RESOURCES-TYPES | RFC 2790, https://www.rfc-editor.org/rfc/rfc2790.txt |
| IF-MIB | RFC 2863, https://www.rfc-editor.org/rfc/rfc2863.txt |
| SNMP-FRAMEWORK-MIB | RFC 3411, https://www.rfc-editor.org/rfc/rfc3411.txt |
| SNMP-TARGET-MIB | RFC 3413, https://www.rfc-editor.org/rfc/rfc3413.txt |
| SNMP-NOTIFICATION-MIB | RFC 3413, https://www.rfc-editor.org/rfc/rfc3413.txt |
| SNMP-PROXY-MIB | RFC 3413, https://www.rfc-editor.org/rfc/rfc3413.txt |
| SNMP-USER-BASED-SM-MIB | RFC 3414, https://www.rfc-editor.org/rfc/rfc3414.txt |
| SNMPv2-MIB | RFC 3418, https://www.rfc-editor.org/rfc/rfc3418.txt |
| ENTITY-SENSOR-MIB | RFC 3433, https://www.rfc-editor.org/rfc/rfc3433.txt |
| INET-ADDRESS-MIB | RFC 4001, https://www.rfc-editor.org/rfc/rfc4001.txt |
| TCP-MIB | RFC 4022, https://www.rfc-editor.org/rfc/rfc4022.txt |
| UDP-MIB | RFC 4113, https://www.rfc-editor.org/rfc/rfc4113.txt |
| BRIDGE-MIB | RFC 4188, https://www.rfc-editor.org/rfc/rfc4188.txt |
| IP-MIB | RFC 4293, https://www.rfc-editor.org/rfc/rfc4293.txt |
| NTPv4-MIB | RFC 5907, https://www.rfc-editor.org/rfc/rfc5907.txt |
| ENTITY-MIB | RFC 6933 (ENTITY-MIB version 4), https://www.rfc-editor.org/rfc/rfc6933.txt |
| UUID-TC-MIB | RFC 6933, https://www.rfc-editor.org/rfc/rfc6933.txt (needed by ENTITY-MIB) |
| IANAifType-MIB | IANA, https://www.iana.org/assignments/ianaiftype-mib/ianaiftype-mib (LAST-UPDATED 202603170000Z) |
| IANA-ADDRESS-FAMILY-NUMBERS-MIB | IANA, https://www.iana.org/assignments/ianaaddressfamilynumbers-mib/ianaaddressfamilynumbers-mib (LAST-UPDATED 202110190000Z) |
| IANA-ENTITY-MIB | IANA, https://www.iana.org/assignments/ianaentity-mib/ianaentity-mib (LAST-UPDATED 201507160000Z) |

Every module resolves with the others: `mib.Standard()` reports no missing imports and no
problems (checked by `TestStandardLoadsEveryBuiltinCleanly`).

## Licence

MIB modules published in RFCs are code components of IETF documents. Under the IETF Trust
Legal Provisions (TLP, https://trustee.ietf.org/license-info), code components are licensed
under the Simplified BSD License (TLP section 4), as the copyright notices inside the newer
modules state; for older RFCs the IETF Trust grants the same terms to the extent it holds
the rights. Redistribution must keep the copyright notice and licence text: the release
ships them in `studio/packaging/snmp-mibs-NOTICE.txt`.

The IANA-maintained modules are published registries of the Internet Assigned Numbers
Authority and are public; IANA-ENTITY-MIB carries the same IETF Trust BSD notice.
