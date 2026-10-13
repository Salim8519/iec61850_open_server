# 0012: Independent MMS model and dataset discovery

Date: 2026-09-09. Status: implemented and tested; top-level array regression open.

The native client now accepts `studio_probe IP PORT browse`. It receives no SCL,
CFG or expected references. It discovers logical devices/nodes, object names,
datasets, buffered/unbuffered report names, ordered dataset members and deletable
flags. It retrieves MMS type descriptions and expands structures/arrays into
typed leaf references. Depth/array/leaf limits return errors, never silent truncation.

Only the probe target was rebuilt; workers in the existing soak were not replaced.
The probe and worker use the same pinned library, in independent processes over
TCP. This is wire evidence, not third-party-client certification.

Evidence and corrections:

- Initial name-list-only audit showed 2,808 apparently missing references in the
  main customer IED. Type-description expansion accounts for all of them.
- Read-only audit of `440KV_Y210_BCU-1.CID` now discovers all 54,201 imported
  scalar references across its five IEDs, with zero missing references and native
  read success for each. Main IED: 36,021 imported attributes, 36,639 observed
  native leaves including stack-generated control fields. Four peers: 4,545 each.
- Protocol tests compare all imported references against discovered leaves,
  including `ldName` and nested arrays, and verify ordered dataset membership
  and buffered/unbuffered report names.
- The nested-array test exposed incorrect FCDA index placement. The compiler now
  emits array-object/index/component separately; native membership returns the
  expected `samples(0).mag.f` rather than `samples.mag.f(0)`.
- A strict expected-failure test retains a newly demonstrated pinned-parser bug:
  top-level DO arrays are rejected. The parser sets array state for nested DOs
  but not for top-level DOs, and its indentation-4 closing branch needs review.
  This remains a release blocker, not supported behavior or a passing check.
- Full regression: 69 passed, one strict expected failure, in 16.65 seconds.
  File length, architecture and documentation checks pass.

The existing hour-long soak is still a separate older-runtime test. Browser,
clean-machine, LAN cleanup and full load/report-loss acceptance remain unfinished.
