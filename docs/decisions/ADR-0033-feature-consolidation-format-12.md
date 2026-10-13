# ADR-0033: Feature consolidation and project format 12

Accepted: 2026-10-14. Owner decisions: remove the hidden legacy symbols, merge
duplicate features, simplify buttons/relays with rare options under "Advanced", and
keep the CONCEN compatibility profile where it is.

- **Format 12** retires TRANSFORMER, LOAD and BUSBAR and the busbar `width` field.
  Opening an older project drops those nodes and their wires, lists each by label in
  the migration notice, and keeps the usual verified recovery archive.
- **One value path**: `features/test/signalActions.ts` is the only frontend caller of
  `/signals/update`, `/signals/pulse`, `/signals/release-override` and
  `/signals/source`. `ValueControl` serves Devices, Test points and relay rows;
  `SourceEditor` serves SOURCE symbols and Test › Generators.
- **Buttons and relays**: data is unchanged; trip polarity is shown as one sentence
  and edited under Advanced; push-button modes default to latching and live under
  Advanced.
- Choosing a breaker/switch command fills its position from the same control's
  status point when no position is mapped yet (beginner shortcut, still editable).
- Bulk IED deletion UI and the unused `GET /network/leftovers` route are removed;
  single deletion with preview and backup remains.
- Real-world SCL: `PhyComAddr` attributes map to libiec61850 type 29; unnamed
  `ReportControl` elements are skipped with a warning instead of failing the import.

Compatibility: older Studio builds cannot open format 12. The CONCEN profile is
unchanged. Backend APIs used by the UI are unchanged.
