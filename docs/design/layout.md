# Fixed visual design (v2, ADR-0032)

| Token | Value |
| --- | --- |
| Background | #ECEEF2 (sidebar #F7F8FA, canvas #E9EDF2) |
| Surface | #FFFFFF (sidebar, top bar, cards, sheets) |
| Text / secondary / tertiary | #0F172A / #3F4856 / #5B6472 |
| Border | #D3D8E0 cards/rows, strong #A9B1BD controls; 1.5 px (`--line`), ADR-0034 |
| Accent | #2563EB on #EFF6FF (one accent only) |
| Success / Warning / Error | #15803D / #B45309 / #B91C1C with soft tints |
| Font | System UI, 15 px base, scale 13/15/17/20/26, line-height 1.5, rem units |
| Controls | 40 px high, primary actions 44 px, 8–10 px radius, 20 px icons |
| Contrast | WCAG AA for text, accent and status colours (enforced by a test) |

Shell: 232 px white sidebar with four places (Devices, SLD, Test, Settings at the
bottom with Activity) that collapses to 72 px icons below 1100 px; a 64 px top bar
with page title, one-line subtitle, File menu, run state words and one Run/Stop pill.
Pages scroll inside a centred 1440 px column with 20 px gaps between cards.
Targets: 1366×768 and 1920×1080.

Calm rules: one title and one subtitle per page; one primary action per area; rare
options inside a single "Advanced" disclosure; hints only in empty states, wizards
and unclear settings; jargon through a "?" InfoTip. No gradients, decorative
dashboards, external fonts or remote runtime assets.

Diagram: full-height canvas on #E9EDF2 with #B4BCC8 dots, #1F2937 2.5 px wires and
white symbol cards outlined #7D8796, with a floating toolbar (Add, Undo, Redo) and
zoom controls bottom-left. Add opens large labelled tiles grouped Switching,
Measuring, Protection & control, Drawing. The 380 px right sheet appears only when
something is selected: status pill, Operate (while running), name and start position,
"comes from" / "goes to" connections, then Advanced and object actions.

Devices: 280 px list (name, data points, role/status pill) beside one detail card
with a Simulate / Real device / Off segmented control and tabs Overview, Data points,
Compare with SCD (real devices) and Files. Test and Settings switch sections with a
segmented control.

Live colouring (running only): live red solid 4 px, maybe live amber dotted, dead grey
dashed, earthed green dash-dot, earth fault purple long dash 5 px; a bottom-right legend
names each style. Switching hints use an amber dashed box with an icon and words.

Status display: words + icon + border pattern (solid good/info, dotted waiting,
dashed warning/off, double problem); colour only repeats the words. Library category
accents describe equipment families, never energized or protection state.

Equipment (ADR-0045): CB body fills with a whole conductor when closed and is hollow
with a broken conductor when open; DS/ES blades (IEC contact bar) swing 40° open; all
move over 300 ms after confirmed feedback. Symbol strokes thicken in zoom bands (3/4/5.5)
and a confirmed position change pulses a blue ring twice; Unknown/Invalid never
resemble Open; reduced motion disables interpolation and holds a static ring. Trip relay states Reset/Operated/Awaiting/Invalid; push button
has no terminals and is pressed by clicking it. Relay panels show one row per signal
with a value chip (filled 1, hollow 0, dashed —). State fills (ADR-0026) colour only
the moving part; Intermediate is dashed amber and Invalid dashed red.
