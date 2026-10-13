# ADR-0032: Clean four-place UI and design system v2

Accepted: 2026-10-14. Owner request: the app looked messy, old and unprofessional
("like an airplane cockpit"); it must be clean and organised like an iPhone, friendly
to beginners and older people, and still pro level. Supersedes the tab list of
ADR-0028 and the visual tokens of the 2026-09-09 colour refresh (ADR-0002).
Specification: [UI structure](../specs/ui-structure.md).

- Navigation is a left sidebar with four places: **Diagram, Devices, Test, Settings**.
  Activity (event log) is a slide-over. The top bar shows title, subtitle, File menu
  and one Run/Stop button; Pause automation sits in a small "…" menu.
- Devices merges IEDs & Signals, Real IEDs and the endpoint table: a list plus one
  detail card with a Simulate / Real device / Off switch and tabs Overview, Data
  points, Compare with SCD (real only) and Files.
- Test merges Signal Test, Scenarios and value generators: Test points, Generators,
  Sequences.
- Settings holds Network & GOOSE and About.
- Diagram uses the full canvas, a floating toolbar with a tile-based Add menu, and a
  right-side sheet that appears only when something is selected.
- Tokens: light shell, one blue accent, 15 px text, 40/44 px controls, 10 px radius,
  AA contrast, rem sizing. Old variable names alias the new tokens.
- Hints move from "under every field" to empty states, wizards and genuinely unclear
  settings; jargon is explained with a "?" InfoTip.

Compatibility: no backend contract change. The legacy demo is untouched. Users who
knew the old tabs find: Connections/Network → Settings, Real IEDs → Devices,
Signal Test/Scenarios → Test, Palette → Diagram › Add.
