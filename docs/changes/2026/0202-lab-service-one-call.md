# 0202: Faster Run and Stop, no DNS Client load from the lab network

Owner: "when we open the app first time, we click run, it takes too much time to start ... i used
lab network", "it happened now also while clicking stop", "my laptop under huge pressure ... why
network has huge usage" (Service Host: DNS Client), "ScadaBench only", "make sure the app is
instant, yet doesnt eat cpu, ram, network, windows overall", "achieve insane speeds using power of
GOlang", "Deploy the new version 0.1.30 once u eliminate all issues in cpu load and windows
resources high usage". Decisions: [ADR-0128](../../decisions/ADR-0128-lab-service-one-call.md),
[ADR-0129](../../decisions/ADR-0129-run-start-without-waiting.md).

## Diagnosis (owner's PC, 17 network cards, 7 lab IEDs, installed 0.1.29)

- Lab journal: each address added 1.4–2.3 s and removed 0.8–2.4 s apart, one after another
  (Run 9–12 s, Stop 10–11 s of 11.5–13 s). Each request made the service list every card three
  times with DNS information; such a listing took 135–800 ms (median 385 ms during the adds)
  instead of 12 ms, against 62 ms without DNS information at the same moments.
- The DNS Client (Dnscache) took about 1 core while `baisabox.exe` (another project, out of
  scope by the owner's choice) listed the cards 4 times a second, and 9–12 of 16 cores, with the
  whole CPU at 100 % and an empty DNS cache, after lab Run/Stop cycles; it calmed down by itself
  about 30 minutes later. The ScadaBench Lab card had "Register this connection's addresses in
  DNS" on. Every Stop also asked to remove the card's own 169.254.71.48 and journaled a failure.
- Timed step by step, Run also spent 14.6–18.9 s on 1,704 snapshot reads sent one at a time,
  5.0–10.9 s in `modePoints` (sorting every reference and two regular expressions on each, per
  call) and 2.3 s of CPU rebuilding a GOOSE subscriber's signal map per subscription.

## Change

- `labnet`: one adapter listing per request (`Ops.Read` → `View`), `GAA_FLAG_SKIP_DNS_INFO`;
  `add`/`remove` take `ips` (≤ 256) with per-address `failed`; requests up to 8 KiB; protocol 2.
  Create/Repair run `Set-DnsClient -RegisterThisConnectionsAddress $false` on the card.
- `labsetup`: `batch.go` sends Run's adds and Release's removals in one request to a protocol 2
  service (one per address otherwise), remembers the version from answers, journals each address
  as before; Release skips addresses `ParseAddress` refuses; `service_update` on the page.
- `netcfg`: listings skip DNS information too (network pages refresh every 3 s).
- `runtime`: `snapshot_reads.go` overlaps the initial snapshot reads (all IEDs in parallel, 64 per
  worker) and uses the values in the old order; mode points cached per model with a suffix filter.
- `goosedom`: `Binder` keeps each subscriber's signal map for one preparation.
- Frontend: one line with the existing Repair button while the service is protocol 1.
- Spec: NET-15 and NET-17 extended, NET-18 added.

## Evidence

- Go: `labnet`, `labsetup`, `app`, `runtime/...`, `acquisition/...`, `netcfg` pass (`modes` and
  `netcfg` built with `go test -c` into another folder: the antivirus refused the temp test exe,
  "Access is denied"; both pass). Live tests on this PC: the listing without DNS information
  equals the full listing for the lab service and for netcfg (link speed left out). Frontend
  types clean; lab frontend test 4/4.
- New handler with the real Windows reading (add/remove recorded, they need administrator
  rights): 7 addresses in one request 0.15–0.28 s; the same 7 one by one 0.85–1.24 s; the old
  service took about 10–12 s for them during a Run.
- New app with the old installed service, lab mode: one request per address as before, no
  169.254 removal, `service_update: true`.
- Timing build, lab mode: snapshot 18.9 s → 0.17 s; initial states 0.16 s.
- "This PC only", same project, same laptop, two rounds each: installed 0.1.29 Run 5.65 / 7.08 s,
  Stop 0.32 / 0.20 s; new build Run 3.48 / 3.50 s, Stop 0.11 / 0.12 s (about 2.7 s of it is the
  seven native workers starting in parallel).
- Release 0.1.30 (owner: "Deploy the new version 0.1.30 ... Fix, test, deploy"): installer
  322016ce, portable 35c55ecf; portable self-test ok; packaged app, This PC only, same project:
  Run 4.79 s (cold) / 3.64 s, Stop 0.09 / 0.11 s, run log application 0.1.30. Live
  `/download/ScadaBench-Setup.exe` and the portable link serve these exact files; signed release
  record 0.1.30 saved; changelog live. Unsigned as before. Built with `studio/.venv` Python and
  the website with Node 22.23 (pi-node).

## Limitations

- The new service reaches a PC only through Create or Repair (one UAC prompt); app updates do not
  replace it. Until then the app behaves as 0.1.29 for the lab card and shows the Repair line.
  The owner could not approve the prompt, so the protocol 2 service and the DNS registration
  setting were not run elevated on the real card; the Windows add/remove calls are unchanged.
- That turning DNS registration off stops the DNS Client storms is inferred from the timeline
  (storms after lab cycles on a registering card), not proven with a DNS Client trace.
- `baisabox.exe` still lists every card 4 times a second (owner chose ScadaBench only).
- The VM host-key helper still lists cards with DNS information every 10 s (about 3 calls/s).
