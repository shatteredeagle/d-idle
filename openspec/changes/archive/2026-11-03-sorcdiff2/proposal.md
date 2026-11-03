# sorcdiff2

## Why

SORCDIFF (card sorcdiff) showed the gate character casts only Charged Bolt, Frozen Armor, Frost Nova and Fire Bolt, and left three items for deeper work: mana paid at the action frame instead of the command (stateful, no harness), the Frozen Armor trigger and armor group, and the Charged Bolt pipeline. This card builds counted differentials for them on the real d2d build against game.exe's own code, and a private fix branch for each proven deviation.

## What Changes

- `docs/research/sorcdiff2.md`: the differentials, their counts, the fix branches, the ranked ready-for-Opus list.
- `tools/emu/sorcdiff2_run.sh`: the WSL script that built and ran them (a copy of the ops-repo script, so the run can be repeated).
- Private fork (not in this repo): branches `local/sorc-*` on upstream main, harness and fix in separate commits, the private remote only.

## Impact

Docs and one script. Nothing pushed to GitHub, no PR, no comment.

Overlap (declared): extends sorcdiff (docs/research/sorcdiff.md, same subject); reads sk-mana-trace, sk-missile-trace, sk-armor-trace, sorcrest and cast-trace-gameexe; edits none of their files.
