# ADR-0123: The SCADA master chooses the network it connects from

Accepted: 2026-10-27. Owner: "when user wants to connect to local gateways, our software acts as a
client ... how he will connect and knows our machine? something is weird"; "windows picks the ip,
that's the weird, we should add ip selection too"; "space is tight ... for network, by default is
all network ... until one picks, or specific ... keeping selection minimal". Change
[0196](../changes/2026/0196-master-network.md).

## Problem

As IEC 104 controlling station, DNP3 master or Modbus TCP client, ScadaBench opened its connection
from whatever address Windows chose. Gateways and outstations usually accept only known master
addresses, but the app never said which address of this PC the device sees, and on a PC with several
cards (Wi-Fi, VMware, Hyper-V, VPN) the user could not choose it.

## Decision

1. **One field, "Network", under Device address** in the master's connection column. Default
   **All networks**: Windows chooses by its routes, exactly as before. Otherwise one address of this
   PC; the closed field shows only the address (never clipped), the list groups addresses by card.
2. **Go builds the list**: IPv4 addresses of connected cards, without link-local addresses, without
   IED addresses (SkipAsSource aliases, the project's IED addresses, everything on the ScadaBench
   Lab card) and without disconnected cards. Only a listed address can be saved.
3. **Go says what the device sees**: "The device sees this PC as A (card)". For All networks this is
   Windows' routing answer (a UDP connect that sends nothing). When the chosen address is not the one
   Windows would use, a warning names the card that reaches the device; a failed connect repeats it.
4. **studio_master binds the connection**: `settings.local` → lib60870
   `CS104_Connection_setLocalAddress`, the opendnp3 TCP client adapter, and `bind` in the Modbus
   client (which reports "This PC cannot reach … from …" at once).
5. **Stored as `local` in the master block** (Go engine only, beside `point_list`); empty means All
   networks. The frozen Python engine does not read it. Routes `GET/PUT /tools/master/network`.

## Compatibility

Existing projects have no `local` and keep All networks, so connections behave as before. The
station gateway role is unchanged. Adding an address the PC does not have ("Use my own address")
is not part of this decision.
