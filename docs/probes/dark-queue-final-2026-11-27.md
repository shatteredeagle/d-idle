# Dark queue: the final flip list (2026-11-27)

> status: current (docs only; nothing was flipped, no lab run was made in this card). Supersedes the verdict column of `docs/probes/dark-queue-plan-2026-11-27.md` (the dossier, kept for the per-row evidence) and the "Flips ready" table of `docs/roadmap-fable.md` where they differ. Rule in force: user decision 2026-11-27, no 24 h waits: flip as soon as the scenario, drill or accelerated soak is green, 2 h hard watch (`tools/flip/flip.py` default), auto-revert. Tags: V = read in the cited file or ledger row on main 093a1aac1, S = secondary claim, U = unverified. All lab evidence is CHEATED-setup, mechanics only.

## 1. Where the two docs disagreed, and the ruling

| row | dossier | roadmap | ruling and evidence |
|---|---|---|---|
| `quests.stepExec` | NEEDS-CODE (duriel BLOCKED) | ready | FLIP NOW on Questy, Act 1 hero. V `cain.scroll.stepexec`, `cain.read.stepexec` green (soak doc 3.3, GREEN of 6 with the one BLOCKED row explained as a `crowd-abandon` exit, not the executor); V `drill.act2.assemble.stepexec` GREEN 3/3 (drill doc). V sixth Duriel run: kill counts, chunk parks at `talk_tyrael` ("not in a town", 3 tries, parked), so the Act 2 end chunk still fails safe, it does not loop. The ledger text asks for all four scenarios green 2 of 3: the fourth is not, so the flip is a knowing departure from that text; the 2 h watch with auto-revert is the safety net. Duriel/Tyrael stays a code card (field-NPC path for Tyrael, drill doc "Next blocker"). |
| `town.plan` 4 | FLIP-NOW for stage 3 (done) | "stage 4 ready on Doofer" | NOT READY. V `docs/design/townflip-p5.md` 4b: the three stage 4 scenarios are "not run" and "the stage 4 flip waits for these three green"; ledger says the owner looks before stages 3 to 5 (ADR 0028). Lab drill first (section 3, G2). |
| `bot.arbiterControl`, `bot.arbiterFamilies` 255 | DRILL (rerun 255 Sorceress 201) | stay on | STAY ON Doofer, no widening. V TASK-1048 result: the 255 Sorceress "RED" was a 40 min quest-board slice outlasting the 10 min watchdog on a soak binary built without loophang (de381e7ea), not a spin; game time and kills advanced. U: the rerun with the fixed binary is not recorded. Weak RED families 3 Necromancer xp (-2224, 2 of 3 seeds) is within one seed's spread (soak doc 5.1a). |
| `agency.planner` | FLIP-NOW mode 1 | not listed | FLIP NOW mode 1 (log line only, V knobs.md: "0 off, 1 shadow"). U: scenario `agency.plan-matches-profile` has no recorded run; mode 1 changes no decision, so the 2 h watch is the check. Mode 2 stays an owner go. |
| `bot.checkpointEnable`, `combat.missileDodge`, `combat.threatDodge`, `combat.standOK`, `combat.deliveryExpect`, `bot.portalGuard` | FLIP-NOW | stay on | ALREADY ON: ledger `enabled` on Doofer (V), windows 45 min clean. Nothing to flip. They have no drill (U), the clean window is the evidence under the 2026-11-27 rule. |
| `bot.arbiterShadow`, `bot.tickLogSec` | FLIP-NOW | promote | FLIP (re-flip to close the window): both `trial` (V), on Questy since 06:01Z. |

## 2. Ordered flip list (run in this order; each line with `--dry-run` first)

Sandboxes (V ledger today): Doofer `d2play-20261108` Paladin clvl 21 per the aura audit (the clvl is U now); Questy `d2play-20261125q` Paladin clvl 12 per the same audit (U now). Doofer already carries: bundle `bot.portalGuard`, `combat.deliveryExpect`, `combat.threatDodge`, `combat.standOK`, `combat.missileDodge`, `bot.checkpointEnable` (enabled), `bot.arbiterControl` 1 and `bot.arbiterFamilies` 255 (enabled), `town.plan` 3 and `combat.deliveryUpkeep` 2 (trial, window closed clean). Questy carries only `bot.arbiterShadow` 1 and `bot.tickLogSec` 1 (trial; `bot.tickLogSec` also on Doofer). Rule to avoid mixing: decision-changing knobs go to one sandbox per window, log-only knobs may ride on any.

Step 0, offline, no slot (the replay evidence the shadow rows ask for):
```
go run ./cmd/d2scenario arbiter-replay -strict tools/provingground/arbiter-outcomes/*.json
go run ./cmd/d2arbiter LOG && go run ./cmd/d2arbiter -follow LOG    # LOG = the first night tick log copied off Questy
```

Flip 1, Doofer, log and shadow only (three rows, none changes a hero decision; V knobs.md):
```
python3 tools/flip/flip.py d2play-20261108 bot.arbiterShadow=1 bot.tickLogSec=1 agency.planner=1 --why "log and shadow only: arbiter replay green, tick log night clean" --window 2h
```
Ledger status: `bot.arbiterShadow` trial, `bot.tickLogSec` trial, `agency.planner` dark: all accepted (V flip.py line 288). Closing marks all three `enabled` (and thus also the Questy copies). Reason for Doofer: the arbiter control rows run there and `bot.arbiterControl` depends on shadow 1 (dossier, U whether Doofer already has it in its tuning file; the flip verifies the value either way).

Flip 2, Questy, in parallel with flip 1 (one window per sandbox, the two sandboxes are independent):
```
python3 tools/flip/flip.py d2play-20261125q quests.stepExec=1 --why "stepExec: cain.scroll, cain.read, act2 assemble green; Duriel/Tyrael chunk parks safely, known blocker" --window 2h
```
Ledger status `dark`: accepted. Watch for: Questy standing in Act 1, the Cain scroll and Akara read chunks drive the executor; a `gave up after 3 tries` park on any step is a finding, a hard invariant breach reverts. U: Questy is not in Act 2 now; if it reaches Act 2 inside the window the Tyrael park is expected (V sixth run).

After both windows close clean: the coordinator sets the Doofer-only trial rows `town.plan` (value 3) and `combat.deliveryUpkeep` (value 2) to `enabled` by ledger edit (same evidence as the other enabled rows: 45 min window closed clean 2026-11-27T14:25Z, V ledger note). flip.py cannot do it (the row is not changed by a flip).

## 3. Gated flips (flip the moment the named drill is green; commands ready)

All drills go on the builder long lane in the 23:00 to 07:00 EST window, one `labrun ... --lane long --detach --notify --pull provingground-out -- tools/provingground/run.sh ...` job each, after `labrun the worker host -- top -bn2 -d1` shows idle plus nice at least 25 percent.

| gate | drill (green means) | flip line after it | instance and ledger status |
|---|---|---|---|
| G1 `quests.boardSM` | `run.sh -id board.sm.restart-mid-chain,board.park.persist-restart,board.sm.stop-cascades -class Paladin -seed 11 -parallel 1`; GREEN, hard invariants 0. V bsm9: restart-mid-chain already GREEN Paladin 11; U `park.persist-restart` cannot make its setup, a scenario fault is not a finding | `python3 tools/flip/flip.py d2play-20261125q quests.boardSM=1 --why "boardSM: restart drill green Paladin 11" --window 2h` | Questy (after flip 2 closes); row `dark`, ok. Needs `quests.board` 1 (U: confirm the sandbox has it) |
| G2 `town.plan` 4 | the three `town.plan.belt-refill-from-pack`, `town.plan.sell-keeps-need`, `town.plan.scroll-hoard` with `town.plan=4`, Paladin 11 and Sorceress 3 (townflip-p5 4b), plus the owner look (ADR 0028) | `python3 tools/flip/flip.py d2play-20261108 town.plan=4 --why "town plan stage 4: sell and pickup scenarios green" --window 2h` | Doofer; row `trial` accepted (flip.py takes trial) |
| G3 nav trio | `tools/provingground/knobs3way.sh <id> Paladin "11 3"` for `nav.bloodmoor-coldplains`, `nav.barracks-return`, `nav.inner-cloister-cathedral`, `nav.catacombs-hop-33-34`, `nav.catacombs-hop-34-35`; ON arms no BLOCKED or RED that `off` lacks. Check caps first for `debug warptriggers` and `debug levelmap` (U); drop `nav.warpRead` and `quests.layoutRead` if absent | `python3 tools/flip/flip.py d2play-20261108 nav.fullGrid=1 nav.warpRead=1 quests.layoutRead=1 --why "map knowledge: nav A/B green on 5 areas" --window 2h` | Doofer; rows `dark` |
| G4 invdesk | add `invdesk.verbs` 1 to the `manage.*` scenario knobs, run `manage.equip-immediate,manage.identify-immediate,manage.recall-now` Paladin 11 and Sorceress 3 | `python3 tools/flip/flip.py d2play-20261125q invdesk.verbs=1 ui.managementFast=1 --why "invdesk verbs and management fast: manage.* green" --window 2h`, then `invdesk.instant=1` in its own window | Questy; rows `dark` |

## 4. Not flippable: needs code or the owner (no flip line)

- NEEDS CODE (one card each, V dossier section 1): `combat.minionRead` (census assertion, Necromancer sandbox only), `combat.deliveryUpkeep` scenario `combat.buffs-stay-up` (mode 2 is already live on Doofer without it, S), `loot.rulesEngine` (replay over recorded drops), `bot.zoneModel` (merc revive, k clamp), `bot.mercReviveUnverified`, `bot.cubeJobs`, `town.identifyTome`, `ladder.enabled`, `input.planner` (add Sorceress to the matrix), `input.hotkeyMap`, `quests.stepExec` Duriel/Tyrael (field-NPC talk).
- OWNER: `resource.pointsDecisions`, `resource.gearDecisions`, `agency.autoAlloc`, `loot.vault`, `watch.paletteFeed`, `items.ladderRunewords`, `agency.planner` mode 2.
- Re-flip of `enabled` rows (widening `bot.arbiterControl` 1 or `combat.deliveryExpect` to Questy): flip.py refuses them. Not wanted now. If wanted, the exact ledger edit is: in `data/darklaunch.json` set the row's `"status"` from `"enabled"` to `"trial"` (keep `enabled_at`), then run the flip line with `--window 2h`; the check accepts `trial` (V `python3 -I tools/darklaunch/check.py`).

## 5. RETIRE (two cards; ledger NOT edited here)

Both knobs are still read by code (V grep), so retiring the ledger row first would leave a marked default-0 knob with a retired row; the ledger edit goes in the same PR as the code removal.

### Card `retire-skillcap`: `resource.skillCapPerLevel` (retired 2026-11-27, TASK-1055)
Evidence: V ledger why_off "Kolbot rule not enforced by d2d or game.exe; off until the player model needs it"; enable criteria `U`; no consumer, no scenario. Steps: (1) `internal/points/legal.go`: delete `CodeLevelCap`, the `LevelCap` field and the level-cap check; (2) `internal/points/desk.go` line 38 comment and the field `LevelCap`; (3) `cmd/d2bot/pointswire.go` line 150 `LevelCap:`; (4) `internal/env/tuning_resource.go` line 57 and `internal/resource/tuning.go` (field in the `PointsApi, PointsDecisions, SkillCapPerLevel` list, line 61, and the `k("skillCapPerLevel", ...)` row, line 123); (5) tests: `internal/points/tables_test.go`, `points_test.go`, `internal/bot/kitlevel_test.go`; (6) regenerate `docs/knobs.md`, edit `docs/points.md`, `docs/kitgate.md`; (7) ledger: set the row `"status": "retired"` with a `note`; run `python3 -I tools/darklaunch/check.py`, `go vet` and `go test -p 1 ./internal/points ./internal/resource ./internal/env ./internal/bot ./cmd/d2bot` (whole repo through `labrun`). Size S. Check first (U): a saved tuning file that names the key must still load (unknown key rule of `internal/resource`).

### Card `retire-farmreset`: `bot.farmReset` (and `farmResetPct`, `farmResetCooldownMin`)
Evidence: V `docs/probes/farmloop-live-20261116.md`: ran 45 min, `farm reset:` never fired; V `docs/farmloop.md` section 3: levelsim gain 2.5 percent, "not clear". Wider than skillcap because the viewer has a zone-clear panel. Steps: (1) delete `internal/bot/farmreset.go`, `farmreset_test.go` and its calls in `task.go`, `goal.go`, `runner.go`, `boardmode.go` comment, `shadow.go` case, `tuner_test.go`; (2) `WhyFarmReset` in `internal/bot/whyline.go` and `internal/watch/whyline.json`, `whyline_test.go`, `docs/whyline.md`; (3) `internal/env/tuning_bot.go` fields `FarmReset`, `FarmResetPct`, `FarmResetCooldownMin` and the tuning rows; (4) `internal/goals/runner.go` `FarmResetOff`, `cmd/d2bot/glancewire_farmoff.go` and its test, `ladder.go`, `d2dbackend.go`, `live.go` call sites; (5) viewer: `internal/watch/zoneclear.go` (keep `bot.exploredPct`, drop the three farm keys and the order rule), `zoneclear_test.go`, `hero.go` Reset field, `hub.go`, `web/panels/zoneclear/panel.js`, `tools/uishots/fixtures_zoneclear.go`; (6) `tools/levelsim/farmreset.go`, its test and the `-farmreset` flag; (7) docs: `docs/farmloop.md` (status banner, then archive stub), `questboard.md`, `process/sandbox-devp9.md`, probes stay as history; regenerate `docs/knobs.md`; (8) ledger row `retired`. Size M. Needs an owner look because it removes a viewer setting (U whether the user uses the zone-clear panel).

## 6. Ledger status resets needed

None for the flips in sections 2 and 3: every row is `dark` or `trial` today (V). Resets are needed only for widening an `enabled` row (section 4). Proposed closing edits after the windows: `town.plan` and `combat.deliveryUpkeep` to `enabled` (coordinator; the card does not edit shared state).
