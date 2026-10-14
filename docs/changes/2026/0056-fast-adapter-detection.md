# 0056: Fast, live adapter detection

Date: 2026-10-14. Owner report: detecting Ethernet ports/adapters in the lab wizard's
"Choose adapter" step "is not working"; on retry it worked but "takes time".

Cause: the list came from PowerShell (`Get-NetAdapter`, then `Get-NetIPAddress` per
adapter and `Get-NetRoute`). It took about 6 s warm on this PC, and a cold PowerShell start
can approach the 20 s timeout. It only refreshed on button press, so a newly plugged cable
was not seen.

Change:
- New `networking/ip_helper.py` reads adapters with the Windows IP Helper API through
  ctypes:
  - `GetAdaptersAddresses` for adapters, MAC, speed, status and gateway;
  - `GetUnicastIpAddressTable` for prefix, origin and SkipAsSource.
  It replaces `local_addresses.py` (the supervisor imports `addresses_on` from it).
- `networking/adapters.py`:
  - uses it, with PowerShell kept only as a logged fallback;
  - lists Ethernet ports first, then Wi-Fi, then virtual adapters.
- Lab wizard:
  - the adapter step silently refreshes every 3 s;
  - an unplugged port reads "No cable" with the hint "Plug the device or switch into
    this port. This list updates by itself."
- No API or data change, so no decision record. Spec rule NET-12.

Evidence (this PC):
- Native listing takes 12–16 ms, against about 6 s with PowerShell.
- It returns the same 9 adapters (GUID, ifIndex, name, MAC, speed, gateway, IPv4 addresses
  and prefixes) as `Get-NetAdapter`.
- Unplugged ports report OperStatus Down, which is shown as Disconnected, as PowerShell
  does.
- Tests: `test_ip_helper.py` and `test_adapter_listing.py` (Windows comparison, < 0.5 s).

Limitations: adapters disabled in Windows are not returned by IP Helper (Get-NetAdapter
shows them as Disabled); they cannot carry lab traffic anyway. Live "No cable" →
"Connected" switching was not observed with a physical cable in this session.
