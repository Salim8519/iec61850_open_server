# Network lifecycle

Default MMS endpoints: 127.50.0.2:102 onward, one stable endpoint per IED
("This PC only"). The browser administration API binds 127.0.0.1 on a free port
and rejects non-loopback callers and hosts in every mode.

Lab mode (ADR-0029, spec [lab network](../specs/lab-network.md)) is an explicit
workflow: adapter selection, address plan with conflict checks, fingerprinted change
preview, then the elevated helper. Only the helper changes Windows, and only with
allow-listed items: non-persistent `SkipAsSource` aliases on the chosen adapter and
one inbound TCP rule in group `Substation Studio` for the owned aliases, MMS ports
and `studio_ied.exe`. Existing addresses and rules are used but never owned.

`data/network/journal.jsonl` is append-only. Clean-up removes only outstanding
owned items; leftovers found later are reported to the user, never removed silently.
Run preflight binds each effective endpoint: loopback in loopback mode, otherwise
the IED's `lab_ip` which must exist on the selected adapter.

GOOSE (ADR-0030) uses the lab adapter through the user-installed Npcap driver.
Real IEDs (ADR-0031) are reached as a read-only client at their SCD address or a
stored override; they never share an endpoint with a simulated IED.

Do not expose the administrative API on LAN. Use session authentication, origin
checks, and explicit script tokens. Loopback availability alone is not authentication.
