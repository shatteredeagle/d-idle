## Why

The generality study (docs/research/bot-generality.md, section 5 #2) found the same four soak symptoms fixed one by one: the town wedge pocket, the clvl-7 kit-less pack death, the poverty loop and the campaign stall. Nothing pins them as whole situations, so a later change (area tiers, a second class) could bring one back unseen. no observable behaviour change: tests and one index doc only, no production code is touched.

## What Changes

- New scenario tests named `TestScenario_<Name>` in `internal/nav`, `internal/bot`, `internal/town` and `internal/combat` (they need the unexported fakes of each package, so a separate package would copy them). Each replays a recorded situation and asserts the end state, with a knob-off subtest that expects the old failure.
- New `docs/scenarios.md`: one-page index (scenario, evidence, invariant, test path, knob that proves it).
