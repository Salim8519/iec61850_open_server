# 0045: Paint the state part of the symbol, not the whole card

Date: 2026-09-29. Owner feedback on 0044: filling the whole card was crude;
colour should be applied part by part with a nice shape. Amends ADR-0026.

Changes: `EquipmentSymbol` takes an optional `paint` (fill, outline, ink) and
draws the CB body with a contrasting contact, a solid outlined blade for DS/ES
(`Blade`), a painted trip-relay body and push-button cap; `paintFor` in
`stateColors.ts` supplies a darker outline (slate for light colours) and ink;
cards stay neutral with a matching state-pill dot; Appearance previews the
painted symbol per state. Contract unchanged.

Evidence: frontend node tests 43 pass (new `paintFor` test); backend unit +
integration pass; build and `verify.py` pass; running screenshots at 1x and
2x on the isolated copy after real operate commands: CB closed green body,
DS open white outlined blade, ES closed custom amber blade, latched button
green cap, operated trip relay red body.

Limitations: owner visual acceptance pending.
