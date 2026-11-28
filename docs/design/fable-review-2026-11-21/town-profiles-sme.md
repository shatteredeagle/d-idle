# Town supply profiles: SME review (card town12)

> status: SME advice written 2026-11-27 (section 3); no number changed. The d2-sme agent fills section 3; advice only, no number in `internal/supply/profiles.json` changes in this card.

## 1. What is reviewed

`internal/supply/profiles.json` (the card says `data/supply/profiles.json`; the file lives under `internal/supply` because it is embedded). One row per excursion profile, lines healing, mana, portal, gold_carry. Every number is a U judgment copied from `internal/task` (route.go, gate.go, goal.go) and resource/recovery.go (town-planner appendix B2). `manaNeedAtLeast: 2` means the mana line applies only to a build that leans on mana potions (build profile data).

| Profile | healing min | mana min | portal min | gold_carry min |
|---|---|---|---|---|
| field | 2 | - | 1 | - |
| monastery_road | 4 | 2 | 2 | - |
| forgotten_tower | 4 | 2 | 2 | - |
| blood_raven | 4 | 2 | 2 | 250 |
| tools_of_the_trade | 4 | 2 | 1 | - |
| countess | 4 | 2 | 1 | - |
| andariel | 8 | 3 | 1 | 3000 |
| den | 2 | - | - | - |
| recovery_farm | 2 | 2 | 1 | - |

Targets (what a visit fills to when gold allows) come from the belt plan and pack buffer, not from this file.

## 2. Brief for the d2-sme run (copy as the agent prompt)

> Read-only, advice only. Review the numbers in section 1 of `docs/design/fable-review-2026-11-21/town-profiles-sme.md` as a veteran Diablo II LoD 1.14d player would, for a bot that plays Sorceress (mana potions used) and Paladin (mana need 0) builds from normal difficulty through Andariel. For each profile and each line say: keep, raise or lower, with the number you would use and one sentence of reason (normal vs nightmare differences if they matter; tag each reason V/S/U: verified in play or a source, secondary, or opinion). Cover specifically: (a) healing minimum 4 for a 4-slot belt versus the 8 for Andariel; (b) mana 2 and 3 for a caster; (c) portal 1 versus 2 scrolls (one to leave, one to come back, and the tome case); (d) gold_carry 250 for Blood Raven (repairs, revive) and 3000 for Andariel (what the gold would pay for); (e) whether Den and the field profile need a portal line; (f) any missing profile (Cave, Cold Plains, Stony Field, Tristram, Cathedral, Duriel) a player would stock differently. Write the advice into section 3 of that file as a table (profile, line, current, advice, reason, tag). Change no number in any other file, create no other file, report in under 200 words.

## 3. Advice

Verdict: depends. The shape (min per stop, more for Monastery road and bosses, nothing for Den) is what a player does; the weak spots are the 8-healing Andariel row against belt capacity, mana for a caster at Blood Raven, and the missing profiles. Tags: SRC (repo doc, path given), EXP (veteran play, not looked up), UNSURE. Classic 1.14d. No source was fetched this run; SRC means an existing doc in this repo.

### 3.1 Ranked changes (data only, nothing applied)

| # | File / field | Current | Proposed | Why | Conf. |
|---|---|---|---|---|---|
| 1 | profiles.json `andariel.healing.min` and `.mana.min` | 8 / 3 | cap by belt slots: with a 12-slot belt keep 8 + 3 (11 slots); with an 8-slot sash/light belt use 6 healing + 2 mana; below 8 slots do not run the profile. Needs a belt-slots condition in the data (like `manaNeedAtLeast`) | A belt has 4 columns and 1 to 4 rows (start belt 4 slots, Sash/Light Belt 8, Belt 12; SRC `docs/sme/pace-and-formulas.md` section 5). 8 + 3 does not fit an 8-slot belt, so the line can never be met and the visit stalls or short-buys | high |
| 2 | `blood_raven.mana.min` | 2 | 4 for a mana build | Blood Raven is a mana race for a Sorceress (50% resist on Ice/Fire Bolt); SRC `docs/sme/blood-raven-caster.md` "4+ mana potions are the real kit", practical gate 4+ healing, 1+ mana | high |
| 3 | `andariel` (new line, not in schema) antidote | none | 2 antidotes, plus potion tier floor Light/Healing | Andariel's poison is the killer in Normal at clvl 15-22; SRC `docs/sme/audit-bot-vs-human.md` C5 and E2, `pace-and-formulas.md` boss kit "8 healing and 2 antidotes". Minor potions (30 hp) are useless there, so a count-only line is not enough: add a tier floor (UNSURE which tier the Akara/Fara stock offers at clvl 18, check the vendor listing live) | med |
| 4 | new profile `duriel` | absent | healing 8, mana 4 (mana build), portal 2, thawing 2, gold_carry 4500 | Tomb fight: cold slows, run speed loss kills at low belt; Thawing is the standard prep (SRC `audit-bot-vs-human.md` E3, `pace-and-formulas.md`). Revive cost at merc level 24 is about 4300 by the formula `min(int(l*l/2)*15, 50000)` (SRC `mercenary.md`, D2R page, UNSURE for 1.14d), hence 4500 | med |
| 5 | new profile `cathedral_catacombs` (also Tristram, Black Marsh) | absent | healing 6, mana 2, portal 2 (Tristram: healing 4, portal 2) | Catacombs mlvl 11-12; `pace-and-formulas.md` line 37 says 4 to 6 healing, 1 to 2 TP. These are deep dungeons, a walk home is 10 minutes, so the 2nd scroll is worth 100 gold (EXP) | med |
| 6 | `countess.portal.min` and `.healing.min` | 1 / 4 | 2 / 6 | Forgotten level 5 (Countess) is the farthest point from town of any Act 1 stop and the Countess sends you back (EXP: two or three retreats are normal at clvl 14 to 16, a second scroll saves a long walk). A portal pair stays open until recast, so one scroll covers one trip out and back; each extra retreat needs another scroll | med |
| 7 | `field.healing.min` | 2 | 2 below clvl 8, 4 from clvl 8 (clvl-banded) | 2 minor potions is fine at clvl 1 to 5 (Blood Moor, Cold Plains, Cave); at Stony Field/Dark Wood (monster level 4-6, clvl 8+) a human keeps a column of 4 (SRC `pace-and-formulas.md` section 5 "Kit: 4 to 6 healing"). Needs a clvl band in the data; if that is too much, set 3 | low |
| 8 | `priority.json` order: `gold_carry` (6) vs `merc_revive` (8) | gold_carry above merc_revive | move merc_revive above gold_carry when a merc is hired | The only reason to hold 3000 (Andariel) is to pay a revive/repair; spending the reserve on a revive now, when the hireling is dead, is the same money. As written the bot keeps gold for a revive while refusing to pay it. UNSURE: depends how merc_revive is modelled | low |
| 9 | `recovery_farm.healing.min` | 2 | 3 | After a death the char is at clvl-corpse risk; 2 is fine for Den-class areas (mlvl 1-3) but recovery is often the very area that killed it. EXP | low |
| 10 | `blood_raven.gold_carry.min` | 250 | keep 250 before the free Rogue; 600 if a merc is already hired | The free Rogue arrives after the quest (SRC `mercenary.md` section 2), so before it no revive is needed and 250 covers potion refill plus repair. If a paid Rogue exists, revive at merc level ~8 is about 480 (D2R formula, UNSURE) | low |

### 3.2 As written is fine

- `den` healing 2, no portal line: Den is next to the camp (Blood Moor, area level 1, SRC `pace-and-formulas.md` section 4), mlvl 1 to 3, walking home is cheaper than a 100 gold scroll. Same for Cave and Cold Plains: reuse `den`/`field`.
- `field` portal 1: one scroll covers out and back (the portal pair persists); the "tome" case: a Tome of Town Portal (20 charges, 450 gold at clvl 3, SRC `audit-bot-vs-human.md` C4) should count as 20 scrolls for every portal line; make sure the planner counts tome charges, otherwise it keeps buying scrolls (verify live: log line `town: portal` count vs tome charges).
- `monastery_road`, the Forgotten profile: healing 4, portal 2, mana 2: matches the Catacombs-class advice (4 to 6 healing, 1 to 2 TP).
- `tools_of_the_trade` portal 1: it is a quest-room walk with a short retreat; fine.
- `andariel.gold_carry` 3000: right size for one revive at merc level 15 to 20 (1700 to 3000 by formula, UNSURE) plus a repair visit; keep.
- Priority order (healing min, portal one, repair, ammo, mana, gold_carry, portal min, merc_revive, then targets): sensible, because life potions and an exit are what keep a character alive; the one change is item 8. Repair at 3 above ammo and mana is right (EXP: a broken weapon is worse than an empty mana column).
- `mana.manaNeedAtLeast 2`: right, a Paladin before Zeal buys no mana potions (SRC `pace-and-formulas.md` section 5).

### 3.3 Gaps (not in the data, say so to queue research)

- No scroll lines beyond TP: an identify scroll/tome (Tome of Identify 1, scrolls 0 to 2) matters only once the bot identifies rare jewelry, see `audit-bot-vs-human.md` H1; keys are not needed for the Normal route (UNSURE which vendor sells them in 1.14d; do not add).
- No merc potion line: mercs take healing potions dragged onto the portrait or shift-clicked; mana potions do nothing (SRC `mercenary.md` section 3). A human keeps 1 to 2 spare healing potions for the merc in Act 2+; not modelled, recommend it as a later line, low value in Normal.
- No repair threshold in these files: human practice is repair below 20 to 40 percent durability (SRC `pace-and-formulas.md` section 6); verify the planner's threshold lives in the repair data and falls in that band.
- No potion tier floor in any line (count only); biggest hidden risk at clvl 15+ (item 3). Vendor stock by act is a gap in `docs/sme/`.
