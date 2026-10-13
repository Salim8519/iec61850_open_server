# ADR-0007: Recover engineering state, never resume a live run

Status: accepted, 2026-09-09.

On launcher startup, acquire an OS-held lock for the chosen data directory before
reading the autosave or creating services. A second launcher using that directory
must stop with an actionable error. The OS releases the lock on normal or abrupt
exit; the harmless lock file may remain and is not evidence of a live process.

Restore a valid autosave through the normal SCL/project validators without writing
it back during startup. Preserve project/IED identities, diagram bindings, saved
sources and scenarios. Runtime always starts stopped: no listener, automatic
source, enabled sequence or previously active command resumes automatically.

If an autosave is unreadable, preserve an exact recovery copy before allowing new
work. Show the diagnostic and backup path in the browser. If preservation fails,
stop startup rather than exposing a writable blank project that could erase the
only copy. A separate restored-state notice explains that all IEDs remain stopped.

Runtime measurements are not a resumable protection state. Re-imported source
initial values remain authoritative. Recovery is not a substitute for project
exports/backups, crash-durable disk guarantees or release acceptance testing.
