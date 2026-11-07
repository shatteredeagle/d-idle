# Fidelity cards: backlog map

> status: map, 2026-11-07, user go 2026-11-07 ('build backlog cards for everything that needs to be checked'). Source: [fidelity-matrix.md](fidelity-matrix.md) (78 rows) and [fidelity-probe-design.md](fidelity-probe-design.md). Cards are Backlog.md tasks (`backlog/tasks/`), created Proposed; the two probe cards that already existed (TASK-55 `fidprobe-core`, TASK-56 `fidprobe-vendor`) only received labels. Rows not listed here are closed in the matrix (oracle 0 differences, identical data table: OD 18, DT 4). Sizes S/M/L and live-time flags are estimates (U).

Labels: `fidelity` on every card; ONE kind label (`probe` same-save probe, `oracle` differential vs original code, `readonly` live memory read check, `d2d-measure` d2d-only measurement, `model-check` model/adapter row, `upstream-pr` fix is an upstream d2d PR, `bot-fix` fix is a bot change, `drill` live drill); one or more d2d area labels (`d2d-combat skills monsters items loot town vendor movement pathing levels network time data ui`); tier/area `gate`, `tooling`, `soak` and `parked` as needed. Where the real fix path is an upstream PR the description says so even if the kind is `oracle` (the proof comes first; the PR needs the proof standard docs).

## Cards

| card | task | matrix rows | kind | d2d areas | live? | size | fix | deps | other labels |
|---|---|---|---|---|---|---|---|---|---|
| fid-probe-readers | TASK-72 | ST07 IT04 ME06 | probe | d2d-items, d2d-data | no | M | bot change (tooling in internal/fidprobe) | TASK-55 | tooling |
| fid-probe-equip | TASK-73 | IT03 IT07 | probe | d2d-items, d2d-ui | yes | M | upstream PR if d2d skips requirement checks (world.hpp:808, fight.hpp:819) | TASK-55 (scenarios 3, 4) |  |
| fid-probe-regen-speed | TASK-74 | ST04 ST05 ST08 | d2d-measure | d2d-combat, d2d-movement, d2d-time | yes | M | upstream PR if a deviation is proven (potion double fill missing 0/24) | TASK-55; user answer to matrix open question 4 |  |
| fid-xp-award | TASK-75 | ST10 | readonly | d2d-monsters, d2d-combat | yes, one short kill count on game.exe | S | none expected (Den +60 % was composition) | none |  |
| fid-price-pr | TASK-76 | VN01 VN02 VN03 IT08 VN06 | upstream-pr | d2d-vendor, d2d-items, d2d-data | no | M | upstream d2d PR (price-trace.md Q1,Q2,Q5, pr-vendor-repair-gate-draft.md) | none (live confirmation by TASK-56) |  |
| fid-stock-oracle | TASK-77 | VN04 VN05 | oracle | d2d-vendor, d2d-items, d2d-loot | yes only for the TASK-56 stock read; the oracle itself is offline | L | upstream PR if stock generation differs | TASK-56 (stock lists), probe core |  |
| fid-belt-pr | TASK-78 | IT09 | upstream-pr | d2d-items, d2d-loot | no | S | upstream PR (pr-belt-draft.md) and retire shim R04 in the bot | lab-binary rebase for the shim |  |
| fid-inv-limits | TASK-79 | IT10 | oracle | d2d-items | no | S | none expected | none |  |
| fid-layout-act1 | TASK-80 | MA04 | oracle | d2d-levels, d2d-pathing | no | L | upstream PR where the DRLG differs (0 of 102 outdoor ports verified) | none |  |
| fid-transitions | TASK-81 | PR09 | probe | d2d-levels, d2d-movement | yes | S | bot shim R05 until a d2d fix | TASK-55 |  |
| fid-monster-move | TASK-82 | MA12 | oracle | d2d-monsters, d2d-pathing, d2d-movement | no | M | upstream PR (F08 monster velocity D1 | none |  |
| fid-missile-player | TASK-83 | SK03 SK12 SK08 | oracle | d2d-skills, d2d-combat | no | L | upstream PR where flight or pierce differs | none |  |
| fid-missile-monster | TASK-84 | MA10 | oracle | d2d-monsters, d2d-combat | no | M | upstream PR (F14 missile block gate, MissA to-hit) | none |  |
| fid-boss-mods | TASK-85 | MA06 | upstream-pr | d2d-monsters | no | S | upstream PR (pr-mods draft) | none |  |
| fid-boss-skills | TASK-86 | MA13 | oracle | d2d-monsters, d2d-skills | yes for the last step only | L | upstream PR where skills or death triggers differ | fid-boss-mods |  |
| fid-status-effects | TASK-87 | MA11 SK08 | oracle | d2d-monsters, d2d-combat, d2d-skills | no | M | upstream PR (freeze rule C3 to C5, 22 state deviations) | none |  |
| fid-sorc-upstream | TASK-88 | SK05 SK06 SK07 | upstream-pr | d2d-skills, d2d-combat, d2d-time | no | M | upstream PR from private branches (local/sorc-mana-timing-r2, shims R01, R03) | lab-binary rebase |  |
| fid-skill-damage-pr | TASK-89 | SK02 | upstream-pr | d2d-skills, d2d-combat | no | S | upstream PR (pr-skill-damage draft) | none |  |
| fid-teleport | TASK-90 | SK11 | oracle | d2d-skills, d2d-movement | no | M | upstream PR (batch 2 D1 room, D2 blocked target | lab-binary rebase |  |
| fid-hit-recovery | TASK-91 | SK16 | oracle | d2d-combat, d2d-skills, d2d-time | no | M | upstream PR (get-hit rule, length, render drafts | none |  |
| fid-lab-rebase | TASK-92 | SK05 SK06 SK07 SK11 IT09 (all private-branch DEV rows) | d2d-measure | d2d-data, d2d-time | yes | M | fork work, no bot change | none |  |
| fid-quest-den-raven | TASK-93 | PR01 PR02 | probe | d2d-monsters, d2d-town, d2d-levels | yes | M | upstream PR if a trigger or reward differs | TASK-55 (scenario 11) |  |
| fid-quest-andariel | TASK-94 | PR04 | drill | d2d-levels, d2d-monsters | yes | L | bot change if the quest FSM differs | TASK-55; fid-boss-skills | gate |
| fid-wp-tp-pr | TASK-95 | PR06 PR07 | upstream-pr | d2d-levels, d2d-town, d2d-movement | no | S | upstream PR (pr-wp draft | none |  |
| fid-death | TASK-96 | PR08 | d2d-measure | d2d-combat, d2d-items | yes | M | upstream PR if the corpse list or floor differs (d2d many bodies, original one) | user answer to matrix open question 4 |  |
| fid-gatea-ab | TASK-97 | EN03 | d2d-measure | d2d-time, d2d-combat, d2d-monsters | yes | L | bot or upstream depending on the gap | fid-lab-rebase; fid-probe-readers; slices 1 to 3 | gate, soak |
| fid-adapter-rows | TASK-98 | SK14 IT06 VN08 VN09 ME06 EN01 | model-check | d2d-ui, d2d-town, d2d-time | no | S | none | none | parked |
| fid-lowvalue | TASK-99 | SK10 SK15 ME05 EN02 PR03 | model-check | d2d-skills, d2d-monsters, d2d-time | no | S | none | none | parked |
| fid-matrix-refresh | TASK-100 | all rows of fidelity-cards.md | model-check | d2d-data | no | S | docs only (owner of docs/research/fidelity-matrix.md since TASK-54 is Done) | after each closing card | tooling |
| fidprobe-core | TASK-55 | ST01 ST02 ST06 ST07 IT03 IT04 PR05 EN04 (scenarios 1 to 14) | probe | d2d-skills, d2d-items, d2d-network | yes | M | bot tool | none | gate |
| fidprobe-vendor | TASK-56 | VN01 VN02 VN03 VN04 VN05 | probe | d2d-vendor, d2d-town | yes (needs input, open question 4) | M | bot tool | TASK-55 | gate |

Row coverage: every matrix row with state DEV (21), NONE (13), DM (7), RO (9) and MODEL (6) is in a card; ST01, ST02, ST06, PR05 and EN04 are closed by TASK-55 itself, VN01 to VN05 live parts by TASK-56. Parked: fid-adapter-rows (MODEL rows) and fid-lowvalue (SK10 SK15 ME05 EN02 PR03: no gated metric changes, matrix open question 2).

## Label counts (new cards, 29)

| kind | cards |
|---|---:|
| probe | 4 |
| oracle | 10 |
| readonly | 1 |
| d2d-measure | 4 |
| model-check | 3 |
| upstream-pr | 6 |
| bot-fix | 0 |
| drill | 1 |

| d2d area | cards |
|---|---:|
| d2d-combat | 10 |
| d2d-data | 4 |
| d2d-items | 7 |
| d2d-levels | 5 |
| d2d-loot | 2 |
| d2d-monsters | 10 |
| d2d-movement | 5 |
| d2d-pathing | 2 |
| d2d-skills | 8 |
| d2d-time | 7 |
| d2d-town | 3 |
| d2d-ui | 2 |
| d2d-vendor | 2 |

Dependencies named in the cards: probe core TASK-55 (offline engine built, not yet run live), vendor probe TASK-56, lab-binary rebase (fid-lab-rebase), user answer to matrix open question 4 (input on the hidden game.exe instance).
