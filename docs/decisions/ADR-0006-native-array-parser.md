# ADR-0006: Narrow pinned-parser correction

Status: accepted, 2026-09-09; isolated candidate protocol tests pass.

Conformance clarification: the bundled SCL2007B4 XSD does not allow `count` on a
top-level LNodeType DO declaration. This correction improves native CFG parsing;
it does not make that synthetic XML schema-valid. Schema-valid SCL array checks
use SDO arrays. Keep these two kinds of evidence separate.

An independent MMS browse regression demonstrates that the pinned libiec61850
configuration parser rejects top-level DO arrays. Its nested DO branch records
array state, but its top-level branch does not. Its indentation-4 closing branch
also bypasses array-element handling.

Keep the original library source intact. At CMake configuration, require the exact
audited parser SHA-256 and generate a copy carrying the existing copyright/license
plus three narrow substitutions: initialize top-level array state; use the normal
depth-4 closing branch only outside arrays; allow the array-aware branch at depth 4.
Replace only that source in the library targets. The maintained correction is in
`studio/native/cmake/parser_array_fix.cmake`, not an untracked vendor-tree edit.

Any upstream parser change fails configuration until reviewed. Generated copied
vendor source remains an explicit build-output exclusion from the 299-line rule;
the correction itself is maintained code and must follow that rule. Distribution
must retain upstream license and corresponding-source obligations, including this
correction. This does not broaden protocol support or prove nested-array-of-array
behavior beyond independently tested cases.

Test a candidate build in isolation while an old worker is used by the soak.
Do not replace the active worker executable. Remove the strict expected failure
only when the candidate passes; record separately which build each test used.
