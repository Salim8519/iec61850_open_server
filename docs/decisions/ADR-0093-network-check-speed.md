# ADR-0093: Faster Network setup checks with visible progress

Accepted: 2026-10-21. Owner: Network setup is slow, the IP check and file opening take time, and
the small spinner makes users think the app hangs. Choices the owner made are marked (owner).

## Decision

- **Firewall rules are read from Windows' own rule store.** Network setup and the gateway card
  asked PowerShell (`Get-NetFirewallRule -Name 'Prefix*'`) for ScadaBench's rule names: about
  5 s per call on the owner's PC, twice per check. The Go engine now reads the value names of
  the local rule store (`HKLM\SYSTEM\CurrentControlSet\Services\SharedAccess\Parameters\
  FirewallPolicy\FirewallRules`, one value per rule, named by the rule's Name), read-only, in
  about 6 ms, with the same case-insensitive prefix match. If the store cannot be read, the
  PowerShell query runs as before, so the answer and the errors never change (owner: "no errors
  please"). Only name listings move; blocking-rule and clean-up checks stay on PowerShell.
- **Up to 64 addresses are checked at once** (16 before). Each address still waits the full time
  Windows gives an unanswered ARP request; a shorter wait was offered and not chosen (owner).
- **Live progress (owner).** `GET /network/plan/progress` (Go only) reports the running check:
  four steps with their state and time, how many addresses were asked and have answered, and
  each row's address as `checking`, `free`, `in_use` or `not_checked`. It only reports; the
  preview, its rows, plan and fingerprint are unchanged. Network setup polls it every 300 ms
  through the shared live feed while its first check runs.
- **One display for long waits (owner):** `StepList` (steps ticked with their time, the running
  step turning with an "N of M" bar, waiting steps as circles, time so far, "Still working,
  nothing is stuck." after 4 s; words and icons, colour only repeats them). Used by Network
  setup (cards and address check) and, in the middle of the window, by opening a project and
  reading or importing an SCL file (shown after 400 ms so quick tasks do not flash).
- **The file's address stays in view (owner):** every IED row shows the address its CID/SCD
  gives, with "Use file address" whenever another address is in the field.
- **No new cache (owner: no caching preferred).** IED addresses were already stored in the
  project at import (`scl_address`); Network setup never re-reads files.

## Compatibility

- Python engine unchanged (frozen): it has no progress route; the frontend then shows the four
  steps with the first one running, as before the first answer.
- Results of the firewall reading are equal to the PowerShell query (live test on Windows).
- Find IEDs keeps its existing progress bar and live results.
