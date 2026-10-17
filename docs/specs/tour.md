# Spec: Guided tour

Decision: [ADR-0047](../decisions/ADR-0047-scadabench-brand-website-tour.md).
Change: [0070](../changes/2026/0070-scadabench-brand-website-tour.md).

## Purpose

A first-time user learns the five places (Devices, SLD, Test, Tools, Settings) and where Run is in about
one minute, without reading a manual, and can reopen the tour at any time.

## Behaviour contract

- The tour opens automatically when the stored "seen" version is older than the current tour version,
  in development and packaged builds alike.
- Each step dims the app, rings one real control (`data-tour` target) and explains it in one or two
  plain sentences from `copy/tour.ts`; it switches to the place where that control lives.
- Esc or "Skip tour" closes it; finishing or skipping stores the version as seen in the data folder.
- Help in the sidebar reopens the tour and links to the video tutorials.

| ID | Rule | Enforced by |
| --- | --- | --- |
| TOUR-01 | Every tour step points at a control that exists in the app. | `studio/tests/frontend/tour.test.ts::every tour step points at an element that exists in the app` |
| TOUR-02 | Tour wording is short, plain and lives in the copy module. | `studio/tests/frontend/tour.test.ts::tour wording is plain and complete` |
| TOUR-03 | The tour opens from stored preferences and can be reopened from Help. | `studio/tests/frontend/tour.test.ts::the tour opens from preferences and can always be reopened from Help` |
| TOUR-04 | "Seen" is stored by the backend and survives restarts. | `studio/tests/integration/test_preferences_api.py::test_tour_is_due_on_first_start_and_remembered_after_finishing` |
| TOUR-05 | Only known, correctly typed preferences are accepted. | `studio/tests/integration/test_preferences_api.py::test_unknown_or_wrongly_typed_preferences_are_refused` |
