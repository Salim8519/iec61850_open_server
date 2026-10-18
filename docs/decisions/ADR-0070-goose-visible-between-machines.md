# ADR-0070: GOOSE between machines is visible and timed to its TAL

Accepted: 2026-10-19. Spec: [signal layers](../specs/signal-layers.md) (LAYER-20, 22, 23).
Change: [0102](../changes/2026/0102-goose-visibility-and-timing.md). Amends ADR-0065 and ADR-0068.

**Problem**

The host ↔ VM test ([0101](../changes/2026/0101-goose-host-vm-0.1.12.md)) found that GOOSE worked
on the wire but the app hid or misdescribed it. Each finding was reproduced with public sample files:

- An outside sender's dot started 230 drawing units left of the leftmost equipment and was drawn
  in drawing units. On a large drawing (zoom 0.2–0.3) it was 2–3 px wide and off screen, with a
  3.3 px label. At normal zoom it was a single 10 px dot with no line.
- With MinTime 4 ms the worker retransmitted every ~15 ms (Windows' default timer). That is longer
  than the 12 ms TAL those frames announce: 7 of 21 frames were late.
- An IED added after the lab setup has no lab address. It answers on loopback, but the address card
  said "On the lab network adapter" and the top bar counted it as reachable.
- The GOOSE partners table showed "Outside project" for a sender whose frames were arriving.
- Equipment names were cut ("PUSH_BUTTO…").

**Decision**

1. An outside sender's dot starts a fixed screen distance (220 px × tag scale) left of and above
   the receiver tag and draws the same fading line as in-project links. Dot radius, line width
   and label size use the tag scale, so they keep their screen size when zoomed out. For ~2 s the
   receiver tag shows one line, "← PUB · change", in the style of the existing "operated" line.
   Nothing remains afterwards, and idle repeats still draw nothing.
2. The IED worker raises the Windows timer resolution to 1 ms for its lifetime
   (`timeBeginPeriod(1)`, released on exit). libiec61850's 1 ms server loop then retransmits at
   MinTime.
3. One rule, `onLabNetwork(ied, network)`, decides "on the lab adapter" for the top bar, the address
   card and the role line. In lab mode the top bar says "1 of 2 IEDs reachable from …".
4. A link from an outside sender that the listener reports as receiving (or with a problem) shows
   that live state, with "not in this project" in the hint. A silent one keeps "Outside project".
5. Equipment names and state pills wrap instead of being cut.

**Compatibility**

- No project format, API or wire change.
- The timer request affects only the worker process and is undone by Windows when the process exits.
- Tags, layer switches and the clean SLD rule (ADR-0065, change 0097) are unchanged.
- The SLD gains no permanent object.
