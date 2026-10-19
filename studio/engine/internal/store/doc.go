// Package store owns every disk artefact of the engineering project (workstream B, ADR-0074
// architecture section 4), split by responsibility into sub-packages:
//
//   - archive: the .subsim ZIP (project.json) reader and writer, byte-compatible with
//     studio_app.projects.archive, with a cached-segment deflate encoder so unchanged SCL
//     text is never compressed twice.
//   - autosave: autosave.subsim written atomically, the coalescing background snapshot
//     writer and the synced change journal (autosave.journal) with Fold.
//   - recovery: recovery/*.subsim backups and startup recovery (startup_recovery.py,
//     background_recovery.py, migration_backup.py, bulk_removal.py backup part).
//   - jsonfile: Python json.dumps(indent=2) text files in Windows text mode (CRLF).
//   - settings: preferences.json file store and tool-settings.json (tool_memory.py).
//   - modelcache: <data>/cache/models derived-model cache.
//
// testdata/crosscheck.py checks the formats against the frozen Python backend.
package store
