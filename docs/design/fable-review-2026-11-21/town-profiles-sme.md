# Town supply profiles: SME review (card town12)

> status: brief ready, review not run (2026-11-27). The d2-sme agent fills section 3; advice only, no number in `internal/supply/profiles.json` changes in this card.

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

Not yet run.
