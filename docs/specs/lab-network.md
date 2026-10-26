# Spec: Lab network

Decision: [ADR-0029](../decisions/ADR-0029-lab-network-and-format-11.md).
Owner policy (2026-10-13): simulated IEDs may be reached from a real adapter only
through an explicit, previewed, reversible workflow.

## Purpose

By default every simulated IED listens on a private `127.50.0.x` loopback address
that only this PC can reach. Lab mode lets a real device or another PC reach the
simulated IEDs by adding app-owned alias addresses to one chosen adapter.

## Behaviour contract

- Modes: `loopback` (default) and `lab`. Lab mode stores one adapter GUID.
- Adapter changes are made only by the elevated network helper after the user has
  reviewed an exact change preview with a fingerprint.
- Alias addresses use `SkipAsSource` and the non-persistent `ActiveStore`, so they
  disappear on reboot. One inbound TCP/102 firewall rule in group
  `Substation Studio` allows only the owned aliases and only `studio_ied.exe`.
- Anything that already existed is reported as *already present* and is never
  journaled as owned or removed.
- The journal is append-only; cleanup removes only what the journal says Studio
  added and has not yet removed.
- The browser administration API stays on `127.0.0.1` in every mode.
- The setup is one page (ADR-0068): it uses the SCD address of each IED set to *Be this IED*;
  a row that cannot be used gets one free address on the card's own network to accept.

## Out of scope

- Changing an adapter's primary address, DHCP, gateway, DNS or VLAN.
- Persisting aliases across reboot.
- IPv6.

## Rules

| ID | Rule | Enforced by |
| --- | --- | --- |
| NET-01 | Projects migrated from format 10 open in loopback mode with every IED simulated and no adapter. | `studio/tests/unit/test_format_11_migration.py::test_format_10_opens_in_loopback_mode` |
| NET-02 | The administration API rejects non-loopback callers in every network mode. | `studio/tests/integration/test_network_api.py::test_admin_api_rejects_remote_callers` |
| NET-03 | Loopback mode rejects LAN endpoints; lab mode accepts a LAN endpoint only when that address exists on the selected adapter. | `studio/tests/unit/test_endpoints.py::test_loopback_mode_rejects_lan_address`, `studio/tests/unit/test_endpoints.py::test_lab_address_must_exist_on_adapter` |
| NET-04 | The address plan never silently proposes the adapter's own address, an address of another adapter or a duplicate; adapter-subnet planning stays inside that subnet; every conflict carries a plain fix hint. | `studio/tests/unit/test_address_plan.py::test_conflicts_are_detected_with_hints`, `studio/tests/unit/test_address_plan.py::test_scd_address_preferred_inside_subnet` |
| NET-05 | Applying a plan requires `confirm: true` and the preview fingerprint; a changed plan is refused. | `studio/tests/integration/test_network_api.py::test_apply_requires_matching_fingerprint` |
| NET-06 | Alias commands use `SkipAsSource` and `ActiveStore`; the firewall rule is scoped to the owned aliases, TCP 102 and the worker program. | `studio/tests/unit/test_change_plan.py::test_alias_is_non_persistent_and_skip_as_source`, `studio/tests/unit/test_change_plan.py::test_firewall_rule_is_narrow` |
| NET-07 | Addresses or rules that already exist are marked not owned and are never removed by cleanup. | `studio/tests/unit/test_change_plan.py::test_existing_address_is_not_owned`, `studio/tests/unit/test_network_journal.py::test_cleanup_removes_only_owned_items` |
| NET-08 | The journal is append-only; recording a removal appends and never rewrites earlier lines. | `studio/tests/unit/test_network_journal.py::test_journal_is_append_only` |
| NET-09 | The elevated helper refuses a plan whose hash does not match and any item outside its allow-list. | `studio/tests/unit/test_network_helper.py::test_rejects_tampered_plan`, `studio/tests/unit/test_network_helper.py::test_rejects_items_outside_allow_list` |
| NET-10 | Runtime endpoint overrides never change the stored SCL source. | `studio/tests/integration/test_network_api.py::test_endpoint_override_keeps_scl` |
| NET-11 | Leftover owned changes found at startup are reported, never silently removed. | `studio/tests/unit/test_network_journal.py::test_leftovers_are_reported_not_removed` |
| NET-12 | Adapters are read through the Windows IP Helper API (PowerShell only as a logged fallback), match what `Get-NetAdapter` shows, list Ethernet ports first, and are fast enough for the adapter step to refresh by itself every 3 s. | `studio/tests/protocol/test_adapter_listing.py::test_native_list_matches_get_netadapter`, `studio/tests/unit/test_ip_helper.py::test_native_records_are_normalised_with_ethernet_first`, `studio/tests/unit/test_ip_helper.py::test_powershell_is_only_a_fallback` |
| NET-13 | With no IED set to *Be this IED* the planner names the mode to choose instead of a bare error. | `studio/tests/integration/test_network_setup_page.py::test_nothing_to_publish_says_which_mode_to_choose` |
| NET-14 | A taken or missing SCD address offers exactly one free address on the card's network; a free SCD address offers none. | `studio/tests/integration/test_network_setup_page.py::test_unusable_scd_address_offers_one_free_address`, `studio/tests/integration/test_network_setup_page.py::test_free_scd_address_offers_no_alternative` |
| NET-15 | The ScadaBench Lab card (ADR-0120) holds each simulated IED's own address as a /32 that can be a source address (ping and clients without a bind work); the service accepts only status, add and remove of one IPv4 unicast address on the card its own registry key names, refuses loopback, link-local, multicast, 0.x, broadcast and another card's address, and repeating a request is harmless. | `studio/engine/internal/labnet/handler_test.go::TestAddPutsAnAddressOnTheCardOnly`, `::TestAddressOfAnotherCardIsRefused`, `::TestUnusableAddressesAreRefused`, `::TestAddAndRemoveAreSafeToRepeat`, `::TestOnlyKnownRequestsDecode` |
| NET-16 | Duplicate, another card's own, invalid and missing addresses block Use and Change; an address in a real card's network is asked by ARP from that card's own address only while the card is connected; a device that answers is a warning with a free address of the same /24, never a block. | `studio/engine/internal/labsetup/rows_test.go::TestBlockingClashes`, `::TestRealNetworkRanges`, `studio/tests/frontend/labNetwork.test.ts` |
| NET-17 | Use puts every simulated IED on the card and switches the project only while stopped and only without clashes; Change moves one IED at any time, stopping and starting it when it runs; Run puts missing card addresses back first. | `studio/engine/internal/labsetup/service_test.go::TestUsePutsEveryIEDOnTheCardAndSwitchesTheProject`, `::TestUseRefusesAClashAndChangesNothing`, `::TestARunningIEDMovesAndStartsAgain`, `::TestRunPutsTheAddressesBackAfterARestart` |

## Evidence still required

Real adapter application with a UAC prompt is recorded in change records; it is not
claimed here until such a record exists. The ScadaBench Lab card (NET-15..17) was created, used,
repaired and removed on the owner's PC in change 0192.
