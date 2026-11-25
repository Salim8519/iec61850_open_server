# 0241 Merge of the GitHub master (Raised panels, What's new) into the local master

Date: 2026-11-26. Owner: "pull main or local and merge both ... the master has other agent work we
should merge and check to local master and then re push to master"; "our local repo is the main
leader". The owner chose each resolution below before it was applied.

## What came together

- Local master (leader): 0.1.37 work, Devices commands step by step (now change 0239, ADR-0149) and
  Re-import CID (now change 0240, ADR-0150). Version stays `0.1.37+dev` / website 0.1.37.
- GitHub master: Raised panels and Find IEDs interfaces (change 0237, ADR-0147), white sidebar,
  What's new and click-rule sweep (change 0238, ADR-0148). Kept as published, numbers unchanged.

## Resolutions

- Numbering: both sides used 0237/0238 and ADR-0147/0148. The GitHub ones stay; the local, never
  pushed ones became 0239/0240 and ADR-0149/0150 (files renamed, every reference in code comments,
  specs, API reference and indexes updated).
- Release notes: the website changelog now renders the one shared list (`copy/releaseNotes`, ADR-0148).
  The local 0.1.37 notes moved from `changelog.astro` into `notes030.ts` as the 0.1.37 entry; the
  Raised panels / What's new notes, written as 0.1.37 on GitHub, are the 0.1.38 entry (shown on the
  website only once it reaches 0.1.38). `ChangelogEarly.astro` is removed: notes000/020 hold those
  releases. Change 0238's text still says 0.1.37 for its notes; this record supersedes that.
- Code: the local `RealCommands.tsx` and `PointCommand.tsx` stay; the `wire` double border (R8,
  ADR-0147) is added to every new send button (`CommandInputs.tsx`, `RealCommandControl.tsx`, point
  panel choices).
- Ledger: both sides appended two entries after change 0236. The GitHub entries stay byte for byte;
  the two local, never pushed entries were appended again after them with the new numbers, their
  original text and times, and a fresh chain (`prev`/`hash`).

## Evidence

- `python studio/tools/verify.py`: all PASS (file lengths, architecture, docs, specs, offline UI,
  GOOSE and Go engine tests); the history check passed after the merge commit.
- Frontend tests 523/523; `tsc --noEmit` clean.
- `releasesUpTo('0.1.37')` starts with the 0.1.37 commands entry; What's new for `0.1.37+dev` shows it.

## Limitations

- The merged UI was not opened in the browser in this change; the website was not rebuilt
  (its `dist` holds the published 0.1.37 download).
