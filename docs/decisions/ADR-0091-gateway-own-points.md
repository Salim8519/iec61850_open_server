# ADR-0091: Gateway own points (made-up values the user controls)

Accepted: 2026-10-20. Owner: "the gateway has auto add ieds points, that is fine, what about
custom ones? ... let him do anything and made up point, while keep the auto add there". Lane G,
log [lane-g-own-points](../architecture/go-engine/streams/lane-g-own-points.md); change
[0144](../changes/2026/0144-gateway-own-points.md).

## Decision

- A gateway point may be an **own point**: `source:"manual"`, id `own:<n>`, no IED/reference, a
  `value` served with good quality while the gateway runs; a command own point may name a
  `target` own status point that follows its commands. Same type, address-range, scale and
  clash rules as IED points; the IED lookup is skipped.
- Two Go-only routes: `POST /api/v1/tools/gateway/points/new` (next id, first free address) and
  `PUT /api/v1/tools/gateway/points/value` (change a value; sent at once when running, saved).
  `POST /tools/gateway/propose` takes an optional `keep` list so Propose keeps own points.
- CSV gains `source,value,id,target` columns only when own points exist; IED-only lists and old
  six-column files are unchanged.

## Compatibility

- IED points behave exactly as before (Python oracle tests unchanged). Projects without own points
  are byte-identical.
- Go engine only. The frozen Python engine skips own points at Start ("its IED is not in the
  project"), reports them on CSV import, and answers 404 on the two routes.
