# 0030: Stable verification of deferred SBO compatibility

Date: 2026-09-16.

The broad regression run exposed a timing race in the independent probe. CONCEN
compatibility deliberately applies feedback on the native server task thread after
the selection callback returns. `sbo-compat` read state immediately after the
successful selection, so under concurrent build/test load it could observe the old
value and fail even though the queued action completed normally. The same test
passed immediately when rerun alone.

The probe now polls the status for up to one second in 20 ms intervals only for the
compatibility action, stopping at the first change. Standard enhanced-SBO checks
still require no movement after selection and remain unchanged. This modifies test
observation timing, not worker protocol behavior or the project format.

Expected result: the independent probe consistently observes the documented
deferred compatibility result under load without hiding a missing transition. A
transition that never occurs still fails after the bounded deadline.

Evidence: the normal Release probe rebuilt successfully, the focused compatibility
case passed 10 consecutive runs, and the final complete Python/native suite passed
126 tests.
