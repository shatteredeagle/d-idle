# Sorceress normal-act table: what exists, what is missing, how long (sorcplan, 2026-11-10)

> status: research, decides nothing, no code. Tags: V read in this worktree (file named), U judgement. Nothing run live. Answers: "how long to make the Sorceress normal30 table so friends can get Paladin AND Sorceress?"

## 1. Verdict

**The premise is stale. The table already exists.** `internal/plandata/build-sorceress_normal30.json` (clvl 2..30, class sorceress, combat lightning) was merged as TASK-144 `sorcnormal30` (V `docs/sorcnormal30.md`, `internal/plandata/sorcnormal30_test.go`, CHANGELOG "Sorceress now has a normal30 leveling table ... the setup wizard lists her without the fallback note"). The review (section 5.2, question 9) and `docs/live-checks.md` row TASK-147 still say it is missing; both predate or ignore the merge (V `docs/research/project-review-2026-11-10.md:206,270`; the ADR 0018 consequence line 41 is also the old text). The wizard lists every class with a `normal30` table, read from data (V `cmd/d2dsetup/wizardclass.go:12-56`), so the Sorceress is already offered; nothing blocks the wizard.

**Under a day: yes. About 6 agent-hours of offline work plus 3 daily host-hours of live measurement (elapsed 3 days because the exit criterion is "three consecutive daily fresh hours").** Making the table: 0 hours. The remaining work is making her first hour as good as the Paladin's evidence says it is, and answering Q9 on evidence.

## 2. What exists for the Sorceress today (V unless marked)

| piece | state | where |
|---|---|---|
| plan table clvl 2..30 | built, rows identical to approved v3 (Charged Bolt 2-5, Frozen Armor 5, Frost Nova 6, Charged Bolt 7-10, Telekinesis 11, Lightning 12, Charged Bolt 13-17, Teleport 18, Chain Lightning 19-30; 5 Vit a level, Str at 25/26/30); selected by `levelerBuild sorceress_normal30`; default for a fresh Sorceress is still v2 (`TestDefaultSorceressTableUnchanged`) | `plandata/build-sorceress_normal30.json`, `docs/sorcnormal30.md` |
| combat profile bands | `sorceress/lightning`: attack Chain Lightning, Lightning, Charged Bolt, Fire Bolt gated by `requires`, final band Charged Bolt, Fire Bolt, Ice Bolt; buffs Frozen Armor family + Energy Shield; layout f1..f8 | `combat/classdata/profiles.json` |
| profile chosen by the table | yes (`combat` key, no Go change); unknown class or profile is an error (palprobe part A) | `docs/classes/paladin-probe.md` |
| unlock ladder | `unlocks-sorceress.json`; test that no table row is refused at its event level | `plandata/unlocks-sorceress.json`, `TestNoBuildRowIsRefusedByAnUnlock` |
| chore defaults | potions 1:1, repair rare, town trip at mana potions < 3 | `plandata/chores-defaults.json:60` |
| belt | `ManaNeed(Sorceress)=2`; two mana columns (V palprobe row e1) | `resource/types.go:168` |
| Den floor | caster knob `campaign.denMinLevelCaster` (5) | `env/tuning_route.go:28` |
| readiness checks | class damage list and nightmare/hell level-gap rows for Sorceress; Normal rows are class-generic | `readiness/checks.json` |
| smoke gate rows | clvl floors minute 20 = 2, minute 45 = 3; `tools/smoke.sh` defaults to paladin,sorceress | `data/smoke/rules.json`, `docs/smoke.md` |
| measured pace | fresh Sorceress, 3 seeds, lockstep: clvl 4 at 60 min and 5 at 90 min (Paladin 5 and 6); after recovery/gold fixes XP at 90 min +34 to +47 percent, clvl 5 | `docs/soaks/classpacing.md`, `classpacing2.md` |
| zone leads | `proposezone.json` sorceress lead +1 (U note); classpacing says the data contradicts it (Paladin ahead one clvl) but left it | `decide/proposezone.json:15` |

## 3. What is missing (ranked by how much it matters to a friend's first hour)

1. **No live evidence beyond the first 90 virtual minutes (U risk, V gap).** Normal30 rows 12 to 30 (Lightning, Teleport at 18, Chain Lightning 19+) have never played on a fresh hero; `docs/live-checks.md` TASK-144 style boxes are open. Capabilities (V `docs/capabilities.md:33-35`): Chain Lightning/Lightning "known broken" on d2d (re-hit and cast-length deviations, fix not covering them), Frost Nova "experimental", Teleport proven live. So a friend's Sorceress at clvl 19+ fights with the weakest-evidenced skills. This is data-selectable: the build is a table, the fallback is the profile's Charged Bolt/Fire Bolt band.
2. **Mana wall clvl 1 to 8 handled only indirectly (U).** SME: 35 mana, Charged Bolt 3 mana plus 0.5 a point, 40 life; empty at melee range kills; out of potions early means stop and go to town (V review 4.4 item 6, `docs/sme/class-sorceress.md:80-81`). The bot's mitigations are the caster Den floor 5, the recovery loop, a 1:1 potion kit and the Den-first row (Frozen Armor at 5). There is no table row for Warmth (the SME: one point is the cheapest mana fix) and the approved v3 order is Charged Bolt-heavy; changing the order or adding Warmth is a table edit but a decision on a user-approved table (rows 2-30 approved 2026-11-01) and the open choice already noted in `docs/sorcnormal30.md`. Mana potions require gold; Sorceress time in Recovery was 10 to 20 minutes of the first 90 (V classpacing2).
3. **Profile bands start at the wrong place for early levels (U).** The first lightning band is gated on Chain Lightning (clvl 19+); before that bands 2 and 3 apply. It works (no failure to fight, kills 63 at minute 10) but the early attack order (Charged Bolt, Fire Bolt, Ice Bolt) is not SME-reviewed for clvl 1 to 8 (free Fire Bolt vs Charged Bolt mana). A review row, data only.
4. **No Sorceress-specific first-hour record.** The headline metric `docs/probes/firsthour.md` is Paladin-only by definition (V review R2); no Sorceress daily line, no per-class pace table with a pass line used as an exit criterion.
5. **Zone lead and recommended level.** `proposezone` lead stays +1 on a U note; classpacing data supports 4 at 60 min vs Paladin 5. Data edit after a second measurement.
6. **Tester profile.** No tester profile names `sorceress_normal30` (V `docs/sorcnormal30.md` "no tester profile code exists"); the wizard writes the config instead. Verify the wizard writes `levelerBuild` for a Sorceress pick (U: not read in this pass; one test).
7. **Not Sorceress-specific but visible to her:** equip failures (TASK-182), Blood Moor recovery farm (TASK-183).

Not missing: plan rows, combat profile bands, belt columns (two mana), kit/chore rules, unlocks, readiness rows, Den floor, smoke clvl row. The belt, gear gate (`geargate.go` caster triple) and merc policy treat her as a caster already (V `build-flexibility-audit.md` rows 27-28, still Go triples but correct for her). "Actplan gate row": no `internal/actplan` package exists; the act gate is `docs/research/sorc-act-gate.md` (draft, text) plus per-act floors in route code (U); not needed for normal Act 1 friends play.

## 4. Data versus Go

- **Data only:** Warmth/Energy row tweak or an SME-reviewed early table variant (a new `build-sorceress_*.json`, no Go: `registerDataTables`), early profile band, `proposezone` lead, `smoke` floors, chore/belt tuning, readiness rows.
- **Go:** none required for the friends beta. Unknown-class errors and per-class tables are in. If the first-hour measurement shows the Sorceress melee-rung fallback (a caster rung written for her, V palprobe f1) misbehaves, that is a small resource-policy change; not predicted.
- **Tests (two classes, F10):** table load and equality with v3 exist; add: the wizard Sorceress pick writes the table; a profile test that at clvl 1 to 8 the attack order has a castable skill; smoke rule row test for both classes (partly exists).

## 5. Work packages

| WP | what | agent-h | host-h | live |
|---|---|---|---|---|
| A | Correct the stale statements (review 5.2/Q9 text, ADR 0018 consequence, live-checks TASK-147 note) as proposed text; list what Q9 now needs | 0.5 | 0 | no |
| B | SME consult on clvl 1 to 12 Sorceress (Warmth point, Charged Bolt vs Fire Bolt vs Ice Bolt, potion kit, Den floor 5), output as data proposals; the user decides any change to the approved table | 1.5 | 0 | no |
| C | Apply the approved data (profile band, proposezone lead, smoke floors) with two-class tests; wizard-writes-table test | 2 | 0 | no |
| D | `tools/smoke.sh --class sorceress --minutes 60 --record` fresh hero, daily, three consecutive days; table in `docs/probes/firsthour.md` next to the Paladin | 1 (setup, then 0.3 a day) | 1 a day | yes, fresh character, zero setup writes, hidden instance, logged in `test-characters.md` |
| E | One 2-hour soak to clvl 12 plus a lockstep run past 18 for Teleport/Chain Lightning (fork binary); record `writes after start: 0` | 1.5 | 2 | yes |
| Total | | about 6.5 | about 5 (spread over 3 days) | |

Offline part (A, B, C): about 4 agent-hours, one day. Live part (D, E) is bounded by wall time, not agent effort.

## 6. Risks

- **Pace honesty:** Sorceress clvl 4 at 60 minutes against the Paladin's 5 and her slowest group (V). A friend picking her gets a slower first hour; the wizard note "mana-bound" should say so. Not a blocker, a message.
- **Evidence gap past clvl 12 and Chain Lightning on d2d** (section 3.1): offer the Sorceress as "early game only" or accept the capability row text; the review's own criterion is Paladin-only first-hour exit, so adding the Sorceress adds a second pass line, not a replacement.
- **Table change vs approved v3:** any Warmth/Fire Bolt line needs the user (open choice already recorded), so WP B ends in a question, not a merge.
- **Live capacity:** WP D/E share the worker host with the Paladin daily hour (one live job at a time).
- Not verified: wizard code path for the table write (item 6), any behaviour past minute 90.

## 7. Proposed text for the guard files (not applied)

- review 5.2 and Q9: "Paladin and Sorceress for friends once the Sorceress has three consecutive fresh first-hour records at or above the clvl floors of `data/smoke/rules.json` (minute 45: 3); the normal30 table exists since TASK-144."
