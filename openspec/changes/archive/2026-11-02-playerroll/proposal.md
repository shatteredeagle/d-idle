## Why

The player-side missile damage roll differs from game.exe in two places that one upstream PR can fix: the element roll never reaches its max (the pair 9ba8d369 / 23e2baec, SKMISS), and the physical roll is shifted to whole points before the resistance (`local/skabrecheck-phys`, with the -100 floor the independent verifier asked for). This card builds the bundled fork branch `pr/player-roll` on upstream main, re-runs the differentials on the combined head and writes the PR draft. Nothing is opened or posted.

## What Changes

- docs/research/pr-playerroll-draft.md: the PR draft, every claim with a counted run, the reach, the side effect, the presubmit WARN answers, what the third-party check must do.
- tools/emu/det_scenarios_lab.py: the scenario `hit_phys` (druid Twister, Shock Wave, Tornado) the independent verifier said was still owed.

## Impact

Docs and the det scenario list only. No spec, policy, ADR or glossary change. The fork branch is on the private remote only.
