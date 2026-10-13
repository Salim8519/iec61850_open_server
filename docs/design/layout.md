# Fixed visual design

| Token | Value |
| --- | --- |
| Background | #F0F5FA |
| Surface | #FFFFFF |
| Text | #16324F |
| Muted text | #52677E |
| Border | #D5E1EC |
| Primary | #1D4ED8 |
| Header | #112B46 with white text |
| Accent | #0F766E on #E5F5F2 |
| Instrument category | #6D28D9 on #F1EAFF |
| Success | #15803D |
| Warning | #B45309 |
| Error | #B91C1C |
| Font | System UI, 14px body; monospace references |
| Spacing | 4, 8, 12, 16, 24, 32px |
| Controls | Minimum 36px height, 6px radius |

56px header. 240px left palette/tree. 320px right inspector. The canvas takes the
remaining area. Bottom event drawer starts collapsed. Panels collapse at narrow
widths. Target 1366x768 and 1920x1080.

Tabs: Substation, IEDs & Signals, Signal Test, Scenarios, Connections.
Persistent actions: Save, Run, Pause Automation, Stop and validation status.
No gradients, decorative dashboards, external fonts or remote runtime assets.
Use SVG equipment, line terminals, a light grid and restrained selection colour.

The user requested a richer color treatment on 2026-09-09. Use a navy header,
blue active navigation/cards and restrained teal/violet library category accents.
Library accents describe equipment categories, never energized or protection state.
Status text and geometry remain authoritative; error/danger uses red consistently.

Signal Test uses the same surface cards and tokens, with an adaptive card grid.
Report selection and full selectable paths precede signal selection in mapping pickers.
CB contacts translate and DS/ES blades rotate over 300 ms after confirmed feedback.
Unknown/invalid states do not resemble confirmed Open; reduced motion disables
interpolation. IED deletion checkboxes are independent of the selected data-model card.
Trip relay uses the shared instrument palette but a distinct coil/contact symbol.
Its text states are Reset, Operated, Awaiting and Invalid; colour is supplementary.
Push Button uses one symbol with momentary-SPC, latching-status and self-resetting
behaviors and no wire terminals; it is pressed by clicking it on the canvas.
Analog Source configuration stays in Equipment Properties and uses shared controls.
Relay panels use a violet-tinted header (name, signal count) and one row per
signal with a value chip: filled marker `1`, hollow `0`, dashed `—` for no value,
plain text otherwise. Rows never overflow the panel; long text ellipsizes.
Properties (ADR-0025): a header card (category tile, friendly type, Ready/Setup
needed checklist, icon actions) then bordered section cards with a tinted title
band and a worded tag (Reads from IED, Sends command, Live, Simulation). While
running, the Live/Operate card comes first. Canvas cards carry a category accent
bar; state pills use dashed (not mapped), dotted (awaiting) or solid (live) borders.
State fills (ADR-0026): two-state objects fill their canvas card with a per-object
colour for the confirmed state (defaults: CB/DS/ES Closed green, Open white; Trip
relay Operated red, Reset white; Push button Pressed/Latched green, Released white).
Intermediate is dashed amber and Invalid dashed red. Text contrast is automatic;
state text and symbol geometry stay authoritative. Category accents stay separate.
