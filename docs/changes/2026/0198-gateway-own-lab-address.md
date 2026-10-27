# 0198: Station gateway own address on the lab network

Date: 2026-10-27. Decision: [ADR-0125](../../decisions/ADR-0125-gateway-own-lab-address.md).

## Change

| File | Why |
| --- | --- |
| `engine/internal/labsetup/own.go` (+ `own_test.go`), `service.go`, `apply.go`, `service_test.go` | check and add an own address, put it back, remove it; `Reserved` blocks IED rows and keeps it on Use |
| `engine/internal/gateway/service/gateway_own.go`, `gateway_network.go` (+ test), `manager.go` | own address list, Add, bind there after ensuring it, leave the card on another choice |
| `engine/internal/project/gateway/network_keys.go` (+ test), `gateway.go` | keep `own_addresses` on load |
| `engine/internal/api/toolsapi/network.go`; `app/wire_lab.go` | `POST /tools/gateway/network/own`; gateway and lab service wired both ways |
| `frontend/src/features/tools/gateway/GatewayNetwork.tsx`; `api/gateway.ts`; `copy/gateway.ts` | Own address group and "Type an address…" in the one Network list, hints |

## Evidence

- Go tests: labsetup (own address added, IED/card/other device/loopback refused, real device by ARP
  refused, IED row blocked, Use keeps it, put back and removed), gateway service
  (`TestGatewayOwnAddressOnTheLabCard`), project gateway (`TestOwnAddressesSurviveALoad`), app pass.
- Dev app (lab card installed): typing an IED's SCL address showed "IED … uses …"; 10.10.0.100 was
  added (Windows: ScadaBench Lab, /32, SkipAsSource False, ping answers). DNP3 gateway started and
  listened on 10.10.0.100:20000 only; an independent TCP client connected to 10.10.0.100 and was
  refused at 127.0.0.1 and 192.168.100.185. Choosing All networks took 10.10.0.100 off the card.
  The owner's gateway was left on DNP3, All networks; 10.10.0.100 stays in the list.

## Limitations

- Reached only from this PC (lab card); another PC or VM uses the machine's address.
- No SCADA master own address yet (next change).
- An address list entry cannot be deleted from the list yet.
