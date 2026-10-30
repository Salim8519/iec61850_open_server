# 0203: Faster verify.py, same checks

Owner: "how to make test verifications faster", "make it fast, same effeciency but much faster",
"Kaspersky could be doing harm. The project folder itself is whistled not the temp", "U don't have
to make it insane fast, just whatever worked". Tooling only: no product, protocol or test change.

## Diagnosis (owner's PC, 16 logical cores, before this change)

- `check_file_lengths.py` took 202 s (77 s on a warm second run) of a `verify.py` run. It walked
  about 61,000 files with `rglob("*")`, including `node_modules`, `build`, `.venv`, `native` and
  `dist-protected`, only to discard all but 5,010 of them.
- The other quick checks (architecture, documentation, history, specifications) took 7 s together.
- All seven checks ran one after another. GOOSE `go test`: 59 s when re-run, 4 s cached.
- Go engine suite, every package re-run: 635 s with `-p 2` (101 packages, 967 s of package time).
  Most packages wait on Python, PowerShell, real servers or workers, not on the CPU.
- Go's test cache works when nothing changed and the environment is the same.
- Kaspersky whitelists the project folder but not %TEMP%. There it refused a fresh test executable
  (`internal/runtime/ports`, "Access is denied"), and licensing tests failed more often on files
  left behind (TempDir cleanup "directory is not empty"; a leftover `.TIME_ANCHOR.tmp` in
  `TestScenariosMatchPython`).

## Change

- `check_file_lengths.py`: `os.walk` prunes excluded folder names before descending. The rule is
  unchanged: the same files are checked.
- `verify.py`: the seven checks start at once (thread pool). Results print in the old fixed order
  with the same lines; the exit status is unchanged.
- `engine_tests.py`: `go test -p 8` instead of `-p 2`.
- `build_goose_engine.py`: `test_environment()` for verify's Go steps sets GOTMPDIR, TEMP and TMP
  to the git-ignored `.local/gotmp`, as the passing runs of changes 0157 and 0171 did. The release
  build (`build()`) is unchanged.

## Evidence

- File list: the old and new `maintained_files()` return the identical 5,010 paths (byte-compared
  sorted lists); both report no failures. Time 77–202 s → 6.8 s.
- Go engine suite with every package re-run (`-count=1`), `-p 8`: 123 s (was 635 s with `-p 2`).
- `verify.py` with `GOFLAGS=-count=1` (every Go test re-run), was about 15 min (202 + 7 + 59 +
  635 s one after another): with only GOTMPDIR moved 105 s and 108 s (hostkey race below both
  times); with TEMP and TMP moved too 117 s (all seven PASS) and 148 s (licensing leftover below).
- Normal `verify.py` (Go cache in use), final code: 143 s, all seven checks PASS. Earlier normal
  runs with TEMP in %TEMP%: 157 s and 163 s, `internal/licensing` failing both times.
- `TestScenariosMatchPython` plus `TestRealServerCancelledKeyStopsAtTheNextWeeklyCheckAndStaysCancelled`,
  30 times each, alone: 7/60 failed with TEMP in %TEMP% (5 leftover `.TIME_ANCHOR.tmp`), 1/60
  with TEMP in `.local/gotmp`. That pair took 266 s and 473 s respectively (one run each, so the
  timing difference is not established).

## Known issues and limitations

- Pre-existing race, not caused by this change:
  `internal/licensing/hostkey` `TestAPeerOverTheRateLimitIsToldToWaitAndIsNotAnswered` fails about
  1 run in 4 on this PC even alone with no load (42/200 with %TEMP%, 49/200 with `.local`). The
  server writes "Too many requests" and closes without reading the request; on Windows the close
  can reset the connection and drop the reply. The over-limit peer is still not served (LIC-75
  holds); only the "wait" message can be lost. Fixing it changes licensing server code and is left
  for the owner's decision.
- Pre-existing: the licensing tests above still leave a file behind about 1 run in 60 with TEMP
  in `.local`; not investigated further.
- A cached run still re-links some test binaries (Go retries the result lookup by binary content
  when the action ID misses); `internal/licensing` spends about 45 s there. Not changed.
- `-p 8` keeps the PC busier during a full re-run (about 2 minutes instead of 10.6).
- pytest and frontend tests are not part of `verify.py` and were not timed.
