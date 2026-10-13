# 0013: Top-level DO array parser correction

Date: 2026-09-09. Status: candidate verified; normal-build promotion pending.

The strict expected failure from 0012 is now a passing regression against an
isolated candidate worker. The correction initializes top-level array state and
routes depth-4 array closing braces through the array-aware branch. Upstream
source stays unchanged; CMake generates a narrowly corrected copy only after
checking the audited hash. See [decision](../../decisions/ADR-0006-native-array-parser.md).

Evidence:

- Candidate build: `studio/native/build/array-fix/Release` (worker and probe).
- Four MMS browse cases pass: ordinary model, explicit `ldName`, nested DO arrays,
  top-level DO arrays with a following scalar sibling.
- Array tests write three distinct element values and verify each with independent
  MMS reads. Dataset member order and scalar/array leaf discovery also pass.
- All protocol tests now support `STUDIO_NATIVE_BIN`. Full suite with that set to
  the candidate: 70 passed in 17.14 seconds, no expected failures. This includes
  native controls, reports, atomic samples, scenarios and CSV playback against the
  candidate. Packaging tests still inspect the earlier frozen preview, not a new
  portable build containing this fix.
- Candidate worker SHA-256:
  `2ebdf5c27a54c1402ca7e1f5220fd6b768fd2edeb45283e51d8a1d8fea77cb4b`.
- Candidate probe SHA-256:
  `f8d170ab8da7c22b06a74a58457cc62a9b489b67038d9d3390d167137f1bcecc`.
- Original parser remains:
  `79c2ccb95dd0d8bc09690acffbb50cbce16a6b4bbff07c5414e1a5b941c0409c`.
- Normal worker remains:
  `02a0ce82173fccaa3c785249e6de1717c3f8942377a892050a59656786680319`.

Do not replace the normal worker until the old soak has ended and stopped its
children. Rebuild/promote, rerun normal-path tests and rebuild the portable preview
afterward. Candidate tests do not prove array-of-array handling, per-index SCL
overrides, clean-machine startup or completion of the full release gates.
