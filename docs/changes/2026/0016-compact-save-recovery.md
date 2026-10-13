# 0016: Live backend restart and compact project recovery

Date: 2026-09-09.

The stale Python backend was replaced with the promoted portable build after
validating and preserving the user's autosave. The updated app subsequently saved
a format-1 project exceeding its own unpacked reopen limit. Original and current
archives were preserved separately; no older project was restored over new work.

Implemented format 2 in a dedicated project document module, with checksum-verified
deduplicated source XML and regenerated runtime tables. Added a bounded legacy
recovery path and pre-write size validation; see
[ADR-0008](../../decisions/ADR-0008-compact-projects.md).

The read-only `studio/tools/check_project.py` validates projects, compares engineering
state through compact save/reopen, and optionally creates a hash-verified exact
backup. The actual project retained its ID, 15 IEDs and 3 diagram equipment items.
Its archive decreased from 13,169,614 to 1,743,969 bytes without engineering changes.
The latest original bytes were backed up before the guarded restart.

The live app now uses the fixed source launcher on port 8765 and the same studio/data
directory. No IED workers were active at restart. The portable folder was rebuilt
successfully (build log: `.local/portable-build-compact.log`). It contains the fix;
the currently running source launcher was left undisturbed after verification.

Verification: all 80 tests passed in 17.53 seconds, including frozen startup,
exclusive data-directory locking, deletion/recovery, edited SCL export, native
MMS reads, controls and reports. File lengths, architecture and documentation
checks passed. Tests cover compact deduplication, corrupt references/checksums,
legacy recovery bounds and rejection of oversized saves before writing.
Browser visual acceptance and clean-machine testing remain outstanding.
