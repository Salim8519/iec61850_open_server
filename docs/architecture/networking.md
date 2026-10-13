# Network lifecycle

Default MMS endpoints: 127.50.0.2:102 onward, one stable endpoint per IED.
The browser administration API binds 127.0.0.1 on a free port.
Local endpoint preflight checks duplicate and unavailable bindings.

LAN setup is a separate explicit workflow: adapter selection, address table,
conflict checks, change preview, then an elevated helper.
Journal aliases and narrow firewall rules created by the app. Remove only those
changes at shutdown or recovery; never replace primary addresses or existing rules.

Do not expose the administrative API on LAN. Use session authentication, origin
checks, and explicit script tokens. Loopback availability alone is not authentication.
