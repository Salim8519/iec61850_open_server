# ADR-0067: Run one IED, equipment test buttons, clean SLD

Accepted: 2026-10-18. Spec: [run one IED, test buttons](../specs/run-single-ied.md),
[signal layers](../specs/signal-layers.md). Change:
[0097](../changes/2026/0097-clean-sld-test-buttons-run-one.md).

**Problem**

Using release 0.1.10 the owner found: *Run* starts every IED that is not *Off* with its whole
CID and there is no way to run or connect a single IED; breakers without a command link cannot
be moved for testing; the SLD showed a big column of "outside project" sender tags and cryptic
short tags ("C03"); GOOSE said "Not watched" without saying why; Devices cut "Simulated".

**Decision**

1. **Run model.** The coordinator keeps one run with a set of running IEDs (`active`,
   `running_ieds` in `GET /project`). `start(ied_ids)` / `stop(ied_ids)` add or remove IEDs;
   `start()` / `stop()` without ids keep their old meaning (everything not Off / everything).
   A new module `simulation/run_selection.py` keeps the signal index, sources, sample locks,
   workers (`Supervisor.add` / `stop_some`), real sessions (`LiveManager.connect` /
   `disconnect`), MMS client lists and equipment states consistent. Endpoint conflicts are
   checked against running IEDs plus the new one. Route: `POST /runtime/ieds/<id>/start|stop`.
   The GOOSE and SV listeners stay started for the whole run (they watch the network, not one IED).
2. **Drawing lock.** Running any IED locks the drawing and IED roles like a full Run. One rule is
   simpler to explain than a partial edit mode, and roles/structure feed every worker.
3. **Test buttons.** `POST /equipment/<node>/test` writes the bound point(s) of a simulated,
   running IED through the existing sample writer (source `equipment_test`), the same path as
   Devices data-point editing. Pure rules in `domain/equipment_test.py`; no MMS command, no new
   authority in the browser. Real IEDs are refused (they use their command link). The command
   link becomes optional in the sheet (calm hint instead of "Needs: command").
4. **Clean SLD.** Senders outside the project never get a canvas tag; they are a toolbar list
   ("N senders outside project ▾" with *Import its CID*). Their GOOSE is one dot from the canvas
   edge for ~2 s. Tags show the full IED name; a link problem is a small ⚠ on the receiver tag.
   "Not watched" becomes "Needs lab network" / "Not checked" / "Outside project".

**Compatibility**

- Project files are unchanged; nothing new is saved. `GET /project` gains `running_ieds`;
  `POST /runtime/<action>` also returns `running`. Existing clients keep working.
- `Supervisor.start` and `LiveManager.start(project)` behave as before for a full Run.
- Native workers and wire semantics are unchanged; dev and packaged behaviour are identical.
- The `runtime` event may carry `running`; a new `ied_runtime` event records each single start
  or stop in the run log; `equipment_test` records each test button.

**Consequences**

- To change roles or the drawing, stop everything (spec run-single-ied).
- GOOSE from senders outside the project only animates when the listener reports it (lab
  network with Npcap); in *This PC only* mode their links say "Outside project".
