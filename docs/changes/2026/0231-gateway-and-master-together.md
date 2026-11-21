# 0231 Station gateway and SCADA master run at the same time (0.1.36)

Date: 2026-11-22. Owner: "now lets focus on 0.1.36: it should allow master and stations work at
same time". Decision: [ADR-0143](../../decisions/ADR-0143-gateway-and-master-together.md).
Layers: Go engine (`internal/gateway`, `internal/gateway/service`, `internal/api/toolsapi`),
frontend (SCADA tool header, shared `SegmentedControl`), website docs page (not deployed).

## What changed

- **Engine.** New `gateway.RoleChangeRefusal` / `gateway.RoleLocks` (`role_change.go`): only
  Protocol converter is locked while the gateway runs or the master is connected; Station gateway
  ⇄ SCADA master always switches. The role checks in gateway Start and master Connect are gone.
  `/activity` carries `role_locks` only while a role is locked. Master values refresh the gateway
  table only under the converter role.
- **Screen.** "Act as" greys out only the locked button, with the engine's reason in its tooltip;
  every running role shows the green live dot and "(running)". The help says which roles run
  together. The role in use is never greyed out (the look the owner asked about on 2026-11-21).
- **Docs.** `website/.../tools/scada-master.md`: "Run the master and the gateway together"
  replaces "Stop or disconnect first".

## Evidence

- `go test ./internal/gateway/... ./internal/api/toolsapi/ ./internal/app/ ./internal/push/...`:
  all pass (temp folders in `.local/gotmp`). New: `TestRoleChangeRefusal`, `TestRoleLocks`,
  `TestMasterKeepsRunningWhileTheGatewayViewOpens` (real `studio_master.exe` connected to a real
  `studio_gateway.exe` device; switches master → gateway → master while connected; converter
  refused with the engine's reason; `role_locks` absent when idle). Updated: `TestGatewayRefusals`,
  `TestMasterRoleDiscoversNamesAndCommandsADevice`. Routes parity with Python (idle answers) passes.
- Frontend: `tsc --noEmit` clean; `bugHuntQ2.test.ts` updated and passing.
- Live, freshly built engine (port 8790, dev data): gateway started (IEC 104, 0.0.0.0:2406), role
  switched to SCADA master while it ran (200), master connected to 127.0.0.1:2406 and received
  2,652 points; gateway listed the master as its client; both buttons showed the live dot and stayed
  clickable, Protocol converter disabled with the reason; switching back to Station gateway kept
  both running ("Stop gateway" shown). Header row fits at 1366×768 without clipping. Test settings
  restored afterwards.

## Limitations

- The Python engine (frozen) still allows one role at a time; Go is the engine of dev and release.
- The master-to-own-gateway check uses ScadaBench's two programs; no third-party 104 client in
  this change (the gateway and master wire behaviour itself is unchanged).
- Website docs change goes live with the 0.1.36 site deploy. No EXE built.
