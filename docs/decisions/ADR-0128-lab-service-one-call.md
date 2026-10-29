# ADR-0128: The lab service changes all addresses in one call, without the DNS Client

Accepted: 2026-10-30. Owner: "when we open the app first time, we click run, it takes too much
time to start ... i used lab network", "it happened now also while clicking stop", "my laptop under
huge pressure ... why network has huge usage" (Task Manager: Service Host: DNS Client), then, of
the proposed fixes, "ScadaBench only" and "make sure this doesnt effect user experience or slow
experience, make sure the app is instant, yet doesnt eat cpu, ram, network, windows overall".
Changes [ADR-0120](ADR-0120-this-pc-lab-network.md) §1 and §3 and
[ADR-0127](ADR-0127-lab-addresses-follow-the-run.md) §6. Change
[0202](../changes/2026/0202-lab-service-one-call.md).

## Problem

ADR-0127 §6 assumed each lab address change takes milliseconds. Measured on the owner's PC
(17 network cards, 7 lab IEDs), each took 0.85–2.35 s, one after another: Run spent about 9–11 s
and Stop about 11 s of its 13 s on the ScadaBench Lab card. Each request made the service read
every network card three times (card check, address list, the add itself), and each
`GetAdaptersAddresses` call without `GAA_FLAG_SKIP_DNS_INFO` asks the Windows DNS Client about
every card (about 35 DNS Client operations per listing here; 12 ms calm, 135–800 ms while the DNS
Client was busy). The card also had "Register this connection's addresses in DNS" on, so the DNS
Client worked on every address change of a card that never uses DNS. Every Stop also tried to
remove the card's own link-local address (169.254.x), which the service refuses, and journaled a
failure.

## Decision

1. **One card listing per request, no DNS Client.** The service reads Windows once per request
   (the card, every card's IPv4 addresses and names) with `GAA_FLAG_SKIP_DNS_INFO`, and adds or
   removes through the card's index and LUID from that same listing. Nothing it reads needs DNS
   information.
2. **Several addresses in one request (protocol 2).** `add` and `remove` take either `ip` (one, as
   before) or `ips` (up to 256). Each address gets the same checks as a single request; the reply
   keeps `ok`, `message` and the card's `addresses`, and adds `failed` {address: reason} for the
   ones Windows or the checks refused. Requests may be 8 KiB. The service reports version `"2"`.
3. **The app sends one request when it can.** Run's adds and Release's removals go in one request
   when the service's last answer said version 2 or later; an older service still gets one
   request per address, exactly as before. The journal keeps one `lab_address` record per address.
4. **Never the card's own addresses.** Release removes only addresses the service could have added
   (`ParseAddress` accepts them); 169.254.x, which Windows gives the card itself, is left alone.
5. **No DNS registration on the card.** Create and Repair turn off "Register this connection's
   addresses in DNS" on the ScadaBench Lab card, with the other card settings (metric, DHCP, IPv6).
6. **Repair brings the faster service.** App updates do not replace the installed service (only
   the elevated install does). When the service answers with version 1, the lab page says so in
   one line with the existing Repair button (one Windows prompt; disabled while the simulation
   runs). The version is remembered from the service's answers at launch, Run and Stop; showing
   the page adds no pipe call.

Nothing new runs in the background and nothing polls: the app makes fewer calls than before
(two per Run or Stop instead of one plus one per address).

## Compatibility

Pipe requests of protocol 1 keep their meaning and answers; an app of 0.1.29 or older works with
the new service. A new app with an old service behaves as 0.1.29 (one request per address) and
shows the Repair line. No project data or HTTP API body changes; `GET /network/lab` gains
`service_update`. The journal stays append-only with the same record shapes.
