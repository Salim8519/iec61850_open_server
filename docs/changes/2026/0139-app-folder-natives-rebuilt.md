# 0139: Native workers rebuilt in the app folder (`studio/native/build/Release`)

Date: 2026-10-20. Owner: "close ScadaBench now, rebuild the native workers into the app folder, then
start it again." Before this, the owner's development app still ran the old workers; the speed work
of [0136](0136-speed-lane-n3-native-waits.md) (ADR-0079, ADR-0088, ADR-0089, N3 waits,
`control_mux_v1`) had only been built in `studio/native/build/integrate2` and measured there
([0138](0138-stress-test-python-vs-go.md): operate p50 523 -> 142 ms with idle MMS clients).

## Change

No source change. ScadaBench was not running (nothing on 8765/5173). Rebuilt from master:

- `cmake -S studio/native -B studio/native/build`, then `cmake --build studio/native/build --config
  Release --parallel 4` for `studio_ied`, `studio_live`, `studio_probe`, `studio_control`,
  `studio_gateway`, `studio_master`, `studio_sv_test_publisher`, `studio_license_check`,
  `studio_test_iec104_master`, `studio_test_dnp3_master` (cache unchanged: license enforcement OFF,
  gateway ON). `studio_license_check` and `studio_test_dnp3_master` had no changed inputs and were
  not relinked.
- `studio/tools/build_goose_engine.py` rebuilt `studio_goose.exe` into the same folder.

## Evidence

- Build exit 0; nine programs dated 2026-10-20 11:34-11:35.
- Go tests that need the app-folder workers now run instead of skipping and pass:
  `TestCanvasOperateMatchesProbeOnTheWorker`, `TestCanvasOperateTiming`,
  `TestCanvasOperateRefusals`, `TestOperateTiming`.
- `start-studio.cmd` started the backend on 8765 and Vite on 5173; the built-in browser shows the
  Devices page with the owner's project loaded.

## Limitations

- No IED was started from the UI after the restart (the owner's project addresses were left as they
  are). The packaged EXE is not rebuilt (not requested).
