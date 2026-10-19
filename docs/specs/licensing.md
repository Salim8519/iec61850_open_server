# Spec: Machine licensing and protected release

Decision: [ADR-0046](../decisions/ADR-0046-licensing-and-protected-release.md).
Change: [0069](../changes/2026/0069-licensing.md); trial hardening:
[0106](../changes/2026/0106-trial-hardening.md) (ADR-0046 addendum 2026-10-19).

## Purpose

Packaged Studio runs only on a computer with a valid license: a one-time 7-day trial or a
lifetime key locked to one PC. Development runs exactly as before, with no license.

## Behaviour contract

- **Modes:** development (source, `start-studio.cmd`) never checks a license and never
  touches the license folder. Packaged builds check in the backend and, independently, in
  the native workers of release builds.
- **Certificate:** `license.cert` is `{payload, signature}`; Ed25519 over the exact payload
  string; payload holds `email, expires_at, hwid, issued_at, kid, license_id, max_machines,
  type (trial|lifetime), version (1)`.
- **Valid when:** signature matches an accepted key, `version == 1`, `hwid` equals this
  computer, `now + 300 >= max(issued_at, latest observed time)`, and for trials
  `now < expires_at`.
- **Trial request:** the client also sends salted hashes of MachineGuid, system volume
  serial and firmware UUID (placeholders sent empty). Every usable part (at least two)
  matching an ended trial refuses; 2 of 3, or more than 3 new trials from one IP in 30
  days, only asks for a confirmed email (6-digit code; one trial per address; a mail
  outage waives the code). A running match continues the same trial. IP never refuses.
- **Trials only:** answers carry a signed `server_time`. The anchor, the newest genuine
  stamp and a missing-anchor mark are kept in a dot-file, `HKCU\Software\ScadaBench` and
  `license_state.json`. Rollback = more than 48 h behind the latest copy or the signed
  time; signed time past `expires_at` ends the trial. Server outages never stop a trial;
  only when every anchor copy and the stamp are gone is one online check asked for within
  3 days. Lifetime licenses keep the 300 s rule and stay offline-friendly.
- **Gate:** while unlicensed, packaged engineering API calls answer 402; the browser shows
  the activation screen (trial, key, buy, offline file). Saved projects are never touched.
- **Wording:** every state is words plus an icon; the reason says what to do next.

| ID | Rule | Enforced by |
| --- | --- | --- |
| LIC-01 | Development never asks for a license and never reads the license folder. | `studio/tests/integration/test_license_api.py::test_development_backend_never_asks_for_a_license`, `studio/tests/unit/test_licensing.py::test_development_mode_is_always_licensed_without_reading_files` |
| LIC-02 | Packaged builds answer 402 on engineering API calls until licensed; session and license endpoints stay reachable. | `studio/tests/integration/test_license_api.py::test_packaged_backend_blocks_engineering_api_until_licensed` |
| LIC-03 | Certificates signed by the Go server verify identically in Python and in the native guard. | `studio/tests/unit/test_licensing.py::test_go_signed_certificate_verifies_in_python`, `studio/tests/unit/test_native_license.py::test_native_accepts_go_and_python_signed_certificates` |
| LIC-04 | Any change to payload or signature, or a foreign key, is rejected. | `studio/tests/unit/test_licensing.py::test_tampered_or_foreign_signature_is_rejected`, `studio/tests/unit/test_native_license.py::test_native_rejects_tamper_wrong_machine_expiry_and_rollback` |
| LIC-05 | The machine ID formula is identical in Python and native code. | `studio/tests/unit/test_licensing.py::test_hwid_composition_is_stable_and_matches_native_format`, `studio/tests/unit/test_native_license.py::test_native_machine_id_matches_python` |
| LIC-06 | A trial lasts 7 days, counts down, then stops; one trial per machine ever. | `studio/tests/unit/test_licensing.py::test_seven_day_trial_counts_down_then_expires`, `studio/tests/integration/test_license_server_e2e.py::test_trial_is_granted_once_per_machine` |
| LIC-07 | Setting the clock back before the last observed time or issue time is refused. | `studio/tests/unit/test_licensing.py::test_clock_rollback_is_detected_from_anchor_and_issue_time`, `studio/tests/unit/test_licensing.py::test_clock_rollback_uses_latest_run_log_time` |
| LIC-08 | A lifetime key works on one PC; another PC must move it, which the old PC learns online. | `studio/tests/integration/test_license_server_e2e.py::test_lifetime_key_locks_to_one_pc_and_can_move`, `studio/tests/unit/test_licensing.py::test_certificate_for_another_machine_is_rejected` |
| LIC-09 | A refunded or revoked key stops working at the next online check; the file is kept aside. | `studio/tests/unit/test_licensing.py::test_online_check_removes_refunded_license_but_keeps_file` |
| LIC-10 | A license file for this machine activates offline; one for another machine is refused. | `studio/tests/integration/test_license_api.py::test_offline_activation_unlocks_the_api`, `studio/tests/unit/test_licensing.py::test_trial_activation_and_offline_install_save_certificate` |
| LIC-11 | Native enforcement exists only in builds configured with STUDIO_ENFORCE_LICENSE; such workers refuse MMS/GOOSE/SV without a license. | `studio/tests/unit/test_protected_packaging.py::test_native_enforcement_is_off_unless_release_build_asks`, `studio/tests/unit/test_native_license.py::test_development_build_does_not_enforce`, `studio/tests/unit/test_native_enforcement.py::test_worker_refuses_to_serve_without_license` |
| LIC-12 | The protected build refuses to run without a release key and ships no studio_app source, bytecode or loose UI files. | `studio/tests/unit/test_protected_packaging.py::test_protected_build_refuses_without_release_key_and_dry_run_lists_steps`, `studio/tests/unit/test_protected_packaging.py::test_leak_check_finds_source_bytecode_and_loose_ui` |
| LIC-13 | The UI shows the activation screen on 402 and describes every state in words. | `studio/tests/frontend/license.test.ts::the app is wrapped by the license boundary and reacts to 402`, `studio/tests/frontend/license.test.ts::license state uses words and tone, never colour alone` |
| LIC-14 | The license server address must be https (loopback http only for tests). | `studio/tests/unit/test_licensing.py::test_server_url_must_be_https` |
| LIC-15 | The packed UI serves byte-identical files to the folder build. | `studio/tests/unit/test_protected_packaging.py::test_packed_frontend_serves_identical_bytes` |
| LIC-16 | Installed copies keep data per user; a data folder beside the exe still wins. | `studio/tests/unit/test_protected_packaging.py::test_installed_copy_keeps_data_per_user_but_portable_copy_stays_beside_exe` |
| LIC-17 | Closing the packaged app window stops ScadaBench; a browser hand-off does not (see [app window](app-window.md)). | `studio/tests/unit/test_app_window.py::test_closing_the_window_stops_the_app`, `studio/tests/unit/test_app_window.py::test_handoff_to_another_browser_keeps_the_app_running_until_it_closes`, `studio/tests/unit/test_app_window.py::test_browser_relaunch_right_after_start_is_not_a_close` |
| LIC-19 | A packaged app refuses to run with a swapped/patched native worker or an attached debugger; development is unaffected. | `studio/tests/unit/test_licensing.py::test_swapped_native_worker_or_debugger_blocks_packaged_mode`, `studio/tests/unit/test_protected_packaging.py::test_protected_copy_is_sealed_and_repository_source_untouched` |
| LIC-20 | Owner removal moves license files aside (never deletes) and can free the key and reset the trial. | `studio/tests/unit/test_license_admin.py::test_remove_moves_license_aside_and_frees_key` |
| LIC-21 | The license client verifies the server's HTTPS certificate with the Windows certificate check (as browsers do), keeping host-name checks and required verification. | `studio/tests/unit/test_licensing.py::test_license_client_verifies_https_with_the_windows_certificate_check` |
| LIC-18 | A new computer can always start the trial; only an expired trial disables the button. | `studio/tests/frontend/license.test.ts::a new computer can start the trial; only an expired trial disables it` |
| LIC-22 | Every usable hardware part (at least two) matching an ended trial refuses a new trial; one part alone does nothing; a running match continues the same trial. | `license-server/internal/store/trials_test.go::TestCheaterWhoseEveryPartMatchesAnEndedTrialIsRefused`, `license-server/internal/store/trials_test.go::TestOneMatchingPartAloneNeverAsksAnything`, `license-server/internal/store/trials_test.go::TestSameHardwareWithNewMachineIDKeepsTheRunningTrial`, `studio/tests/integration/test_license_server_e2e.py::test_trial_parts_and_signed_time_with_the_real_server` |
| LIC-23 | Placeholder firmware UUIDs and junk volume serials are never sent and never compared; part hashes equal the Go server's. | `license-server/internal/store/trials_test.go::TestPlaceholderFirmwareUUIDsAreNeverCompared`, `studio/tests/unit/test_trial_parts.py::test_placeholder_firmware_uuids_are_never_sent`, `studio/tests/unit/test_trial_parts.py::test_part_hashes_match_the_go_server_byte_for_byte` |
| LIC-24 | Two of three parts, or a busy network, only asks for a confirmed email, one trial per normalised address; IP never refuses; old clients keep the machine-ID rule. | `license-server/internal/store/trials_test.go::TestTwoOfThreeAsksForAnEmailOncePerAddress`, `license-server/internal/store/trials_test.go::TestOfficeOfFourOnOneIPOnlyTypesAnEmailForTheFourth`, `license-server/internal/store/trials_test.go::TestOldClientsWithoutPartsKeepTheMachineIDRule`, `license-server/internal/trialemail/trialemail_test.go::TestOneMailboxAlwaysNormalisesToTheSameAddress` |
| LIC-25 | The email code is mailed, single-use, limited to five tries and bound to the address; throwaway domains are refused; a mail outage never blocks the trial. | `license-server/internal/api/trial_test.go::TestSuspiciousTrialConfirmsAnEmailCodeOncePerAddress`, `license-server/internal/store/trials_test.go::TestEmailCodeIsSingleUseLimitedAndBoundToTheAddress`, `license-server/internal/api/trial_test.go::TestMailOutageNeverCountsAgainstTheUserAndLogsHoldNoSecrets`, `studio/tests/frontend/license.test.ts::a trial the server wants confirmed asks for an email, and a mail outage never blocks` |
| LIC-26 | Refusals and extra steps are plain English with a support link; the server logs each reason without full machine IDs, part hashes, keys or emails. | `studio/tests/unit/test_trial_parts.py::test_trial_refusals_and_steps_are_plain_english_with_a_support_link`, `license-server/internal/api/trial_test.go::TestMailOutageNeverCountsAgainstTheUserAndLogsHoldNoSecrets` |
| LIC-27 | Licensing answers carry a signed server time kept in three places; a stamp and a certificate can never stand in for each other. | `license-server/internal/api/trial_test.go::TestEveryLicensingAnswerCarriesSignedServerTime`, `studio/tests/unit/test_trial_clock.py::test_trial_start_keeps_the_signed_time_in_all_three_places`, `studio/tests/unit/test_trial_clock.py::test_stamps_and_certificates_can_never_stand_in_for_each_other` |
| LIC-28 | Deleting the anchor file (or every anchor copy, keeping the signed time) and setting a trial's clock back more than 48 h is caught; smaller drift is fine. | `studio/tests/unit/test_trial_clock.py::test_cheater_deleting_the_anchor_file_and_setting_the_clock_back_is_caught`, `studio/tests/unit/test_trial_clock.py::test_signed_time_catches_a_rollback_after_every_anchor_copy_is_deleted`, `studio/tests/unit/test_trial_clock.py::test_small_clock_drift_never_stops_a_trial` |
| LIC-29 | A server outage, an older-build trial or blocked writes never stop a trial; only a fully missing anchor asks one online check within 3 days, with "Check online now". | `studio/tests/unit/test_trial_clock.py::test_server_down_never_stops_a_trial`, `studio/tests/unit/test_trial_clock.py::test_trial_from_an_older_build_is_never_asked_anything`, `studio/tests/unit/test_trial_clock.py::test_blocked_writes_everywhere_never_stop_a_trial`, `studio/tests/unit/test_trial_clock.py::test_missing_anchor_asks_one_online_check_within_three_days`, `studio/tests/frontend/license.test.ts::a trial that could not confirm the date online offers "Check online now"` |
| LIC-30 | Lifetime licenses keep the 300 s rollback rule, never need the daily check and stay valid offline. | `studio/tests/unit/test_trial_clock.py::test_lifetime_keeps_its_original_rollback_rule`, `studio/tests/unit/test_trial_clock.py::test_lifetime_license_never_needs_the_daily_check`, `studio/tests/unit/test_trial_clock_native.py::test_lifetime_offline_for_years_is_fine_in_both` |
| LIC-31 | The native guard reaches the same trial-clock verdicts as Python on the same files. | `studio/tests/unit/test_trial_clock_native.py::test_small_drift_is_allowed_and_big_rollback_is_not_in_either`, `studio/tests/unit/test_trial_clock_native.py::test_missing_anchor_asks_one_online_check_in_both`, `studio/tests/unit/test_trial_clock_native.py::test_forged_stamp_is_ignored_by_both`, `studio/tests/unit/test_native_enforcement.py::test_worker_applies_the_trial_online_rule` |
| LIC-32 | Owner overrides: reset a trial (this PC's parts or a customer's Machine ID), a network's count, and allow an email again. | `studio/tests/unit/test_license_admin.py::test_owner_trial_overrides_send_this_pcs_parts_or_a_customer_machine_id`, `license-server/internal/store/trials_test.go::TestOwnerOverridesClearHardwareAndAddresses` |
