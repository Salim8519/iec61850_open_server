# ADR-0107: Gateway tap positions, step commands, set points and command steps

Accepted: 2026-10-22. Owner request: add the SCADA command kinds a substation gateway uses most
(regulating step, set points) to the gateway simulator, route them to the linked IEC 61850
controls, and honour select/execute and deactivation for every command. Change
[0170](../changes/2026/0170-gateway-step-setpoint-commands.md). Builds on
[ADR-0057](ADR-0057-gateway-explorer-commands.md), [ADR-0091](ADR-0091-gateway-own-points.md) and
[ADR-0095](ADR-0095-universal-point-kinds.md); calls the typed IEC 61850 commands of ADR-0104
(branch `cmd-61850-0167`).

## Decision

- **Three Go-only point kinds** join the six Python kinds (`gateway.TypedKinds`; `Kinds` keeps
  the Python list, order and texts):

  | Kind | Link | IEC 104 | DNP3 | Modbus |
  | --- | --- | --- | --- | --- |
  | `step_position` | an INT* signal (BSC `valWTr.posVal`) | M_ST_NA_1 5 / M_ST_TB_1 32 (-64..63) | analog input g30v1 / g32v3 | input register, int16 |
  | `command_step` | a BSC or ISC control | C_RC_NA_1 47 / C_RC_TA_1 60 | CROB: close / ON = raise, trip / OFF = lower | refused (no step on Modbus) |
  | `command_setpoint` | an APC or INC control | C_SE_NA_1 48, NB 49, NC 50 / TA 61, TB 62, TC 63 | g41 var 1/2/3 (status g40 by format) | holding register(s) FC6 / FC16 |

  A set point has `format` (104: `float`, `scaled`, `normalized`; DNP3: `float`, `int16`,
  `int32`; Modbus: `float`, `int16`, `uint16`, `int32`; the first is the default) and, on
  Modbus, `word_order` (`high_first` default, `low_first`). A 32-bit Modbus set point takes two
  registers; DNP3 set points have their own analog output index space. The value sent to the IED
  is the SCADA value divided by the point's scale (SCADA sees IED value × scale), rounded for INC.
- **Validation** checks the control class (BSC/ISC, APC/INC), the signal type of a tap position,
  the protocol and the format, with one sentence per problem. CSV gains `format,word_order`
  columns only when a list has a set point. Own points: a tap position value (-64..63), a step
  command that moves an own tap position or whole number by one, a set point that sets an own
  measured value, whole number or tap position.
- **Routing table in Go** (`service/routing.go`, pure): `PhaseOf(protocol, wire)` maps what the
  outstation decoded to select / execute / cancel (104: COT 6 + S/E, COT 8; DNP3: select,
  operate, direct, direct without answer; Modbus: execute); `Route(kind, phase, selected)` picks
  SelectTyped, OperateTyped, CancelTyped or the existing single/double path; `CommandValue`
  converts the value. A select keeps the selection until execute, cancel or the IED's time-out.
- **Single and double commands**: a direct execute keeps the existing path (one operate through
  `studio_control`, real IEDs included). A select now goes to `SelectTyped`; while that answers
  `not_available` (the temporary adapter) the select is confirmed by ScadaBench as before, and
  the execute takes the existing path. After ADR-0104 merges, select and execute reach the IED.
- **Channel extension (backward compatible both ways).** studio_gateway answers `hello` with
  `typed_commands`; the engine then adds `"typed_commands": true` to `start`. Only then does the
  outstation hand every command step to the engine, adding `"wire"` to `gw_command`
  (`{type_id, select, cot, test}` for 104, `{function, code | variation}` for DNP3,
  `{function}` for Modbus) and a typed `value` (step state, number). Without the flag (the
  frozen Python engine) select and cancel are confirmed by the outstation exactly as before and
  the new 104 types are unknown (COT 44). An engine with typed points refuses an older gateway
  program ("too old … rebuild the native programs"). `command_result` is unchanged.
- **Confirmations.** IEC 104: ACT_CON positive/negative for select and execute, ACT_TERM after
  a successful execute, DEACT_CON for COT 8; unknown IOA COT 47, wrong type or format COT 44,
  wrong cause COT 45, each with a plain Traffic line. DNP3: SUCCESS or HARDWARE_ERROR from the
  engine, NOT_SUPPORTED, NOT_AUTHORIZED. Modbus: echo, or exception 1/2/3/4.
- **Contract with ADR-0104** (`runtime/typed_command.go`, marked `// Replaced by ADR-0104 at
  merge`): `CommandOptions`, `TypedCommand`, `SelectResult`, `SelectTyped`, `OperateTyped`,
  `CancelTyped`. The temporary adapters answer `not_available`, except `OperateTyped` with a
  bool value, which uses `OperateControl`. The gateway reaches them through the
  `service.TypedCommands` port (`runtime/ports/gateway_typed.go`), with orCat 3 (remote) for
  SCADA, 2 (station) for the page, and the 104 test bit.
- **Page**: the kinds show their name, type IDs and hint in every kind list
  (`copy/gatewayTypedKinds.ts`, merged into the type catalog later); a step row has Raise… /
  Lower…, a set point row a number box and Send…; `POST /tools/gateway/command` also takes
  `"higher"`, `"lower"` or a number.

## Compatibility

- The six existing kinds keep their stored form, Python oracle results, CSV bytes, problem
  texts and wire bytes. Existing native tests run unchanged (they start without the flag).
- The frozen Python engine has none of this. It refuses to open a project whose gateway list
  holds a new kind ("A stored … gateway point has no valid type or address"); projects without
  them are unchanged.
- An IEC 104 deactivation of an unknown IOA now gets COT 47 in typed mode (before: confirmed).
- Not done: 104 transient indicator of M_ST (always 0), DNP3 g41 var 4 (double), Modbus step
  commands, a select time-out of the gateway's own, format editing in the point row (CSV or
  API; the Add points dialog is rebuilt separately).
