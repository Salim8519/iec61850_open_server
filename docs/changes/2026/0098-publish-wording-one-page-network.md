# 0098: "Be this IED" wording and a one-page network setup

Date: 2026-10-18. Decision: [ADR-0068](../../decisions/ADR-0068-publish-wording-and-one-page-network.md).
Spec: [lab network](../../specs/lab-network.md) (NET-13, NET-14).

Owner report: setting IEDs to "Real device" (meant as "publish on the real network") made the
network setup fail with "There are no simulated IEDs to put on the network"; after switching back
to Simulate it worked, and the words, steps and buttons were called misleading.

## Changes

- **Mode names** (stored roles unchanged): *Be this IED* / *Read real IED* / *Leave out* in the
  device header, filter chips ("Be this IED", "Read real", "Left out") and pills ("Publishing",
  "Be this IED · stopped", "Reads real IED", "Left out"). Role hints and backend refusals use the
  same words.
- **Role line** under the device header (`features/devices/RoleLine.tsx`): who publishes, address,
  "reachable from this PC only" with *Put on a network*, or the network card.
- **Top bar** "N IEDs reachable from this PC only" / from the card name; opens Settings › Network.
- **One-page setup** (`NetworkSetup.tsx`, `AddressTable.tsx`, `setupWords.ts`) replaces the
  4-step wizard (`LabWizard`, `AdapterStep`, `AddressStep` removed): network card list, SCD address
  per IED, check per row with one suggested free address (`alternative` from the planner), one
  sentence for the Windows changes, exact list in a disclosure, one button "Put N IEDs on <card>".
- **No published IED** (`NoPublishers.tsx`): names each IED's mode, *Make them publish*.
- Device header name wraps instead of being cut; network check column stacks instead of clipping.
- `check_file_lengths.py` excludes the git-ignored release outputs `dist-protected`/`dist-sandbox`
  (the portable exe writes its data folder there).

## Evidence

| Check | Result |
| --- | --- |
| `verify.py` | PASS |
| Python unit + integration + packaging | 430 passed, 2 skipped (new `test_network_setup_page.py`: 3) |
| Protocol (`-k "run or network or ied"`) | 16 passed, 1 skipped |
| Frontend `node --experimental-strip-types --test` | 156 passed (new `networkSetup.test.ts`: 3); `tsc` clean |
| Dev app in the in-app browser (fresh data folder, `scd/model_substation.scd` FEED1 + BUS1) | Devices header *Be this IED / Read real IED / Leave out*, role line "publishes as this IED at 127.50.0.2:102 · reachable from this PC only"; top bar "2 IEDs reachable from this PC only"; both set to *Read real IED* → Settings shows "No IED is set to Be this IED" with "FEED1: Read real IED · BUS1: Read real IED" and *Make all 2 IEDs publish*, which fixed it; setup on VMnet1 previewed 172.16.1.20 / 172.16.1.30 "from the SCD", ✓ Free, one sentence, button "Put 2 IEDs on VMware Network Adapter VMnet1" (not pressed: no Windows change made); Run → pills "Publishing" |
| Clipped-text check (`tools/clip_check.js` selectors plus `.reach-chip`, `.role-line`, `h2`, `button`) | 0 clipped on Devices (49 checked) at 1366×768 and 1920×1080, on the setup page (31) and while running (50) |

## Limitations

- *Put N IEDs on <card>* was not applied in this check (it needs a UAC approval); the apply path
  is unchanged and covered by NET-05/NET-09 tests.
- A typed address is still not checked against the other rows for duplicates (existing behaviour).
- No EXE built (owner: test in the browser).
