# 0197: Station gateway Network, SCADA address line and allowed SCADA address

Date: 2026-10-27. Decision: [ADR-0124](../../decisions/ADR-0124-gateway-network.md).

## Change

| File | Why |
| --- | --- |
| `native/gateway/server_104.cpp`, `modbus_server.cpp` | refuse a caller that is not `settings.allowed`, with a Traffic line |
| `engine/internal/gateway/service/gateway_network.go` (+ test), `start.go`, `manager.go`, `master_network.go` | Network, address SCADA connects to, save checks, bind to the chosen card, allowed address to the worker; shared address list |
| `engine/internal/project/gateway/network_keys.go` (+ test), `gateway.go`, `master.go` | keep `listen_address`, `allowed_scada` and the master's `local` (0196 lost it on reload) |
| `engine/internal/api/toolsapi/network.go`, `routes.go`, `master_list.go`; `app/wire_tools.go`, `routes_test.go` | `GET/PUT /tools/gateway/network` beside the master's; Go-only route list |
| `frontend/src/features/tools/gateway/GatewayNetwork.tsx`, `GatewaySettings.tsx`, `GatewayTool.tsx`, `converter/ConverterTool.tsx`; `api/gateway.ts`; `copy/gateway.ts` | Network field, "SCADA connects to" line, Allowed SCADA address in Advanced |

## Evidence

- Go tests: `TestGatewayNetworkBindsTheChosenCardAndAllowedScada`, `TestNetworkAddressesSurviveALoad`,
  service/project/api packages pass; verify passes except Windows Defender blocking two test programs
  (`goosedom`, `runtime/ports`), both PASS when built to D: and run.
- Frontend 488/488; types clean; file lengths pass.
- Dev app: DNP3 on vEthernet listened on 172.17.48.1:20000 only. IEC 104 (port 2414; 2404 is used by
  another program on this PC) on 172.17.48.1 with allowed 10.10.10.1: a client from 172.17.48.1 was
  closed, Traffic "connection refused: only the allowed SCADA address 10.10.10.1 may connect"; loopback
  could not connect; with allowed 172.17.48.1 the same client was accepted ("Master connected").
  Modbus on 1503 refused the same way. The owner's gateway settings were restored afterwards
  (DNP3, All networks; 104 port 2406 and Modbus 502 from the last known values).

## Limitations

- DNP3 cannot refuse by SCADA address (shown in the UI).
- Not tested with SCADA on a second PC.
