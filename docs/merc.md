# Mercenary policy (internal/merc)

Pure policy, no game, no UI, wired on d2d behind merc.enabled, see [merc-wire.md](merc-wire.md). Plan: docs/plan/plan-v2.md sections 3, 4 (M3 to M8), 10.1, 10.6. Sources: docs/sme/mercenary.md, docs/sme/d2decomped-findings.md section 2 (D2D, secondary), docs/merc-farm.md.
Tags: V verified (offline test or live), S secondary source, U unverified. Everything below marked S or U is a hypothesis for the live probe (plan M4).

## Packages
- `internal/merc`: types (`MercState`, `HireOffer`, `Item`, `Ctx`, `Decision`), `Reader`/`Actor`/`Backend`, policy (`policy.go`), gear (`gear.go`), formulas, log lines (`logs.go`), knobs (`knobs.go`).
- `internal/merc/sim`: fake `Backend` with gold accounting, the formulas, the free Rogue and gear loss on replace.
- Knobs: group `merc.*` in `internal/env/tuning_merc.go` (10 knobs, all NoHandicap, docs/knobs.md), converted by `merc.KnobsFrom(tuning.Merc)`.

## Policy and reasons
1. Free Rogue (`ShouldClaimFreeRogue`): Normal only, at Kashya, Blood Raven quest word done, no merc owned. A merc that is owned but dead still forfeits the reward (S), so an unknown state still claims (it costs nothing). The user decision (plan 10) is to take the free Rogue and never pay Kashya before the quest.
2. Paid hire (`ShouldHire`): nothing while any merc is owned. The Act 1 Rogue only when the bot skipped Blood Raven for good (`Ctx.BloodRavenSkipped`) and `merc.freeRogueOnly` is 0, and only with a Cold Arrow skill. Desert Mercenary only with an acceptable aura. Price rule: `price x merc.hireGoldMult <= gold - kit` (kit = TP, potions, antidotes first).
3. Aura table (`AuraPrefs`, best first): Normal and Hell Defiance, Prayer, plus Blessed Aim last and only for bow/javelin Amazons; Nightmare adds Might (physical classes) or Holy Freeze (Sorceress, Necromancer, Druid) in front. Hell rows have neither (V hireling.tsv per docs/town.md). Blessed Aim last is a U judgment (survival first). `AuraRank` gives the order so a Nightmare visit can upgrade Defiance to Might.
4. Replace (`ShouldReplace`): only Act 2, first Lut Gholein visit per difficulty (`Ctx.FirstLutVisit`), `merc.replaceAtAct2` on, gold rule as above, offer a Desert Mercenary with a better-ranked aura than the current one (a Rogue always loses; unknown current aura never replaces). A hire destroys the old gear (V plan section 3), so a living merc with gear returns `StripFirst`. A dead merc cannot be stripped (U: gear stays on him); replacing him loses it.
5. Revive (`ShouldRevive`): merc owned, known, dead; revive cap `merc.reviveMaxPerHour`; TP and potions present; gold >= `merc.reviveGoldMult` x price. More than `merc.stopDeathsPerHour` merc deaths in the hour raises the bar to `merc.stopGoldMult` x price. Price: the dialog quote if read, else `ReviveCost(level)` = `level^2/2 x 15`, integer halving first, cap 50,000 (S: 9 -> 600, 10 -> 750, 11 -> 900; pays carried gold then stash). Level 0 with no quote refuses (`level unknown`). `MercDied(was, now, inTown)` returns `LeaveArea` for a field death.
6. Boss gate (`MercAliveGate`, `AliveHook`): same numbers as internal/task/gate.go (dead merc and gold < 1000 raises the floor by 2). `AliveHook(reader)` has the signature of `task.MercAlive` and reports "unknown" for an unemployed merc, so the Andariel floor never rises for a character that has none yet. Tested against `task.MercAlive` assignment (hook_test.go). Not installed anywhere.
7. Gear (`EquipIfBetter`, `Score`): helm, armor, weapon (shield for Iron Wolf). Score = defense + 3 x life + (min+max damage) + 5 per socket, + 2 x resists in Hell only, x1.05 if ethereal (all U weights). Swap only at `merc.swapGainPct` (10 %) gain; an empty slot takes anything allowed with score > 0. Allowed kinds (S d2decomped): any helm and armor; Rogue bow, Desert spear/polearm, Iron Wolf shield or one-handed sword, Barbarian one-handed sword. Items must be identified and unbroken.
8. Requirements (`MeetsReq`): the MERC'S OWN level, and with `merc.reqStrDex` (default 1) his own Str and Dex (S d2decomped: the check uses the merc's stats, not the player's). UNVERIFIED live; docs/sme/mercenary.md says level only. With stats unread and the rule on, the check fails closed (`stats unknown`). Probe: see below.
9. Strip (`PlanStrip`): weapon, shield, armor, helm; `CellsNeeded` is the footprint (helm 2x2 + armor 2x3 + polearm 2x4 = 18); `StashCells` is what the player must stash first when `FreeCells` is short or unknown (-1).
10. Potions (`ShouldGiveHealing`): healing only, only below `merc.potionLifePct` (40) in a non-trivial fight, from potions the player has; `BuyPotionsForMerc()` is always false. Rejuvenation is not given (see UNVERIFIED 5).

## Log lines (logs.go, tested)
All start with `merc: `; `KindOf(line)` classifies them. Fields are key=value, spaces become `_`.
```
merc: hired class=338 aura=Defiance price=360 gold_before=1500 gold_after=1140
merc: seen class=271 alive=true life=87 level=9 aura=-
merc: revived price=600 level=9
merc: revive skipped (gold) price=600 gold=900      reasons: gold, revive cap, kit, stopped by deaths, level unknown
merc: equip slot=helm item=Skull_Cap reason=empty_slot
merc: strip slot=weapon item=Partizan cells=8
merc: reqcheck item=Skull_Cap ok=true reason=ok rule=level_only str=0 dex=0 level=9
```
Plan v2 names `merc_equip`/`merc_strip`/`merc_reqcheck`; this package uses the `merc: equip|strip|reqcheck` form from the brief, so one grep (`^merc: `) covers all of them.

## Backend interface (what d2d or internal/mem must implement)
`merc.Reader`: `State() (MercState, error)` (class, owned, alive, life %, merc level, aura if the caller kept it, own Str/Dex with `StatsKnown`, gear per slot with name/base/kind/sockets/ethereal/defense/damage/life/resists/requirements/footprint, `FreeCells`; `Known` false when unreadable) and `Offers() ([]HireOffer, error)` (rows of the open hire window: row, class, aura or arrow skill, level, price; error when no window is open).
`merc.Actor`: `Hire(ctx, row) (paid, err)`, `Revive(ctx) (paid, err)`, `Equip(ctx, slot, item)`, `Unequip(ctx, slot)`, `GiveHealing(ctx)`. Each verifies itself (gold delta, slot content) and returns the price actually paid so the log line carries the truth.
Context supplied by the caller (`Ctx`): gold (carried plus stash), kit gold, difficulty, act, player class, bow/javelin flag, Blood Raven skipped flag, first-Lut-Gholein-visit flag per difficulty, TP and potion flags, revives and merc deaths in the last 60 minutes.
Wiring later: call the policy from the town visit; install `task.MercAlive = merc.AliveHook(r)`; map `bot.mercReviveUnverified` to "revive path verified".

## UNVERIFIED (probe list, plan M4; throwaway character, ledgered cheats only)
1. Hire window rows: layout, scroll, whether level/price columns read as the formula says (`OfferedLevel`, `HireCost`: S), whether Kashya lists rows before Blood Raven dies, and whether the free-reward row shows 0 gold or a flag. Hire payment source (carried vs stash).
2. Revive dialog line, which NPCs offer it, the price against `ReviveCost`.
3. Str/Dex rule: put an item the merc fails by Str only on him and watch the equip result (`merc: reqcheck` records the rule used). One-line probe.
4. Offered level range clvl-5+rand(5) (S) against the live rows at several clvls.
5. Rejuvenation to the merc: d2decomped lists only healing, antidote, thawing for Shift+belt; docs/sme/mercenary.md says rejuvenation works. Shift+belt-key a rejuvenation on a hurt merc, read potion count and merc life.
6. Score weights, the 10 % swap gain and the aura order are judgments, not data.
7. Whether a dead merc's gear can be taken off, and whether a merc that left the area (not in the unit table) still reads `Owned`.
