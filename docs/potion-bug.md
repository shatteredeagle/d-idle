# Potion bug ("none drunk") triage, 2026-10-30

Evidence: Fresha (i5, Game.exe backend, build a920fef) soak log `soak24-i5/runs/20261030-120747-d2bot.log`; Soaky i4 logs `soak24/runs/20261030-015906` and `-082115` (old build). All V (read from the logs); causes marked below.

## Fresha deaths, Dark Wood (clvl 8, 113 HP, belt 4 hp 3 mp)
- 12:56:55 portal into Dark Wood. 12:57:02 to 12:57:45 fights Carvers, `combat fight minlife=66 potions=1`, one mana drink, life 97% at 12:57:20.
- 12:57:45 crowd 15, life 66%, `crowd: stepping away`; 12:57:47 `town portal cast` ok (1256 ms), state to resupplying. No potion line for the next 9 s while 28 hostiles stood around (`status life 0 ... stuckFor 9 s`). 12:57:56.73 first `combat potion healing col 1 life 0` (too late), death 12:57:56.9. Autopsy: not tagged (2 potions in the last minute).
- Death 1 corpse: not retrieved (worth 712 < 1000). Town: gear equip failing ("item still on the cursor"), vendor dialog ignoring clicks, no gold, 0 scrolls.
- 13:00:02 `back to field` portal = the same Dark Wood spot, then the gold sweep ("Sweeping the death spot for gold", expected 644). 13:00:07.07 `crowd: stepping away life 20%` with 21 in the crowd, 13:00:07.98 dead: 5.7 s after arrival, 0 drinks. Autopsy tagged BUG (belt 2 hp, 0 drunk in the last minute).

## Soaky i4 (same tag, other cases)
- Stony Field 12:55:33: safety chose TownPortal (action 4); three cast tries failed ("no portal appeared", "inventory panel did not open", about 6 s), no drink in between, dead with 3 healing potions in the belt, state recovering (gold sweep again).
- Tamoe Highland 02:30:18: drank at 50% and at 0% (3 s apart, 22 hostiles, 2 drinks one second gap), yet autopsy said 0 drunk: the i4 build did not count them (`potionsUsed` stays 0 in that log).

## Root causes
1. (V from logs plus code reading, not reproduced live) A town portal cast is a blocking sequence (up to 3 tries x about 2 s) that the guard cannot interrupt, and Decide prefers TownPortal over a potion when not surrounded (crowd.go). Life drains with no drink. FIXED: `Fighter.drinkIfDue` before every try (combat/fighter.go, test `TestOpenPortalDrinksBetweenTries`).
2. (V) Fresha death 2 is not a failed drink: the bot re-entered the portal into the pack that had just killed it, for a gold pile. A 113 HP character loses 80% in under a second from 25 Carvers. The tag is a false positive in the sense that no decision point was reached. FIXED: `sweepDecision` skips the sweep when the autopsy saw >= CrowdN hostiles (goldsweep.go, `TestGoldSweepSkips`). The back-to-field portal after a death with a corpse run decision of "no" is not changed (it is only used by the sweep and the normal return).
3. (U) 9 s without any Safety result in death 1 after the portal opened. Candidates: ToTown enterPortal GoTo through the crowd, `stuckFor 9 s` says the character did not move. Not provable from the log; the new skipped-drink line will show which rule held the drink back.
4. The tag counts drinks since the trail start (60 s) so a character that spent 5 s in the field gets BUG. Gate semantics unchanged (cmd/d2soak ORs the same rule, spec death-rule); added `fieldSecs` and `lowLifeSecs` to the autopsy and the summary so a verdict can tell "no time" from "never decided". Proposal for the coordinator: require `lowLifeSecs >= 2` in the gate.
5. Tamoe: the i4 build undercounts drinks (old build, not investigated).

## Log line
`potion: not drinking because <reason> life=<n>% belt=hp=.. mp=.. rv=..` (reasons: no healing potion in the belt, potion cooldown, chose <action> first, no rule asks for a drink), at most once per reason per 2 s, only in a fight under `combat.drinkLifePct`.

## Watch live
Grep the soak log for `potion: not drinking`, `combat potion`, `death autopsy` with `fieldSecs`/`lowLifeSecs`, `death: gold sweep skipped`. Cursor-stuck equip clicks ("item still on the cursor") are a separate Game.exe-path problem seen on Fresha; a potion hotkey with an item on the cursor is U (not tested).
