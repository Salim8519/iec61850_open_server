# ADR-0019: Shared tab identity and stale-write rejection

Status: accepted, 2026-09-16.

Browser tabs are views of one backend-owned project, not independent engineering
sessions. Each backend process assigns a visible session identifier and increments
an in-memory project revision after every saved mutation. Mutations publish a
`project_changed` event so other tabs reload the authoritative snapshot.

Diagram writes include the revision that the tab edited and a per-tab identifier.
The backend returns conflict instead of accepting a stale revision. The originating
tab advances its revision without replacing newer unsaved local geometry; other
tabs cancel pending stale saves, reset local undo history and synchronize. This
prevents an old browser tab from silently restoring an earlier diagram.

Session identity and revision are runtime coordination metadata and are not placed
inside the engineering archive. A backend restart creates a new session identifier
and restores the autosave with all IEDs stopped. Native capability checks remain
independent: a backend cannot run an older worker even when browser tabs agree.
The frontend identifies its development origin on API calls. A tab served by an
old Vite port is redirected to the backend's active frontend instead of presenting
another apparently valid engine view.
