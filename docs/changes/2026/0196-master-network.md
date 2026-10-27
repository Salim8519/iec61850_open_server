# 0196: The SCADA master's Network: All networks or one card of this PC

Date: 2026-10-27. Decision: [ADR-0123](../../decisions/ADR-0123-master-network.md).

## Change

| File | Why |
| --- | --- |
| `native/master/client_104.cpp`, `client_dnp3.cpp`, `client_modbus.cpp`, `modbus_link.cpp/.hpp` | connect from `settings.local` when set; Modbus says at once when Windows has no route from that address |
| `engine/internal/gateway/service/master_network.go` (+ test), `master_connect.go`, `master.go` | list of this PC's addresses, save check, the address the device sees, the warning, `local` sent to the worker on connect and reconnect |
| `engine/internal/api/toolsapi/master_network.go`, `routes.go`, `master_list.go`; `engine/internal/app/wire_tools.go` | `GET/PUT /tools/master/network` (Go only); adapters from the IP Helper API |
| `frontend/src/features/tools/gateway/master/MasterNetwork.tsx`, `MasterConnect.tsx`; `api/master.ts`; `copy/master.ts` | the Network field under Device address |

## Evidence

- Go: `TestMasterNetworkListsThePCsOwnAddresses`, `TestMasterNetworkSavesOnlyAnAddressOfThisPC` pass;
  verify: all checks pass except the known Windows Defender block of the `goosedom` test program.
- Frontend 488/488; types clean; file lengths pass.
- Dev app (Go engine) with a TCP listener on this PC that logs each peer address: Modbus, IEC 104 and
  DNP3 from All networks and from Wi-Fi 192.168.118.112 arrived from 192.168.118.112; from VMnet2
  10.10.10.1 no connection reached the listener: Modbus answered in 0.3 s "This PC cannot reach
  192.168.118.112:1502 from 10.10.10.1. Choose All networks, or the network the device is on.", IEC 104
  answered "No answer … within 3 s" followed by the Network warning. An address not on the PC
  (8.8.8.8) is refused. The ScadaBench Lab IED addresses are not offered.

## Limitations

- Not tested against a remote gateway on a second PC; the bind and its refusal were shown locally.
- IEC 104 and DNP3 report an unroutable address as no answer (library behaviour), with the warning.
- The station gateway role still listens on This PC only or All network adapters.
