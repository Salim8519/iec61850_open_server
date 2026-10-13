# 0044: Configurable state colours for two-state equipment

Date: 2026-09-29. Decision: ADR-0026. A closed breaker was only visible from
its small contact geometry and a tiny pill; users had to zoom in.

Changes: optional `active_color`/`inactive_color` in the backend contract and
frontend types; `stateColors.ts` (defaults, overrides, fill rules, contrast);
card fills in `EquipmentNode.tsx`/`equipment-node.css`; `AppearanceEditor.tsx`
with `appearance.css` in Properties; "Apply to all" in `App.tsx`.

Evidence:

- `test_equipment_colors.py` (9: defaults, valid, 6 invalid values, stored
  with diagram) and `stateColors.test.ts` (7). Backend unit + integration:
  133 pass; frontend node tests: 42 pass; build and `verify.py` pass.
- Isolated project copy while running: CB-1 and ES-1 closed through the real
  operate path (IED feedback Dbpos=2) and rendered green and custom amber;
  DS-1 open rendered white; trip relay Operated red; latched push button
  green. Clicking Reset to defaults saved one revision with both colours cleared.
- Protocol suite: one failure in a run slowed by a parallel build (61 s vs
  ~20 s); three following runs passed 22/22. The failing test was not captured;
  no backend engine or native code changed.

Limitations: "Apply to all" not exercised through the UI (single object per
kind in the test project). Owner visual acceptance pending.
