# ADR-0148: Re-import an IED's CID, keeping its id, settings and links

Accepted: 2026-11-23. Owner, 2026-11-23, for 0.1.37: "can u add re import cid option? so same cid gets
updated without deleting equipment? or warns user what will be lost?, sometime we do this, but want
as everything as possible to stay the same". Change [0238](../changes/2026/0238-reimport-cid.md).
Spec [Re-import CID](../specs/reimport-cid.md).

## Problem

A newer CID of an IED (a relay's configuration changed) could only be taken by deleting the IED
and importing it again. Deleting clears every diagram link, point panel, source, test point and
scenario step of the IED and its settings (role, addresses, Allow commands, card), and the new
import gets a new id, so nothing reconnects by itself.

## Decision

1. **Domain (`project.Reimport`, pure).** The new model (imported under the IED's own name) takes
   the old IED's id, endpoint, state and validated settings (`netset.ValidateIEDSettings`) and
   `scd_deadbands`. A pruner walks every place a project links to an IED: a link stays when its
   reference is in the new model with the same type (signal kind, command class; typed commands
   from the stored SCL); a report that no longer holds the signal is cleared and the link stays.
   Other links are removed and recorded `{where, ref, why}`. Each touched equipment, source, test
   point and scenario step is then checked alone with the real validators; equipment that still
   fails loses all its links to this IED (the equipment stays). Finally the whole result runs the
   checks of opening a project (`ValidateDiagram`, `ValidateSources`, `ValidateScenarios`,
   `ValidatePanel`); a failure refuses the re-import and nothing changes.
2. **Gateway points** (by IED name and reference) are not checked against the model, so they stay;
   the ones whose reference is gone are named.
3. **Service and route.** `POST /ieds/{key}/reimport` (multipart `file`, `apply` "1"), run stopped.
   Without apply it answers the comparison (`signals`, `controls`, `reports` before/after/added/
   removed, `kept`, `lost`, `gateway_gone`, `same_file`); with apply it writes a verified
   `recovery/before-ied-reimport-*.scadabench`, then commits (an Undo step). A file without the
   IED's name is refused and names the IEDs it has.
4. **Screen.** Devices › Files › "Re-import CID…": choose the file, read the comparison and the
   links that cannot stay, then "Re-import" (or "Re-import and remove N links").

## Compatibility

Projects and their stored format are unchanged; the IED's stored source and filename become the
new file's. Delete IED keeps its behaviour (its per-equipment cleanup moved to `dropNodeLinks`).
Python has no such route (Go-only).
