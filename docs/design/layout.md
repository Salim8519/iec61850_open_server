# Fixed visual design (v2, ADR-0032)

| Token | Value |
| --- | --- |
| Background | #F5F6F8 |
| Surface | #FFFFFF (sidebar, top bar, cards, sheets) |
| Text / secondary / tertiary | #111827 / #4B5563 / #6B7280 |
| Border | #E5E7EB, strong #D1D5DB |
| Accent | #2563EB on #EFF6FF (one accent only) |
| Success / Warning / Error | #15803D / #B45309 / #B91C1C with soft tints |
| Font | System UI, 15 px base, scale 13/15/17/20/26, line-height 1.5, rem units |
| Controls | 40 px high, primary actions 44 px, 8–10 px radius, 20 px icons |
| Contrast | WCAG AA for text, accent and status colours (enforced by a test) |

Shell: 232 px white sidebar with four places (Diagram, Devices, Test, Settings at the
bottom with Activity) that collapses to 72 px icons below 1100 px; a 64 px top bar
with page title, one-line subtitle, File menu, run state words and one Run/Stop pill.
Pages scroll inside a centred 1440 px column with 20 px gaps between cards.
Targets: 1366×768 and 1920×1080.

Calm rules: one title and one subtitle per page; one primary action per area; rare
options inside a single "Advanced" disclosure; hints only in empty states, wizards
and unclear settings; jargon through a "?" InfoTip. No gradients, decorative
dashboards, external fonts or remote runtime assets.

Diagram: full-height canvas on #FBFCFD with a floating toolbar (Add, Undo, Redo) and
zoom controls bottom-left. Add opens large labelled tiles grouped Switching,
Measuring, Protection & control, Drawing. The 380 px right sheet appears only when
something is selected: status pill, Operate (while running), name and start position,
"comes from" / "goes to" connections, then Advanced and object actions.

Devices: 280 px list (name, data points, role/status pill) beside one detail card
with a Simulate / Real device / Off segmented control and tabs Overview, Data points,
Compare with SCD (real devices) and Files. Test and Settings switch sections with a
segmented control.

Status display: words + icon + border pattern (solid good/info, dotted waiting,
dashed warning/off, double problem); colour only repeats the words. Library category
accents describe equipment families, never energized or protection state.

Equipment (unchanged rules): CB contacts translate and DS/ES blades rotate over
300 ms after confirmed feedback; Unknown/Invalid never resemble Open; reduced motion
disables interpolation. Trip relay states Reset/Operated/Awaiting/Invalid; push button
has no terminals and is pressed by clicking it. Relay panels show one row per signal
with a value chip (filled 1, hollow 0, dashed —). State fills (ADR-0026) colour only
the moving part; Intermediate is dashed amber and Invalid dashed red.
