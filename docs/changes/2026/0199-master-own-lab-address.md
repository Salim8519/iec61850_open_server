# 0199: SCADA master own address on the lab network

Date: 2026-10-27. Decision: [ADR-0125](../../decisions/ADR-0125-gateway-own-lab-address.md) (SCADA master section).

## Change

| File | Why |
| --- | --- |
| `engine/internal/gateway/service/master_own.go`, `master_network.go` (+ test), `master_connect.go`, `master.go` | own address list, Add, connect from it after putting it back, warn when the device is on a real card, leave the card on another choice |
| `engine/internal/gateway/service/gateway_network.go` | typed addresses shown as "ScadaBench Lab" |
| `engine/internal/project/gateway/master.go` | keep the master's `own_addresses` on load |
| `engine/internal/api/toolsapi/network.go`; `app/wire_lab.go` | `POST /tools/master/network/own`; lab page sees the gateway's and the master's addresses |
| `frontend/src/features/tools/gateway/TypeAddress.tsx`, `GatewayNetwork.tsx`, `master/MasterNetwork.tsx`; `api/master.ts`; `copy/master.ts`, `copy/gateway.ts` | one shared "Type an address…" box; the master's list gets the ScadaBench Lab group |

## Evidence

- Go tests: `TestMasterOwnAddressOnTheLabCard`, gateway, labsetup, project gateway, api and app pass.
- Frontend 488/488; types clean.
- Dev app: gateway own address 10.10.0.100 on the card; the master's 10.10.0.100 was refused
  ("Station gateway uses 10.10.0.100"); master own address 10.10.0.200 added. An independent TCP
  listener on 10.10.0.100:2414 logged "peer 10.10.0.200" for the master's IEC 104 connect (2404 is
  held by another program on this PC). Afterwards the owner's settings were restored (master host
  empty, IEC 104 port 2404, role Station gateway, gateway All networks) and both addresses were off
  the card; they stay in the lists.

- Release 0.1.28 rebuilt in place with 0194-0199 (owner: "push the new version as 0.1.28 too"):
  installer 200f12e3, portable 889bb916; portable self-test ok; live installer, portable and the
  release record hashes match; `/download/ScadaBench-Setup.exe` redirects to 0.1.28. Unsigned as before;
  previous files kept in `D:\ScadaBench-workackup-0.1.28-0be5b9c0`. The website build needed Node
  22.23 (pi-node) instead of the PATH Node 22.11. Apps already on 0.1.28 are not offered the rebuild.

## Limitations

- The master's screen was not clicked through: the browser pane stopped drawing (window behind
  another window). It reuses the gateway's checked list and box.
- After the test the dev browser tab logged many `ERR_NETWORK_CHANGED` errors (Chromium cancels
  requests when this PC's addresses change) and stayed blank on reload; a new tab loaded normally.
  Not yet investigated whether the packaged app's window recovers by itself.
- Gateway and master cannot run at the same time in one ScadaBench (one role), so master-to-gateway
  over the lab network on one PC needs the protocol converter role or a second program.
