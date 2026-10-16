# Spec: Machine licensing and protected release

Decision: [ADR-0046](../decisions/ADR-0046-licensing-and-protected-release.md).
Change: [0069](../changes/2026/0069-licensing.md).

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
| LIC-17 | Closing the packaged app window stops ScadaBench; a browser hand-off does not (see [app window](app-window.md)). | `studio/tests/unit/test_app_window.py::test_closing_the_window_stops_the_app`, `studio/tests/unit/test_app_window.py::test_handoff_to_another_browser_keeps_the_app_running` |
| LIC-19 | A packaged app refuses to run with a swapped/patched native worker or an attached debugger; development is unaffected. | `studio/tests/unit/test_licensing.py::test_swapped_native_worker_or_debugger_blocks_packaged_mode`, `studio/tests/unit/test_protected_packaging.py::test_protected_copy_is_sealed_and_repository_source_untouched` |
| LIC-20 | Owner removal moves license files aside (never deletes) and can free the key and reset the trial. | `studio/tests/unit/test_license_admin.py::test_remove_moves_license_aside_and_frees_key` |
| LIC-18 | A new computer can always start the trial; only an expired trial disables the button. | `studio/tests/frontend/license.test.ts::a new computer can start the trial; only an expired trial disables it` |
