# LOOTSKIP: why ground potions were left (card lootskip)

> status: current. Source: container log of the sandbox soak (Paladin clvl 10 to 14, build 0.0.261110171517), read-only, plus code read and offline tests.

## Findings
- V (log counts, whole container log): 1134 `loot: N seen` lines; 320 name "keep [C] potion already in the pack" among their skips, 109 "keep [C] town supplies". 388 `loot: potion skip` lines (hp1/hp2/mp1/mp2): 209 with beltFree <= 0, 173 with beltFree > 0 (88 of those with an empty pack).
- V: "keep [C] potion already in the pack" is only the label of the catch-all rule `spare supplies` (tier C). It is what ANY potion gets when the "healing and mana" rule (needed: potion) says no. The summary line does not name the item.
- V (offline, real game tables from the container cache, test TestPaladinSashHealingOnGroundWanted): Paladin clvl 14, sash, 1 healing in column 0, 2 mana in column 3 (the state logged at 17:20): hp1, hp2, mp1 are all WANTED. So the logged skips at that moment were NOT healing/mana potions the belt could take.
- S (code): the skipped things there are potions the rule never wants: stamina, antidote, thawing (types spot/apot/wpot, by design, SME agrees), or, when the belt boxes of that kind are full, healing/mana with pack buffer 0 (`loot.potionBuffer` was 0). `beltFree=-4` lines are a full 8-box belt read with 4 slots (belt not in the equipment read); same outcome.
- U: no log line names the skipped item or says it was a healing potion; whether the user's potion was a healing one on a full-for-healing belt (mana column had the free boxes) or a stamina/antidote cannot be told from the log. No "failed before"/pick-failure lines exist: nothing was attempted and failed.
- So: not a bug in the rules, a policy gap: with the belt full for a kind the bot carried zero spares, and the poverty loop (no gold to rebuy) made that costly.

## Decided rule (user spec 2026-11-10 plus d2-sme advice, tags EXP = expert opinion, not verified)
1. Potions: take healing/mana/rejuvenation while the belt (class columns, `belt.WillBeltFor`) takes them, plus a pack buffer of `loot.potionBuffer` per kind. DONE: default 0 to 2 (data knob; belt columns stay per-class data in resource.PlanFor). SME: total 3-4; Paladin 2 healing + 1 mana, Sorceress 2 healing + 2 mana (EXP). 2 per kind is the simple bound; a per-class buffer is NOT done.
2. Rejuvenation: own rule, up to `loot.rejuvSpare` (3) total (existing).
3. Tier floor by clvl (SME EXP; DONE in TASK-262, data in `internal/loot/potionfloor.json`, applies to healing and mana, not rejuvenation; unverified live; the vendor-stock check is still open): Normal clvl 1-12 tier 1+, 13-25 tier 2+, 26+ tier 3+; Nightmare tier 3+ (4+ from clvl 40); Hell tier 4+. Minor healing is not worth a belt box from clvl 30. Needs a data table in resource/belt bands plus the vendor-stock check (SME UNSURE).
4. Stamina/antidote/thawing: skip (unchanged).
5. Scrolls (user spec: pick only until the tome is full, no loose scrolls once a tome exists): DONE in TASK-262: with the tome in the pack `needsOf` reads its quantity stat (70) and wants the scroll while qty < `loot.TomeCap` (20); an empty tome (no stat list) or an unreadable stat counts as 0. Offline tests only, UNVERIFIED live. SME with no tome (EXP): buy a Town Portal tome first (about 450 gold, UNSURE), until then 2 loose TP and 2-3 loose ID scrolls.

## Also proposed
- DONE (TASK-262): the `loot:` summary names the skipped items per reason (`2x no room: Cap, Sash`, at most 3 names then `+N more`).

## The ground Cap (screenshot, Blood Moor)
- V (log): the hero wears "1:Cap" in every `sell: worn before/after` line (16:14 to 17:06), so the Cap was never taken off by the bot; the broken Cap of the Repair panel is the worn one. `inventory contents` at 16:18 held a spare `cap x1`, and the bot sold Caps in town (sell: sold Cap 19/32/9 gold). No log line names a Cap among "item left on the ground" (78 such lines, none a Cap): not a pack-full skip.
- S (code + log shape): a plain (normal quality) Cap evaluates to "drop [C] plain or inferior item" (seen in the loot summaries); a Cap the hero sold or dropped earlier on the same ground is then ignored by design. Whether this ground Cap was the hero's own dropped one or a fresh drop cannot be told: the loot summary does not name skipped items (see "Also proposed").
- U: no evidence of a re-target loop or a failed pick on it.
