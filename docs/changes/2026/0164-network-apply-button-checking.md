# 0164: Network setup: the apply button shows a spinner while the addresses are checked

Date: 2026-10-22. Owner request after [0163](0163-typed-addresses-checked.md). Frontend only.

While a typed address or a new network card is being checked, the apply button stays unusable but no
longer looks greyed out: it keeps its colour and reads "⟳ Checking the addresses…" (`aria-busy`), then
turns back into "Put N IEDs … on <card>" when the check is done. No spinner before a card is chosen or
after a failed check.

Evidence: frontend 373/373; tsc clean; Playwright 1366×768 on a private copy (preview only, nothing
applied to Windows): button read "Checking the addresses…" 60 ms after typing and "Put 6 IEDs and 1 SNMP
device on “VMware Network Adapter VMnet8”" after the check.
