# 0157: "Using a virtual machine?" box, helper status fix, VM picture guide; version 0.1.21

Date: 2026-10-22. Follow-up to [0156](0156-activate-once-vm-host-key.md) after its first real VM
run ([test record](../../testing/vm-host-real-vm-2026-10-22.md)). Decision:
[ADR-0099](../../decisions/ADR-0099-license-vm-host-key.md) (VM host key), unchanged. Spec:
[licensing](../../specs/licensing.md) LIC-77, LIC-84. No new decision record: contracts, protocol and
dependencies are unchanged; this is wording, placement and one start-up wait.

## Why

The real VM run showed three things the owner wanted fixed:

- The activation screen inside a VM had no clear instructions: one line of text at the very
  bottom, below the offline steps. The owner wants a visible VM option next to offline activation,
  both options equal, both closed, both bold.
- Right after ticking "Let my virtual machines use this license" on the host, Settings said "On, but the
  helper is not running yet. It starts the next time you sign in to Windows." The helper was in fact
  running; the status was read before it had taken its lock.
- A tutorial for the VM steps, with pictures, for the website.

## What changed

- **Helper status** (`studio/engine/internal/licensing/hostkey/sharing.go`): turning sharing on, and the
  refresh at app start, now start the helper and wait up to 3 seconds for it to take its lock before the
  state is read. A helper that never takes the lock (sharing switched off meanwhile) is not an error. The
  start-up wait is the only behaviour change in Go code.
- **Activation screen** (`VmHostNotice.tsx`, `copy/vmHost.ts`, `license-gate.css`): the single line is now a
  closed disclosure "Using a virtual machine?" right after the closed "Server & offline options". It holds a
  short intro, three numbered steps (activate the PC, tick the switch, come back to the VM), a
  "Looking for a licensed PC on the virtual network..." spinner and a note (works offline while that PC is on
  with the option ticked; a copied VM stops). Both disclosure titles are bold. The screen still asks for the
  license status every 6 seconds while the box is closed, so it continues on its own. `gateNotice` was
  replaced by `gateTitle`, `gateIntro`, `gateSteps`, `gateLooking` and `gateNote`.
- **Website**: new data file `website/src/data/guides.ts` and a "Picture guides" section on
  `tutorials.astro` after the videos: "Use ScadaBench in a virtual machine", four steps with screenshots
  (`website/public/guides/vm-0` to `vm-3`, `.webp`). New FAQ "Can I use ScadaBench in a virtual machine?"
  in `faqs.ts`. The screenshots are of the real app with demo answers only; no private engineering data
  (rule: no private engineering data in public material).
- **Version 0.1.21** is prepared, not released: `studio/backend/studio_app/version.py` (`0.1.21+dev`),
  `website/src/data/site.ts` (`VERSION '0.1.21'`) and a new top entry in `website/src/pages/changelog.astro`
  ("Clearer setup for virtual machines"; the 0.1.20 entry moved down).
- **Tests**: `TestTurningSharingOnWaitsForAHelperThatIsStillStarting` (a helper that takes its lock only
  after a few checks); frontend test for the new box (title, three steps, box sits right after the offline
  options, closed disclosure, spinner); the real-helper e2e test now skips when a real VM helper already runs
  in this Windows session (one helper per session, the test's helper would end at once).

## Evidence

Run in this worktree on the owner's PC, 2026-10-22, with `studio/.venv` a junction to the main checkout's
venv (the Go suite's Python parity tests need Flask) and TEMP, TMP and GOTMPDIR set to `.local/gotmp`
(antivirus interferes with the system temp folder).

- `python studio/tools/verify.py`: all sections PASS (file lengths, architecture, documentation, engineering
  history, specifications, GOOSE engine go test, Go engine go test including pyjson vs CPython), 4 min 33 s.
  It is run again after this note and the ledger entry were added; see the ledger entry of this change.
- Frontend: all 66 test files with `node --experimental-strip-types --no-warnings --test`: 314 tests, 314
  pass, 0 fail (313 before, plus the new box test). `npx tsc --noEmit` in `studio/frontend`: no output.
- Go, in `studio/engine`: `go vet` clean and `go test -count=1 -v` pass for `./internal/licensing/...`
  (licensing, hostkey, pytext), `./internal/api/hostkeyapi/` and `./internal/app/`: 5 packages ok,
  0 failures, 10 skips (the real-helper e2e test, see Limitations; one legacy-machine scenario that is
  Python-only; and 8 opt-in tests that need `STUDIO_NATIVE_BIN`, `STUDIO_STRESS` or `STUDIO_TIMING`).
  The new sharing test passes (0.20 s).
- Website: `npx astro build` with Node 22.23 from `%LOCALAPPDATA%\pi-node\current` (the system Node 22.11
  is too old for Astro): 11 pages built; `dist` has the guide section, the four images and 0.1.21 in the
  changelog; `website/dist` is git-ignored.
- The screenshots of the closed and open box (`vm-1`, `vm-2`) show both options closed and bold, the three
  steps and the spinner line, with the same weight for both titles.
- The real VM run that led to this change: a VMware Workstation guest (Windows 10) was licensed by the
  host through the helper with no license files of its own; see the
  [test record](../../testing/vm-host-real-vm-2026-10-22.md).

## Limitations

- The e2e test with the real helper skips on this PC while sharing is switched on (a real helper already
  runs in this Windows session), so it was not run for this change here. Turn sharing off to run it; it
  passed on this PC for 0156 when sharing was off.
- The status fix and the new box have not been tried on a real VM or a real PC with a new build; the real
  run recorded above used the 0.1.20 build.
- The Python backend is frozen and unchanged. The helper, the wait and host discovery are Go-only (as in
  0156); the box itself is part of the activation screen and is shown for either backend.
- Version 0.1.21 is only prepared in files: no installer or portable build was made, nothing is committed,
  pushed or deployed, and the website is not published. The picture guide is not on the live site yet.
- The activation tutorial VIDEO is outdated: `marketing/videos/lessons_license.py` still says "New computer?
  Enter the key there and choose Move the key here", but moving keys was removed in 0153 (a new computer or
  motherboard needs a new license). It needs a re-record; this change does not touch the videos.
- Not covered on a real VM (unchanged from the test record): simulator engines running under the host
  proof, stopping the helper, copying the VM to another PC.
