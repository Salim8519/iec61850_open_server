# ADR-0022: Relay as a named status set

Accepted: 2026-09-17.

The RELAY canvas equipment becomes a named, expandable set of signal statuses
instead of a connectable wire-path object:

- A relay node renders a non-connectable panel: its name and one row per
  status entry, each row showing the entry name and its live value. Relay
  nodes have no terminals; wiring other equipment is unchanged.
- Each status entry stores a user-assigned label and one exact signal binding
  (`RelayStatus` in the project contract, at most 50 entries per relay). Any
  imported signal may be assigned. Only runtime-editable signals accept manual
  initiation; the Inspector marks the others display-only.
- While running, an entry can be initiated through the existing test-panel
  runtime paths: set value, Boolean pulse, or release override. There are no
  new endpoints and no new native behaviour; overrides remain per signal and
  the Signal Test tab stays the authoritative override manager.
- Live projection reuses the equipment state store with composite keys
  `nodeId:entryId` per entry, so hydration and event plumbing are unchanged.
- The project format bumps to 9. Existing relays with a single legacy state
  mapping migrate to one named status entry with the exact binding preserved
  and a visible notice; IED removal counts and strips relay entry bindings.

The relay symbol loses its connector stubs because a status panel is not part
of the electrical path. Rejected alternative: keeping relays in the wire path
with multiple mapped signals — it would continue to suggest electrical
connectivity that relays do not have, and a wire-path symbol cannot host a
variable status list.
