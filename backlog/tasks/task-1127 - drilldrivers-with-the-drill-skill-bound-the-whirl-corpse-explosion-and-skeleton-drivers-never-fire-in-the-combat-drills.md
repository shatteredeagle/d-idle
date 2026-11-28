---
id: TASK-1127
title: 'drilldrivers: with the drill skill bound, the whirl, corpse-explosion and skeleton drivers never fire in the combat drills'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-28T07:26:00Z'
labels:
  - bot
  - bug
dependencies:
  - TASK-1105
priority: medium
ordinal: 535000
card: "drilldrivers"
lane: next
owns:
  - "internal/combat/"
  - "tools/provingground/scenarios/combatdrills.json"
  - "tools/provingground/scenarios/necro.json"
  - "tools/provingground/golden/setup-played.tsv"
exclusive: []
contracts: []
result: "drv2, offline only, NOT verified live or in the lab (no rerun). Built: one `combat hold driver=<whirl|escape|minion|corpse-explosion> ticks=N why=<reason:ticks>` line per fight at knob 2 (internal/combat/holds.go; tests Barbarian and Necromancer, not build-specific, no data added). Classified from the 2026-11-28 run dirs: whirlwind-clears = setup (S: about 39 mana, buffs and Bash paid first, a leg costs 25; scenario now writes mana 200, U); corpse-explodes, necro.corpse-ce1/ce5 and army-holds = bot defect, open (V logs: Teeth 28 to 33 casts, mana 0 to 6 percent at every fight end, so no raise and no explosion is ever affordable; the summoner tables with a Teeth point starve the army, the sme table without Teeth is GREEN; fix proposed: the attack leaves the mana of the next wanted summon, a profile field); hold-interruptible and teleport-reposition = scenario fault (teleport-reposition gets pre.life_pct 35, U; hold-interruptible stays open: its offline prep test has no debug hit verb); pets-hold = same assertion as army-holds, not triaged; hammer-in-pack = setup BLOCKED outside this card (no quest point for Holy Bolt) and the act= assertion needs the profile row enabled, shadow line accepted. Arbiter soak 14 against 9: expected, see notes."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Lab reruns 2026-11-28 on main 28216bd08 (nf1b pre.bind and pre.summon merged; run dirs provingground-out/20261128-022149, -022229, -022258, -022322, -022352, -022436, -022528 of the labops2 lab checkout). The setup side now holds (prep self-check passed, tags hand-skill and summon), so the verdicts are about the drivers. (1) combat.whirlwind-clears Barbarian 11 RED: Whirlwind 1 point bound right, combat.deliveryUpkeep 2, 98 kills by Attack 42, Double Swing 47, Bash 9; no 'combat move' line and no 'move shadow' line; minlife 100. The gates of whirlPlan (moveRow, life floor, pressMayStart, whirlLeg pack 3 in radius 4) log nothing when they hold, so the cause is U. (2) combat.corpse-explodes Necromancer 11 RED: Corpse Explosion 3 points bound right, kills by Attack 69 and Teeth 13, no 'combat corpse-explosion' line. (3) necro.corpse-ce1 and ce5, seeds 11 to 13, RED x6: 'minions 1' (the Clay Golem of the setup); hero_casts Teeth 28 to 31, Amplify Damage 2, Bone Armor 1, Clay Golem 1; Raise Skeleton has 8 points (skill_levels) and is never cast with 85 kills of corpses around. (4) combat.hold-interruptible Paladin 11 RED only on '^combat potion' (Zeal is now used: 25 kills with last_skill Zeal) and combat.teleport-reposition Sorceress 11 RED on the escape line: every fight has minlife 100, twelve to fourteen Fallen do not hurt a clvl 24 hero, so these two are scenario faults. Hints from the same night (main 28216bd08): necro.upkeep-pack is GREEN with build necromancer_summoner_sme (golem and skeletons raised) and necro.revive-after-death raises skeletons too (Raise Skeleton 3 casts, 5 minions; RED only on the Revive line), while ce1/ce5 use build necromancer_summoner_normal30: compare the two build rows first. Deferred block: combat.point-spell-hits GREEN (Sorceress and Druid); combat.traps-stand Assassin 11 RED (28 casts of Fire Trauma, no 'combat trap' line with Charged Bolt Sentry 4 and Death Sentry 1 learned); combat.army-holds and combat.pets-hold RED on '^combat minion' only (the stance summary line IS logged, kind behindArmy; the scenarios set combat.deliveryUpkeep 2 but not combat.minionRead, U whether the minion line needs it); combat.hammer-in-pack and its control BLOCKED at setup ('build paladin_hammerdin_guide at clvl 24: main attack Blessed Hammer gets no point from the plan (banked: 0 skill points)'). CHEATED setups, mechanics only, not live-verified. Evidence table: docs/probes/lab-night-results-2026-11-28b.md. Done when the criteria hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A fight in mode 2 logs why a learned move or corpse row holds (one line per fight, not per tick), so a silent driver can be read from the log
- [ ] #2 combat.whirlwind-clears shows at least one 'combat move ... use=whirl' leg, or the card names the gate that holds it with the log line
- [ ] #3 combat.corpse-explodes shows a 'combat corpse-explosion' line and necro.corpse-ce1/ce5 end with at least 2 minions, or the scenario expectation is corrected with evidence
- [ ] #4 combat.hold-interruptible and combat.teleport-reposition get an encounter that lowers the hero's life (their potion and escape triggers cannot fire at minlife 100)
- [x] #5 Tested for two classes
<!-- AC:END -->

## Notes (drv2, 2026-11-28)

Arbiter soak 14 against 9 (V both logs of the lab soak `out-arbiter255`, Sorceress 201, one seed): the five extra lines are 2 `field-entry-no-kit` and 3 `arbiter-starved`. The arbiter arm made 11 town trips against 8 and 380 kills against 215, so it entered the field more often with the same poor kit: 11 lines in 11 trips against 9 in 8, the rate did not rise. `arbiter-starved` is a soft row that exists only with the arbiter knobs on (the base arm cannot log it): `level/sweep` (the filler, `nothing else wanted`) outranked for 600 s by `recovery/farm_gold` (broke) and by `upkeep/upkeep`, and `level/level_up` by `quest/step`. Recovery, upkeep and a quest step outranking the levelling filler is the designed order, not a family bit or admission fault. Proposed (internal/bot, not this card): do not count the filler intent as starved, and judge the flip criterion `no new hit kinds` without the arbiter-only row.
