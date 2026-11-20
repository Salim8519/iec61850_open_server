# 0225 Trial N11: one fixed firewall rule per program (dropped)

Date: 2026-11-20. Owner: "do all of them" (speed and network trials); idea N11 of the
[speed and network trials plan](../../testing/speed-trials-plan-2026-11-20.md).
Builds on [ADR-0062](../../decisions/ADR-0062-gateway-firewall.md),
[ADR-0132](../../decisions/ADR-0132-real-card-addresses-follow-the-run.md) and
[ADR-0136](../../decisions/ADR-0136-leave-nothing-on-the-network.md) (which left this idea open).
Layers: Go engine (tests only). No product behaviour changed; no ADR written.

## Idea

Today every new set of IED addresses or MMS ports gets a new inbound rule
`SubstationStudio-MMS-<hash of addresses, ports, worker>` (`netcfg/change_plan.go`,
`BuildPlanWithSNMP`; command in `netcfg/commands.go`). N11 proposed one fixed rule per program,
so changing an IED's address 5 times on Ethernet would ask Windows fewer times while a PC on the
network can still connect.

## How prompts are counted today (code path)

Changing an address on a real card (Devices › Clients connect to, `netaddr/real.go`, or Network
setup) runs `Preview` then `Apply` with every simulated IED at its address. `Apply` starts the
elevated helper (one Windows prompt) exactly once when `ownedAdds` of the plan is not empty
(`netcfg/service_apply.go`). With the network service (protocol 3, ADR-0132) the plan holds:

- one `approval` per address not yet in the service's `Approved` key (`netcfg/approval.go`,
  `useApprovals`); an approved address is "keep";
- one `firewall` item, "keep" when a rule with the same name (same addresses, ports and worker)
  already exists. Apply never removes earlier rules or approvals; Clean up, the reopen check
  (ADR-0136) and the uninstaller (B-3) do.

So a new address always brings its own `approval` in the same request as the new rule.

## Measurement (helper-request builder, `netcfg/prompt_count_test.go`)

The bench runs the planner exactly as Preview does (BuildPlan, approvals from the service's key,
`ownedAdds`) and treats each applied item as Windows would then have it. One non-empty request is
one prompt. "Lower bound" makes every firewall item free (as if one rule already allowed
everything): no firewall design can ask fewer times than that.

| 3 IEDs on Ethernet, 5 changes after the first Apply | Today | Lower bound (firewall free) |
|---|---|---|
| A moved to 5 new addresses (the owner's N11 test) | 5 | 5 |
| A moved back and forth between two addresses | 1 | 1 |
| C's Simulate switched off and on, all approved | 1 | 0 |
| B's MMS port changed 5 times | 5 | 0 |

In the first row every prompt is one request with exactly `[approval, firewall]`: the rule rides
in the prompt the approval needs anyway. Deterministic code, so one run per scenario is the
measurement; the tests pin the counts.

## Verdict: Drop

1. **No gain in the test the plan names.** 5 prompts today, 5 at best: each new address must be
   approved by an administrator (ADR-0132: the service may only put on what an administrator
   approved). Removing that would widen what any local program can do through the service pipe.
2. **The savings that exist need a wider rule.** A rule can save the port-change prompts only
   without a port filter, and the literal "per-program" rule (worker + TCP, any local address)
   would let the IED worker accept connections on every address of every card, the owner's
   Wi-Fi address included. Today's rules allow only the planned IED addresses on the planned
   ports. Scoping to the card (`-InterfaceAlias`) is still wider (the card's own address).
3. **A narrow rule changes whenever today's does.** A rule that allows no more than today's must
   grow exactly when a plan needs an address or port no rule allows yet, which is when today's
   code adds a rule. The only prompt it could save is a new, smaller set inside an existing rule
   (row 3, at most one prompt per new set), and that is reuse of an existing rule, not N11.

Nothing was built, so Clean up, the reopen check and the uninstaller's firewall undo
(`scadabench_labnet firewall`, B-3) are unchanged.

## Found on the way (not built; for the owner)

- Rule reuse: Preview could "keep" an existing journaled `SubstationStudio-MMS-` rule that
  already allows every planned address and port for the same worker, instead of adding a rule
  for the smaller set. It saves at most one prompt per new smaller set (Simulate off, a project
  with fewer IEDs on the same addresses) and allows nothing new. Not done tonight: small gain,
  and it changes what the preview lists.
- Rules pile up: Apply leaves earlier MMS rules in place until Clean up, so after several
  address changes the PC keeps allowing the old addresses (only while the worker listens there,
  which happens only while an IED runs on them). This is today's behaviour, unchanged.

## Evidence

- `go test ./internal/netcfg/ -run TestPrompts -v`: 4 of 4 pass (counts above).
- `go test ./internal/netcfg/` passes; `go vet` and `GOOS=windows go vet` on `internal/netcfg`
  are clean.
- `python studio/tools/verify.py`: all 8 checks PASS. The first run, while other agents' engines
  and builds loaded the PC, ended with one Go engine package failing (the load-sensitive suite;
  this change adds only a pure planner test); the next run passed every check unchanged.

## Limitations

- Counted in code and unit tests, not with real Windows prompts (none could be answered
  tonight). Owner check: change one IED's address 5 times on Ethernet (Devices › Clients connect
  to, a new address each time) and count the Windows prompts: expected 5, one per change, each
  listing one "Let ScadaBench put … on 'Ethernet'" line and one firewall line.
- With the network service missing, stopped or too old, the plan adds the service set-up to the
  same prompt as the approvals (change 0216, `service_plan.go`); the count per new address is the
  same.
