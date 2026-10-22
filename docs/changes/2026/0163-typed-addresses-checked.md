# 0163: Network setup: typed addresses are checked like planned ones (Go engine only)

Date: 2026-10-22. Decision [ADR-0100](../../decisions/ADR-0100-typed-network-addresses-checked.md); found
by the UI review [0160](0160-ui-review-no-clipping.md). The frozen Python backend is unchanged and still
answers "ok" for these cases (intended difference).

## What changed

- `engine/internal/netcfg/typed_addresses.go` (new), `service_preview.go`, `address_plan.go`: typed
  addresses get `duplicate`, `used_by_other_adapter` and `already_on_adapter`, in memory; planned rows
  hold their address first, then typed rows in list order; a stale planned duplicate is cleared;
  adapter-strategy picks avoid typed addresses; spaces around a typed address or prefix are ignored.
  Hints say e.g. "FEED1 already uses 172.16.1.21. Each IED needs its own address."
- `features/network/typedAddressCheck.ts` (new), `AddressTable.tsx`, `NetworkSetup.tsx`,
  `copy/addressHints.ts`, `copy/status.ts`: the browser shows a duplicate or own-PC clash at once, trims
  typed values, waits 300 ms (was 500 ms) before asking the engine, and keeps Apply off until the
  engine's answer belongs to what is typed.

## Evidence

- Go: 12 new cases in `typed_addresses_test.go` (duplicates against SCD, planned and typed rows, own
  adapter, other adapter, free address stays ok with a probe-count check, refused rows not probed, spaces,
  stale duplicate cleared, SNMP rows, Apply refused); `go test ./internal/netcfg/...` ok; generated
  parity vectors and HTTP golden files unchanged and passing.
- Frontend: `typedAddressCheck.test.ts` (7); 372/372 merged.

## Limitations

- Not tried against real Windows adapters in a running app (tests use fakes; nothing changes Windows).
- Two untyped rows that share this PC's own address still both read "already on adapter", as before.
