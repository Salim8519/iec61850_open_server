# 0142: IED roles renamed: Simulate / Monitor real device / Skip, one sentence each

Date: 2026-10-20. Owner: "real ied confusing, no one knows the purpose of it anymore, or make it
more meaningful"; "left out the same issue, confusing"; approved: "Yes, rename the roles to
Simulate / Monitor real device / Skip with one sentence under each."

## Change

- `frontend/src/copy/roles.ts` (new): label, chip and one sentence per role; the help tip next to
  the choice ("Which one should I choose?") lists all three. Used by the Devices role control,
  filter chips (Simulate / Monitor real / Skip), list badges (Simulate · stopped, Monitors real
  device, Skipped), `RoleLine`, overview, health view, top-bar and run tooltips, `NoPublishers`.
- `RoleLine`: starts with "<Role>: <sentence>". Bug fixed: a monitored real device showed the
  simulated address (`ied.ip`, e.g. 127.50.0.3); it now shows the device address the backend
  uses (typed address, else the SCD address; `realEndpointOf`), or "Type the device address below".
- Same wording in both engines' messages (text only, owner-approved; keeps Go = Python):
  Run refusals, SLD operate refusal, health "This IED is skipped.", network plan refusal
  (`runtime/lifecycle.go`, `runtime/equipment_control.go`, `acquisition/health/station.go`,
  `netcfg/service_plan.go` and their Python counterparts); recorded goldens updated accordingly.
- Stored role values (`simulated`, `real`, `off`) and the API are unchanged; projects need nothing.

## Evidence

- Frontend tests 232/232 (new `iedRoles.test.ts`), `npm run check` clean; Go tests for runtime,
  netcfg, acquisition, platformapi, launcher pass; pytest role/network tests 6 passed.
- Devices page at 1366x768: control, chips, badges and role line show the new words untruncated.
