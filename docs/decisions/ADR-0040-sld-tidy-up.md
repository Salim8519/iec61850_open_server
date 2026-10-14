# ADR-0040: Tidy up the single-line diagram in the browser

Accepted: 2026-10-14. Owner request: a smart "tidy up" so a random diagram becomes a
clear, real-looking single-line diagram. Specification: [SLD tidy](../specs/sld-tidy.md).

Hand-drawn and imported diagrams drift into crossing wires and uneven spacing.
Engineers had to drag every symbol back into line.

**Decision**
- **A pure frontend layout function** (`frontend/src/equipment/tidy/`), like the
  clipboard helpers: it changes only positions, wire sides and symbol turns. It is
  drawing geometry, not simulation state, so it does not belong to the backend. It
  needs the canvas's measured symbol sizes, and the result goes through the existing
  revision-guarded `PUT /project/diagram` as one undo step. There's no new endpoint or
  contract and no format change.
- **The SLD-specific layered layout** (Sugiyama style):
  - busbar detection from junction clusters, with double busbars recognised from shared bays;
  - wire-distance rows;
  - barycentre crossing reduction with bar blocks that keep their order;
  - weighted isotonic regression for columns, so runs stay straight and branches step aside.
- **Repeat until settled** (at most four passes) so a second press does nothing.
- **Split by responsibility:** `graph.ts` (wiring, groups, busbars, distances),
  `rows.ts` (rows), `position.ts` (columns and row heights), `wires.ts` (wire sides and
  symbol turns) and `tidy.ts` (groups, parking, anchoring, passes). The UI is
  `features/diagram/TidyButton.tsx` with wording in `tidyCopy.ts`.
- **Tooling:** `frontend/tsconfig.json` sets `allowImportingTsExtensions`. The tidy modules
  import each other with `.ts` paths so the Node test runner can load them without a
  bundler. This is allowed with `noEmit`, and Vite resolves it unchanged.

**Compatibility:** additive. Diagrams, project format 12, API and worker behaviour are
unchanged. A tidy writes an ordinary diagram revision. Wire `source`/`target` may swap
ends (the wiring is undirected in loose connection mode); wire ids are kept.
Development and packaged builds run the same code.
