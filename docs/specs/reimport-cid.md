# Spec: Re-import an IED's CID

Decision: [ADR-0148](../decisions/ADR-0148-reimport-cid.md). Owner request (2026-11-23): "add re import
cid option, so same cid gets updated without deleting equipment, or warns user what will be lost …
want as everything as possible to stay the same".

## Behaviour contract

- Devices › Files › Re-import CID… takes a newer file of the same IED (same IED name; an SCD that
  holds it is fine). Another IED's file is refused in words and names the IEDs the file has.
- The IED keeps its id, its settings (role, simulation and monitoring addresses, Allow commands,
  network card, GOOSE publishing, test mode, SCD deadbands) and its place in the project.
- Every link to the IED (diagram positions, commands, relay statuses, measurements, point panels,
  tap changers, GOOSE links, signal sources, test panel, scenario steps) stays when its reference
  is in the new file with the same type. The others are listed with where they are, the reference
  and why, before anything changes, and only they are removed.
- Gateway points of the IED whose reference is gone are named; they stay in the gateway list.
- The result passes the checks a project gets when it is opened, or nothing changes.
- Re-import needs the run stopped, writes a verified recovery backup first, and Undo takes it back.

## Rules

| ID | Rule | Enforced by |
| --- | --- | --- |
| REIMP-01 | The re-imported IED keeps its id and settings; only links the new file cannot keep are removed, each listed in words, and the result opens. | `studio/engine/internal/project/reimport_test.go::TestReimportKeepsIdentitySettingsAndLinks` |
| REIMP-02 | The same file changes nothing and loses nothing. | `studio/engine/internal/project/reimport_test.go::TestReimportOfTheSameFileChangesNothing` |
| REIMP-03 | A file whose IED has another name is refused. | `studio/engine/internal/project/reimport_test.go::TestReimportRefusesAnotherIED` |
