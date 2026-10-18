# ADR-0069: Standard IEC 61850 commands only, sent where the IED listens

Accepted: 2026-10-18. Supersedes [ADR-0018](ADR-0018-concen-sbow-compatibility.md). Change:
[0099](../changes/2026/0099-standard-commands-lab-operate.md).

**Problem**

1. ADR-0018 added `concen_inverted_sbow`: for originator `CONCEN` the simulated IED inverted
   `ctlVal` and moved the switch on `SBOw`. It was based on a capture that ended about 30 ms
   before the concentrator's `Oper` (change 0031). On 2026-10-18 complete captures from the GE
   Viewer + Concentrator showed standard enhanced SBO: Close = `SBOw(true)` then `Oper(true)`,
   Open = `false`. With the profile on, a correct Close was executed as Open (no movement on an
   open breaker) while every MMS answer said success. Equipment sharing one control also had to
   share the profile, which blocked switching one breaker back to standard.
2. In lab network mode each worker listens only on its lab address (`effective_endpoint`), but
   ScadaBench's own Operate button and the gateway dialled the IED's loopback address. The probe
   waited out its 10 s connect timeout and the UI said "MMS control failed or command termination
   was not received"; nothing reached the IED.

**Decision**

1. **One command meaning for every client.** The native worker applies `ctlVal` as given
   (DPC true = close, false = open; SPC true = set) and moves only on `Oper`; select only
   reserves. The originator identity is never inspected. The worker advertises
   `standard_controls`; the backend requires it, so an older worker that still carries the
   profile is refused as stale. A start request that still contains `compatibility` is refused.
2. **Project format 15.** `control_binding` is `{ied_id, ref}`. Loading formats 8-14 drops the
   stored profile; each CONCEN binding gets a migration notice ("commands now follow standard
   IEC 61850") and the usual verified recovery copy is written first. The Equipment Properties
   "External client behavior" choice and the shared-profile rule are removed.
3. **Commands go to the confirmed endpoint.** The supervisor records the address and port each
   worker confirmed in its start reply and exposes `endpoint(ied_id)`. Operate
   (`/equipment/{id}/operate`, `/controls/operate`) and gateway commands to simulated IEDs use
   it; real IEDs keep their own address. Failure text names the address that was tried.

**Compatibility impact**

- Projects saved by this build are format 15; older builds reject them (as for every format).
- Behaviour changes only for bindings that used the CONCEN profile; they now behave like the
  real GE BCU (standard SBO). Probe `sbo-compat*` actions are replaced by an optional
  originator argument to `sbo-*` so tests can prove any `orIdent` gets the same behaviour.
- HMIs that gate commands on vendor FlexLogic outputs (GE "PROCEED" virtual outputs) still need
  those points driven by the project; that is a separate, known limitation.
