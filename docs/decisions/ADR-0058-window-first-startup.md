# ADR-0058: Open the window first, restore the last project in the background

Accepted: 2026-10-17. Owner report: the installed app "really opens slow"; users
installing it "don't have time".

**Problem**

`create_app` restored the autosave synchronously, so the server and window waited for
it. On the owner's station (9 IEDs, 14.8 MB SCD) the window appeared after 29 s.
Change 0084 removed the repeated imports, which left about 6 s of real model building
that still delayed the window.

**Decision**

- `projects/background_recovery.py` runs `recover_startup` on a daemon thread that
  `create_app` starts after every service exists. Recovery still never starts IEDs.
- `api/startup_gate.py` is a `before_request` hook registered **after**
  `security.protect_request`, so loopback, session and license (402) checks always
  answer first. Every other `/api/` request waits for recovery to finish (at most
  300 s, then 503 `project_opening`). Session, license and preference calls never wait.
- A recovery that stopped to protect the autosave (formerly a startup failure) makes
  every waiting request answer 503 `startup_recovery_failed` with the reason. No
  request can modify or overwrite the autosave until ScadaBench restarts.
- `GET /project` reports `startup_recovery` from the background result. The
  frontend shows "Opening your last project…" while the project is still loading.
- `validate_schema` accepts the already parsed root, so the importer no longer parses
  each document twice.

**Compatibility**

- Development and packaged builds behave identically; nothing is mode-specific.
- API responses are unchanged once recovery finishes. Callers that read
  `app.extensions` directly must first call `app.extensions["startup"].wait()`; the
  extension key `startup_recovery` was replaced by `startup`.
- Licensing order (ADR-0050): the `--dev` refusal and `_announce_signed_module()`
  still run before the app exists, and no worker starts during recovery.
