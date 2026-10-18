# Spec: Tools place

Decision: [ADR-0054](../decisions/ADR-0054-tools-place.md).
Change: [0079](../changes/2026/0079-tools-place.md).

## Purpose

One calm place for testing and commissioning helpers. A beginner sees what each tool is for
before opening it, and every tool listed actually works.

## Behaviour contract

- Tools sits after Test in the sidebar (UI-01).
- The list shows one large card per delivered tool: icon, name, one plain line and a
  "What is this?" hint. Wording lives in `frontend/src/copy/tools.ts`.
- Opening a card shows the tool in the same page with an "All tools" button back to the list.
- Planned tools are specified in `docs/specs/proposed/` and are not listed until built.
- GOOSE & Sampled Values traffic is the first tool. Settings → Network links to it.
- An open tool shows one header row (All tools, the tool's name, its own switches) and fills the
  window; only its panes scroll ([tool layout](tool-layout.md), ADR-0064). The GOOSE & SV traffic
  report keeps the scrolling page.

| ID | Rule | Enforced by |
| --- | --- | --- |
| TOOLS-01 | Every tool card has a unique id, a short name, one plain line and a hint. | `studio/tests/frontend/tools.test.ts::every tool card has a name, one plain line and a hint` |
| TOOLS-02 | Every listed tool opens a built view; no card points to unfinished work. | `studio/tests/frontend/tools.test.ts::every listed tool opens a built tool view` |
| TOOLS-03 | GOOSE and Sampled Values traffic is shown only in Tools; Settings links to it. | `studio/tests/frontend/tools.test.ts::GOOSE and Sampled Values traffic is shown only in Tools` |
| TOOLS-04 | An open tool always offers the way back to the list. | `studio/tests/frontend/tools.test.ts::an open tool always offers the way back to the list` |
