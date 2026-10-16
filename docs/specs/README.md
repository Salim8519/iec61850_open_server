# Feature specifications

Each large feature has one specification. A specification is the contract that
code, tests and later LLM sessions must respect. It exists so behaviour cannot be
changed, dropped or invented by accident.

## Rules for every specification

- Every rule has a stable ID such as `GOOSE-03`. IDs are never reused or renumbered.
- Every rule names at least one enforcing test in the **Enforced by** column, written
  as `` `path/to/test_file.py::test_name` `` or `` `path/to/file.test.ts::test name` ``.
- `python studio/tools/check_specs.py` (run by `studio/tools/verify.py`) fails when
  a rule has no enforcing test or names a test that does not exist.
- Hardware-only evidence may add `evidence:` links to change records, but a rule
  still needs an automated test for everything that can be checked offline.

## Changing a specification

1. Read the whole specification before editing code it covers.
2. If the code must behave differently, change the rule **first**, explain why in a
   decision record, then change the enforcing test, then the code.
3. Never delete a rule to make a test pass. Retire it with `Retired:` and the reason.
4. Never claim behaviour in a rule that has no evidence. Planned behaviour belongs
   in known issues, not in a specification.

| Specification | Feature |
| --- | --- |
| [Lab network](lab-network.md) | Binding simulated IEDs to a real adapter |
| [GOOSE](goose.md) | Publishing, receiving and supervising layer-2 GOOSE |
| [Real IEDs](real-ieds.md) | Read-only live acquisition and SCD comparison |
| [Beginner guidance](beginner-guidance.md) | Hints, plain wording and status display |
| [UI structure](ui-structure.md) | Four places, design tokens, one implementation per job |
| [SLD import](sld-import.md) | Single-line diagram built from the SCD `Substation` section |
| [Signal flow](signal-flow.md) | Declared GOOSE and report links between IEDs |
| [Station health](station-health.md) | Live network and real IEDs compared with the SCD |
| [Sampled Values](sampled-values.md) | Listening to IEC 61850-9-2 streams and checking them against the SCD |
| [Test mode](test-mode.md) | Mod/Beh/Sim of real and simulated IEDs, simulation traffic |
| [SLD tidy](sld-tidy.md) | One-press layout of the diagram like a real single-line diagram |
| [Live colouring](energization.md) | Infeed symbol and live/dead/earthed wires from switch positions |
| [Interlock hints](interlock-hints.md) | Friendly switching hints; never blocking |
| [App window](app-window.md) | Windowed EXE, one instance, show again, confirm and stop on close |
| [Bay templates](bay-templates.md) | One-click line feeder, transformer bay and bus coupler |
| [Guided tour](tour.md) | First-start walkthrough of the four places, reopened from Help |
| [Licensing](licensing.md) | Trial and 1-PC lifetime license, offline activation, protected release |

## Proposed specifications

Large features are specified **before** they are built, in [proposed/](proposed/README.md).
A proposed spec is planned behaviour, not a capability. Its rule IDs are reserved, and
it moves into this folder (and this table) only when its enforcing tests exist.
`check_specs.py` does not scan `proposed/`.
