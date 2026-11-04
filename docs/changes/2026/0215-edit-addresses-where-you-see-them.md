# 0215 Edit IED and SNMP addresses where you see them

Date: 2026-11-04. Owner: "add snmp ability to add ips like ieds, user can edit", "in the devices in
the ieds let user edit the ips if he wants no need to go to the settings and then networks
anymore". Builds on [ADR-0120](../../decisions/ADR-0120-this-pc-lab-network.md), [ADR-0132](../../decisions/ADR-0132-real-card-addresses-follow-the-run.md)
and [ADR-0136](../../decisions/ADR-0136-leave-nothing-on-the-network.md).
Layers: Go engine (`netaddr`, `api/addressapi`, `labsetup`, `snmp/service`, `app`), frontend.

## Change

- **One address change in Go.** The new package `netaddr` serves
  `PUT /api/v1/network/address/ied/{id}` and `.../snmp/{id}` with `{"ip"}` (Go only). The
  project's network decides how:
  - **ScadaBench Lab card:** an IED goes through the lab page's `SetAddress`, with the same checks;
    it is restarted if running and no Windows prompt appears. An SNMP device goes through
    `labsetup.CheckOwn` (new, the gateway's own-address checks without adding) plus a duplicate
    check against the other devices, then `SetLabIPs`.
  - **Real card (Ethernet…):** the address plan is reviewed with every current address kept and
    this one changed, then applied (one Windows prompt, an approval). The prefix comes from the
    card's own network. This is refused while running, and the plan's own hint is passed on when
    the address is not usable.
  - **This PC only:** a plain answer: put the project on a network first.
- **Devices page:** "Clients connect to" shows the network address as editable text (✎, Enter
  saves, Esc cancels, ADR-0095) with a one-line hint, and the engine's answer under it.
- **SNMP device Settings:** "Address on the network" with the same editor. The engine tells the
  page that the project is on a network (`on_network`), so the field appears before an address
  exists. The 127.x field is labelled "Address on this PC". The device's error now says "Type one
  in its Settings tab".
- **Safety:** an SNMP settings save keeps the stored network address (`UpdateDevice`), so an
  editor holding an older copy cannot undo a new address.

## Evidence

- Go: `go test ./internal/netaddr/ ./internal/labsetup/ ./internal/snmp/... ./internal/app/
  ./internal/api/...` pass. The new `netaddr` tests cover:
  - lab card: IED moved, SNMP duplicate refused, SNMP saved;
  - real card: plan with every address kept, prefix from the card (/25), plan problem passed on,
    refused while running;
  - This PC only.
- Frontend: `tsc --noEmit` clean; 483 tests pass (the reachability test follows the new wording).
- Scratch engine with a sample IED and one SNMP device on the Lab card, at 1366x768:
  - Devices shows "Type an address ✎ :102" with the hint.
  - Typing 192.168.235.20 with the owner's network service stopped showed "The lab network
    service is not running. Press Repair." in the error bar, and nothing changed.
  - The SNMP Settings tab shows "Address on the network — Type an address ✎ :161".

## Limitations

- The happy paths were not clicked through on this PC. The network service is stopped, so the
  lab card refuses; the real-card path needs the owner's Windows prompt. Both are covered by the
  unit tests.
- Not in the installed 0.1.32. The EXE was not rebuilt (owner rule). Python engine untouched.

## Release 0.1.33 (with changes 0212–0214)

- `compile_go_release.py --version 0.1.33` with `studio\.venv`: antivirus gate 19 programs, 0 findings.
- Packaged portable on a throwaway data folder: version 0.1.33; `GET /network/leftovers` 200; the address
  route answers `not_found` for an unknown IED; sample IED imported, Start, an independent MMS read
  of `Feeder01CTRL/XCBR1.Pos.stVal` with `studio_probe` (ok, 01), Stop.
- `website/deploy.cmd` under `fnm exec --using=24`: Pages 9dd30909, "Saved release 0.1.33 from
  2026-11-04"; installed apps are told about 0.1.33 by the signed release record.
- Live installer b74004d0 (19,593,782 bytes) and portable 2d44c952 (53,242,368 bytes) match the build;
  copies of both downloads scan clean (2 programs, 0 findings). The changelog page shows
  "0.1.33 · Nothing left on your network, addresses edited where you see them".
- This PC's curl needs `--ssl-no-revoke` for download.scadabench.com (revocation check offline).
