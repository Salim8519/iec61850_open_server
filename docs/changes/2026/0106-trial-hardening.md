# 0106: Free-trial hardening, "smart, not soft"

Date: 2026-10-19. Decision: [ADR-0046](../../decisions/ADR-0046-licensing-and-protected-release.md)
(addendum 2026-10-19). Spec: [licensing](../../specs/licensing.md) LIC-22 to LIC-32.

## Why

A read-only security review found two cheap loopholes. A new trial was granted whenever
the combined machine ID changed (one identifier changed, or a VM was cloned). Deleting
`.time_anchor` and setting the clock back stretched a running trial. The owner wants
cheaters stopped and honest users never blocked by mistake ("we should not make issue for
good people"). Only clear proof blocks. A suspicion asks for one extra step, and errors
or outages never count against a user. Paid lifetime licenses stay exactly as they were.

## What changed

- **Server** (`license-server`): `store/trials.go`. A trial request may carry salted part
  hashes. If every usable part (at least two) matches an ended trial, the trial is refused.
  Two of three, or more than 3 new trials from one IP in 30 days, means suspicious: the
  trial needs a confirmed email. A running match continues that trial. `store/parts.go`
  drops placeholder UUIDs and junk serials. `store/trial_signals.go` holds the IP window,
  the one-trial-per-address record and email codes. `store/trial_admin.go` holds the
  owner resets. `trialemail/` normalises addresses (lowercase, no `+tag`, Gmail dots), stores
  them only as hashes, makes the HMAC-bound codes and keeps the throwaway-domain list.
  `api/trial.go` and `api/trial_email.go` add `POST /api/v1/trial/email-code` (Mailgun;
  when mail is down, no code is needed and the outage is logged). They log every
  refusal or step with its reason. `withTime` adds a signed `server_time` to trial,
  activate, transfer and verify answers. Admin: `reset-trial` (also a refused
  machine's hardware), `reset-trial-parts`, `reset-trial-ip`, `allow-email`; `list` shows
  busy networks. Config: `trial_ip_email_after` (3) and `trial_ip_window_days` (30).
- **Python** (`studio_app`): `domain/trial_policy.py` (pure trial rules and wording),
  `licensing/trial_parts.py`, `server_time.py`, `clock_store.py` (dot-file,
  `HKCU\Software\ScadaBench`, `license_state.json`; failures ignored), `trial_clock.py`, and
  `online_check.py` (the weekly lifetime check moved here unchanged, plus a quiet daily trial
  check and "Check online now"). `time_anchor.py` reads and repairs all three copies.
  `service.py` evaluates trials with a 48 h rollback tolerance (lifetime keeps the original
  300 s path) and offers `request_trial_code`. `client.py` sends parts and the email/code
  and shows the plain-English wording. `license_policy.py` takes an optional tolerance.
- **API/UI**: `POST /api/v1/license/trial` takes `email`/`code`;
  `POST /api/v1/license/trial/email-code`; `POST /api/v1/license/check-online`.
  `TrialStart.tsx` turns the trial button into email → code → "Start free trial" only when
  the server asks. `app_factory.py` passes the real registry.
- **Native** (`studio/native/common`): `signed_bundle.cpp` (signature check shared by
  certificates and stamps, split out of `license_guard.cpp`) and `license_clock.cpp` (same
  three places and rules). `verify_certificate_text` takes an optional tolerance. Test
  tool: `verify-clock`.
- **Owner tool**: `license_admin.py reset-trial` sends this PC's parts (`--machine-id` for a
  customer's PC), plus `reset-trial-ip --ip` and `allow-email --email`. `remove` also moves the
  new files and registry copies aside (`registry.json`).

## Evidence

- `go test ./...` (license-server): all packages ok. New store tests cover: cheater with
  every part matching refused; 2 of 3 asks for an email, one trial per address; one part
  does nothing; running trial continues; placeholder UUIDs never compared; office of 4 on
  one IP only types an email for the fourth; old clients keep the machine-ID rule; email
  code single-use, 5 tries, bound to the address, waived on mail outage, expires; owner
  overrides. API tests cover: signed time on every answer; mailed code flow with wrong/right
  code, reused and throwaway addresses, allow-email; mail outage never blocks; log holds no
  machine ID, address or domain. `go test ./api` once failed only in temp-folder cleanup
  of two PayPal tests (Windows file lock) and passed on 3 reruns.
- Python, full unit + integration: 477 passed, 2 skipped. The one failure,
  `test_gateway_refuses_to_listen_without_license`, needs the gateway worker, which this
  worktree cannot build because the gateway dependencies are missing. New: `test_trial_clock.py` (16),
  `test_trial_parts.py` (19), `test_trial_clock_native.py` (8, the native tool on the same
  files), e2e against the real Go server: parts, a running match continuing, busy network →
  email step → mail outage → trial; reused address refused.
- Native: dev build and enforced build (test key) compile; enforced
  `test_native_enforcement`: 7 passed, 1 skipped (gateway not built). New
  `test_worker_applies_the_trial_online_rule` passes. Enforced `test_native_license` +
  parity: 14 passed.
- Frontend: `license.test.ts` 7 passed; `tsc --noEmit` clean. In the browser at 1366×768
  with the license sandbox (real Go server, `LICENSE_TRIAL_IP_EMAIL_AFTER=1`): first trial
  started directly. After an owner reset, the next trial showed the email step (text not
  clipped). With mail unavailable the trial started at once. The server log showed
  `needs_email`/`busy_ip`, `trial_email_unsent` and no address.
- `verify.py`: PASS.

## Limitations

- **Deploy the server first.** An older server has no email step and sends no signed time.
  Trials still work; rollback detection then relies on the local anchors only.
- The emailed-code screen was not seen in the browser: the sandbox has no Mailgun, so
  only the email and mail-outage paths were walked. The code path is covered by Go and
  frontend tests.
- Cheaters with a new real mailbox for each trial still get a trial each time after the
  first suspicious one; with the mail service down, any unique address works.
- Un-sysprepped cloned VMs whose firmware UUID is a placeholder share MachineGuid and disk
  serial, so they count as the same hardware and are refused once the first trial ended.
  The owner can reset them.
- An offline PC whose owner deletes every copy (file, registry, state, run log) and resets the
  clock each time can still stretch a trial in 3-day steps. That falls under the accepted
  "offline PCs" limit, and binary patching stays out of scope.
- Trials now allow up to 48 h of clock drift (was 300 s), so a cheater gains at most 2 days.
- No EXE built and nothing deployed. The real server keeps the old rules until redeployed.
