# ADR-0121: The lab network is a choice in the network card list

Accepted: 2026-10-26. Owner, on the 0192 design (a separate "Lab network on this PC" card under the
banner): "design is bad, I prefer a drop down list same as [the network card list], with highlight of
loopback and another colour highlight for lab network". Amends the UI part of
[ADR-0120](ADR-0120-this-pc-lab-network.md); the card, service, addresses and clash rules are unchanged.
Change [0193](../changes/2026/0193-lab-network-in-the-card-list.md).

## Decision

1. **One list for "where do my IEDs answer".** "Put my IEDs on a network" opens the same page as
   before; its network card list now starts with a group "On this PC, nothing goes out" holding
   "This PC only (loopback)" and "ScadaBench Lab: each IED at its own SCL address", then "Real network
   cards" (virtual adapters in a lighter text). The ScadaBench Lab card is never listed as an
   ordinary card.
2. **Two colours, never colour alone.** Loopback uses the accent blue, the lab network the teal
   token; each row also has its own icon and a text tag ("Simplest", "No VM · ping works"), and the
   chosen row keeps its colour and icon in the field.
3. **A styled list instead of the native `<select>`.** A native select cannot show icons or tags.
   `NetworkCardPicker` is a button that opens a listbox in the shared popover (placement, close on
   outside click, scroll or resize as `Menu`); arrow keys move, Enter or Space picks, Escape closes.
4. **Picking the lab network** shows the lab panel in place of the address check: one line, the
   folded addresses (edit, check, Ping) and one button (create, repair or use) beside Cancel. Use
   needs no Windows prompt; creating the card asks once, as in ADR-0120.
5. **While the IEDs use the lab network** the banner holds the folded addresses (move one, Ping,
   remove the card) and its button reads "Change network". The separate lab card is removed.

## Compatibility

No route, project field or engine behaviour changes; the frontend only displays engine answers.
Projects, the Python engine and the packaged app are unaffected beyond these screens.
