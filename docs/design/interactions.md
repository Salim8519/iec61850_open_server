# Interaction contract

- Add equipment from SLD › Add (click or drag a tile); connect terminals; select to edit in the sheet.
- Junctions are drawing-only connection points with four terminals and no
  mappings, start position or rotation; wiring is free-form and may dangle.
- Every terminal may connect to every terminal; wires may leave equipment from
  any side. Existing saved wiring keeps its behavior.
- A relay is a named status panel, not a wire-path object: no terminals, one
  row per status entry with its live value. Entries are added, named and
  re-assigned while stopped against any imported signal; while running a
  writable entry can be set or pulsed and a manual override resumed, and a
  non-writable entry is display-only with an explicit badge.
- Relay signals are compact cards (ADR-0024): inline rename, quiet reorder and
  remove, and a Triggerable/Display-only tag visible while stopped. A single
  Pulse button names its target, the opposite of the current Boolean value.
  "Add signals" opens an inline picker that stays open for several adds.
- Canvas highlight and sheet are projections of the same selected object ID.
- Choosing a breaker/switch command fills its position from that control's status
  point when no position is mapped; the user can change it.
  A persisted diagram update must not clear one without clearing the other.
- Live values use keyed equipment subscriptions. Unrelated reports and Event Log
  updates must not recreate, hide or remeasure unaffected canvas equipment.
- Rotate, duplicate, multi-select and undo/redo keep stable object identifiers.
- Canvas editing (ADR-0027): left-drag on empty canvas box-selects; Space,
  middle or right drag pans; Shift/Ctrl-click toggles. Ctrl+A/C/X/V/D, Delete,
  Esc and arrows act on the selection. Paste/duplicate create new IDs, keep
  only wires between copied objects and give unique labels. Terminals have a
  generous hit zone and snap radius; a drag commits once on drop.
- Tidy up (ADR-0040, spec `sld-tidy`) moves items, re-sides wires and turns symbols
  only; busbars on top, bays straight down, unwired panels parked right; one undo
  step; with two or more selected it tidies only the selection.
- Infeed (ADR-0041) marks where power enters; drawing only, never mapped or exported.
  While running, wires show Live/Maybe live/Dead/Earthed/Earth fault by line style
  with a worded legend (ADR-0042); the sheet names the part's power state.
- Switching hints (ADR-0043) appear under Open/Close and never disable them; external
  switching against a hint is noted in Activity. Studio never claims CILO interlocking.
- Add › Whole bays (ADR-0044) adds a line feeder, transformer bay or bus coupler as one
  edit, wired to the selected busbar junction(s).
- Physical equipment and IEDs are separate entities connected through bindings.
- Choose compatible imported signals and control objects from filtered lists.
- Open/Closed/Intermediate/Invalid use explicit text and distinct symbol geometry,
  readable when zoomed out (ADR-0045). A confirmed CB/DS/ES change pulses the card
  twice (a static ring with reduced motion); the first value after load never pulses.
- Errors identify an object and link to its editor. Do not rely on toast-only errors.
- Every list has the one filter bar (ADR-0064): filtering only hides rows on the page (Data
  points filters in the backend over all points); the count says "N of M" and Clear resets.
  Chips and dropdowns are remembered per list by the backend; the typed search and Pause are not.
- Tool settings, status, errors and the primary button stay in the tool's left column; the
  list beside it fills the window. Migration and recovery notices are one dismissible line.
- Disable structural edits while running and explain why; allow runtime values.
- Manual override visibly suspends an automatic source until released.
- Stop ends the run. Pause Automation leaves MMS and manual control active.
- Port/IP addresses are visible and copyable; no terminal commands in user flows.
- Templates and examples must be clearly labelled as simulation models.
- CID/ICD/SCD import first lists the file's IEDs with none selected. The user
  explicitly chooses one or more IEDs and confirms; Cancel adds nothing.
  Existing IED names and unsupported server entries explain why they cannot be selected.
- Export SCL stages explicit initial-value edits before a verified download. Explain
  that the entire original document, including unselected peer IEDs, is retained.
  Do not silently include live values, simulation addresses or canvas topology.
  Original download remains separate; failed validation must not download a file.
- After startup recovery, show that the saved substation was restored with all
  IEDs stopped. Unreadable-autosave notices expose the preserved backup location
  and diagnostic; do not silently present a blank project as a successful recovery.
- Mapping follows IED, report, signal description and exact path. Selecting the
  final signal assigns it immediately; `Not assigned` removes it.
  All signals is a deliberate alternative, not a silent fallback from a failed report.
- Signal selection lists only runtime-editable points. One-point switch mapping
  derives native Dbpos or Boolean conversion from the selected imported SCL type;
  it never infers type or function from a reference name.
- CB/DS/ES state and command mappings are independent. State accepts native Dbpos,
  one Boolean, or separate open/closed Boolean contacts from arbitrary IEDs. Show
  Intermediate and Invalid explicitly; never guess from integer/enum values.
- CB/DS/ES commands accept DPC only. False means Open and true means Close; the
  two-bit Dbpos representation belongs to feedback, not the MMS command value.
- After MMS accepts an equipment command from Studio or an external client, update
  every explicitly mapped simulated state point or contact through the normal
  sample path. If feedback fails, state clearly that the command was applied;
  never disguise the partial outcome.
- Trip relay accepts SPC only and stores command and state polarity separately.
  New nodes default to Operate=0 / Reset=1 and Operated=1 / Reset=0; users can
  invert either setting. Never auto-reset or copy raw `ctlVal` into feedback.
- Enhanced-SBO selection displays "selected; waiting for Oper" with its imported
  timeout. Selection alone never changes switching position.
- Enhanced-SBO DPC mappings may explicitly enable the warned CONCEN compatibility
  profile. Only origin `CONCEN` is inverted and executed from successful `SBOw`;
  standard clients and all SPC/Trip controls remain unchanged.
- Never describe the CONCEN profile as automatic intent detection. If one origin
  sends the same Boolean for operator-requested Open and Close, show that the input
  is ambiguous and require an explicit client or compatibility-policy correction.
- Show the backend session and project revision in every tab. Synchronize remote
  tab edits automatically and reject a stale diagram save before it can overwrite
  the authoritative project.
- Push Button has no wire terminals and is pressed by clicking it on the canvas
  while running (press and select are one gesture); the properties panel stays
  as the detailed view. New buttons default to latching Boolean status.
  Momentary mode sends one chosen SPC value per press. Latching mode toggles
  one Boolean status point through the acknowledged sample/report path.
  Self-resetting mode drives one Boolean status point to the opposite of its
  configured rest value (0 or 1) and returns it automatically after the
  configured delay through the backend pulse path.
- Analog Source maps a numeric MX point and configures the shared manual,
  constant, ramp, sine, step, noise or CSV source in Equipment Properties.
- Signal Test controls persist without SLD symbols. Set changes feedback directly;
  Boolean Pulse returns to its previous acknowledged value unless a newer write wins.
  Pulse runs on the backend, including while automation is paused. Never replay on reopen.
- Bulk deletion reviews all selected IEDs and affected test controls together, preserves
  drawing equipment/wires, and saves one recovery archive before changing the project.
- Properties list what each object still needs (ADR-0025). With one project IED,
  signal and control pickers preselect it; nothing saves until a point is chosen.
- Diagram edits that change no persisted field (for example React Flow size
  measurement on load or selection) are not saved and create no revision.

- Signal layers (ADR-0065): *GOOSE messages* and *Who is reading* are remembered view switches;
  they never edit the diagram. One dot per real stNum change, at most one per link per second;
  hollow "(expected)" dots when no listener runs; problems stay on the subscriber tag in words.
  Clicking a tag or link opens its details in the sheet. GOOSE and MMS traffic must not
  re-render equipment nodes.

Guided network and real-IED work (ADR-0028, ADR-0029, ADR-0031):
- Windows changes are listed in plain sentences before Apply; nothing changes until
  Apply, and the administrator prompt is announced first.
- Every conflict, GOOSE problem and live state shows a fix hint next to it.
- Real IEDs never receive commands; the refusal is shown, not hidden.
- Old live values after a lost connection are marked "Old value", never shown as fresh.
