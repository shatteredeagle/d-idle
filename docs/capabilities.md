# What the current build can and cannot do

> status: refreshed 2026-11-01 (card capdocs, docs only; first written by card readmedocs; nothing here was run against a game by the author). This page is a reading guide to evidence that lives elsewhere; when it disagrees with a generated or dated report, the report wins. Where things stand right now is always in [plan/STATUS.md](plan/STATUS.md) (generated block at the bottom). How to run it: [running.md](running.md). Changed in this refresh: Teleport hops are now wired from the bound skill and tuned (proven live, DRILLS24, 25, 27), the Frozen Armor recast retry is proven live (DRILLS29), resistance and melee handling are offline-verified, Paladin has an offline probe only, and the upstream PR counts moved (see the last section). No Gate 1 result is claimed: the soak is still running.

## Summary

d.idle is a Go bot that plays one Diablo II: Lord of Destruction 1.14d character, offline, on a game you own. Its reference
character is a Sorceress, and the only ground it has covered unattended is Normal Act 1 (character level 1 to about 13) on
the d2d backend, a reimplementation of the game engine, not on the original game under Wine. Two unattended two-hour soaks ran
on d2d; the second one, on the current line of fixes, did NOT pass Gate 1 (one seed, so it is not a statistical result either way).
Fixes merged after that soak have not been soaked again. A live drill suite checks the basic actions (potions, portals,
waypoints, shopping, repair, stash, save and rejoin) on throwaway characters; most pass, some fail or are blocked, and the
reports say which. Nothing beyond early Act 1 has been played, other classes only had short smoke runs, and the bot never
touches Battle.net.

## Capability table

Status words: **Proven live** (a dated live drill or soak shows it, on the backend named), **Works offline (tests)** (unit tests,
fakes or simulations only), **Experimental** (runs, but with open deviations or gaps), **Not yet verified**, **Known broken**.
"d2d" means the reimplementation, not the original game; see the gaps below for why that matters.

| Capability | Status | Evidence |
|---|---|---|
| Basic actions on d2d: belt fill and drink, ground pickup, town portal there and back, waypoints (all five acts' town rows), vendor buy and sell, repair, stash and gold, identify, merc potion, save and rejoin | Proven live | [probes/drills/README.md](probes/drills/README.md) (regression tables of DRILLS8 and DRILLS9), [primitives.md](primitives.md) (verdict column) |
| Recovering when d2d is killed mid-run (restart, saved XP not lower, no leftovers) | Proven live | [probes/drills/README.md](probes/drills/README.md) (D10 in DRILLS4, DRILLS5, DRILLS9), [probes/postfix-soak2.md](probes/postfix-soak2.md) (fault recovery) |
| Frozen Armor kept up in long fights (recast timed from the skill table, state read back) | Proven live | [probes/drills/README.md](probes/drills/README.md) (DRILLS13 and DRILLS16, section B), [fixbuff.md](fixbuff.md) |
| Frozen Armor recast retried when a hit cuts the cast | Proven live (one scenario) | DRILLS29 (D29-C, on both lab binaries): a hit cut the recast at 114.0 s, the bot retried 1.6 s later, armor absent 0 ms over 544 samples; [armorcut.md](armorcut.md). Unproven: a cut at normal monster damage (the drill used a boosted blow; plain zombies did not cut) and a cut within 6 s of the real expiry |
| Frozen Armor freeze rule and freeze-end resume (d2d side) | Experimental | In the private fork only, proposed upstream as PR #159 (open); [forkrebase.md](forkrebase.md), [research/sk-armor-trace.md](research/sk-armor-trace.md). Not in an unmodified d2d |
| Running unattended for two wall hours on d2d without a crash or a stuck process (Sorceress, Normal Act 1, one seed per soak) | Proven live | [probes/postfix-soak1.md](probes/postfix-soak1.md), [probes/postfix-soak2.md](probes/postfix-soak2.md) |
| Passing Gate 1 of the soak ladder | Known broken | [probes/postfix-soak2.md](probes/postfix-soak2.md): NOT PASSED (belt window and no-progress criteria fail in virtual minutes); [docs/contracts/soak-gates.md](contracts/soak-gates.md) |
| Matching the original game's XP per hour (the fidelity gate) | Not yet verified | [probes/postfix-soak2.md](probes/postfix-soak2.md) (INCONCLUSIVE at one seed, a gap of about 40 percent), [d2d-baselines-early.md](d2d-baselines-early.md), [docs/contracts/fidelity-gate.md](contracts/fidelity-gate.md) |
| Leaving the Den of Evil cave without bouncing back in | Proven live | every nav click passes one exit gate in caves (fixmouth4); DRILLS19: 0 bounces; known limit: a blocked away route can make the fighter retry (card fixmouthloop): [probes/drills/README.md](probes/drills/README.md) (DRILLS19 section), [fixmouth4.md](fixmouth4.md) |
| Fire Bolt and Charged Bolt as the main attacks to about level 25 | Experimental | [research/sorc-act-gate.md](research/sorc-act-gate.md) (YELLOW rows), [research/sk-missile-trace.md](research/sk-missile-trace.md) |
| Frost Nova | Experimental | On d2d the stock engine hits a monster many times per cast (RED in [research/sorc-act-gate.md](research/sorc-act-gate.md), [research/sk-nova-trace.md](research/sk-nova-trace.md)). The private maintainer fork carries a hit-once fix (0 of 100,198 differential cases differ): [forknext.md](forknext.md). That fix is merged upstream as PR #158 and is in the rebuilt fork stack ([forkrebase.md](forkrebase.md)). Not yet re-soaked |
| Chain Lightning and Lightning (Act 3 and later in the plan), on d2d | Known broken | d2d re-hits and casts differently from the real game, a d2d deviation and not a bot fault ([research/sorc-act-gate.md](research/sorc-act-gate.md), [research/sk-lightning-trace.md](research/sk-lightning-trace.md)); the hit-once fix does not yet cover these two ([research/pr-nexthit-proof.md](research/pr-nexthit-proof.md)) |
| Teleport hops in navigation (tuned: path-aware cell, break-even, stall escape, pockets) | Proven live | The hops are wired from the bound skill (production had none before). Stony Tomb 211 ticks against 254 walking, Halls of the Dead 1 533 against 578, stall escape and pocket exit proven: DRILLS24, 25, 27 in [probes/drills/README.md](probes/drills/README.md); [teletune.md](teletune.md), [teletune2.md](teletune2.md), [teletune3.md](teletune3.md). One border leg costs about 9 ticks over walking with 0 hops (measured as walking variance in DRILLS27). Fork binary, Sorceress, clvl 18 at 150 mana; no soak yet |
| Teleport against the real game's rules (range, landing, Duriel's Lair) | Experimental | [research/sk-teleport-trace.md](research/sk-teleport-trace.md): d2d places the player at any clicked distance and charges mana for a refused cast; several d2d drills fail against the real rules |
| Plain staff swing as the mana-out fallback at level 1 to 3 | Works offline (tests) | The melee model (to-hit, damage in 256ths, Blunt against undead) is in the resource simulator and fed from the live snapshot, unit-level only, not live: [skmeleebot.md](skmeleebot.md), [skmeleewire.md](skmeleewire.md); the game's rule was read and run in an emulator, and d2d deviates (floors damage, no Blunt term, off-weapon enhanced damage lost): [research/sk-melee-trace.md](research/sk-melee-trace.md), [research/sk-melee2-trace.md](research/sk-melee2-trace.md) |
| Monster resistance and immunity handling (floor -100, pierce only below 100) | Works offline (tests) | [skresistbot.md](skresistbot.md) against [research/sk-resist-trace.md](research/sk-resist-trace.md); d2d adds a full curse on an immune monster (PR #160, open). Not run live |
| Mercenary hire, revive, feed | Experimental | [primitives.md](primitives.md) (P18), [probes/drills/README.md](probes/drills/README.md) (DRILLS5, D8 rows) |
| Quests, Acts 2 to 5, Nightmare and Hell, Andariel | Not yet verified | Code paths may exist, but the bot has not played them: no soak reached them ([plan/STATUS.md](plan/STATUS.md) "Not started", [BACKLOG.md](BACKLOG.md)) |
| Classes other than the Sorceress | Experimental | [quickstart.md](quickstart.md) (section 4: short smoke runs only) |
| Paladin (Zealot-lite table, `--build paladin-zealot-lite`) | Works offline (tests) | Offline probe: 16 of 22 probes working, four findings fixed, a Zealot-lite table selectable; [classes/paladin-probe.md](classes/paladin-probe.md), [palfix.md](palfix.md). Nothing Paladin has run on d2d or in a game |
| Death handling and the death rule used to judge runs | Works offline (tests) | [docs/contracts/death-rule.md](contracts/death-rule.md), [soak-analysis.md](soak-analysis.md); real deaths in soaks: [probes/postfix-soak2-analysis.md](probes/postfix-soak2-analysis.md) |
| Town and economy policy tuning (trip rules, gold floor, post-death selling) | Experimental | [BACKLOG.md](BACKLOG.md) (REFOCUS lines: policy cards paused), [goldfloor.md](goldfloor.md) |
| Replaying a soak log into a verdict (`d2dsoak compare`) | Works offline (tests) | [d2d-soak.md](d2d-soak.md); the committed log is [probes/soak-logs/](probes/soak-logs/) |
| Dashboard and the sanitized viewer page (`d2bot`, `d2dash`, `d2dview`) | Works offline (tests) | [api.md](api.md), [d2d-view.md](d2d-view.md) |
| The original Game.exe path under Wine | Proven live | [g1-log.md](g1-log.md) (older soaks; frozen, no new features, [adr/0013-retire-game-exe-gates.md](adr/0013-retire-game-exe-gates.md)) |

## What is proven live

The drill suite (`cmd/d2drill`, code in `internal/drill`) runs one action at a time on a hidden d2d slot with a throwaway
character, twice, and compares the end states. The list of actions and what "correct" means for each is in
[primitives.md](primitives.md); the results, dated and in order, are in [probes/drills/README.md](probes/drills/README.md)
(DRILLS3 for the town, NPC and hover set, DRILLS4 and DRILLS9 for fault recovery, DRILLS5 and DRILLS8 for the regression
tables, DRILLS13 to DRILLS16 for skills and armor). Those reports are the source for pass and fail counts; this page does not
copy them because they change with every run. A drill proves the action works in the lab on d2d. It does not prove the bot picks
that action at the right moment over hours; that is what a soak is for.

## What has run unattended

- Soak 1 ([probes/postfix-soak1.md](probes/postfix-soak1.md)): two wall hours on d2d, Sorceress, Normal Act 1, one seed, no deaths.
- Soak 2 ([probes/postfix-soak2.md](probes/postfix-soak2.md)): the same recipe on a later build, one injected d2d kill that the
  bot recovered from in seconds, one death early in the run. Verdict: **Gate 1 NOT PASSED on this build (one seed)**. The failing
  criteria are the belt window and the ten-minute no-progress rule, read in virtual minutes (the user's decision of 2026-11-01);
  XP per hour is below the d2d baseline and INCONCLUSIVE at one seed. Corrections to the first write-up are in
  [probes/postfix-soak2-analysis.md](probes/postfix-soak2-analysis.md).
- Fixes merged after soak 2 are **not yet re-soaked**. Until a new Gate 1 run passes on several seeds, treat every improvement
  claimed in a fix document as measured by drills and offline tests only.
- The ladder (2 h, 8 h, 24 h, with the gates for each) is the soak-gates spec and the "d2d gate ladder" in
  [review-strategy.md](review-strategy.md). Only the first rung has been run on d2d.
- Older Game.exe soaks (two hours, eight hours, four hours) are history in [g1-log.md](g1-log.md); that backend is frozen.

## What the bot plays today

- Class: Sorceress. Game: Diablo II Lord of Destruction 1.14d, offline single player, softcore.
- Reached by evidence: Normal Act 1, character level 1 to about 13, in the soak logs. The campaign and farming goals exist for
  later content; nothing is claimed for it. Andariel, Act 2 and later are listed as not started in [plan/STATUS.md](plan/STATUS.md).
- Skills it uses: Fire Bolt, Charged Bolt, Frozen Armor, with a staff swing as a fallback (level 1 to 3). Per-skill verdicts
  by act band (green, yellow, red, unknown) are in [research/sorc-act-gate.md](research/sorc-act-gate.md), which is a draft: the
  rule it proposes (a skill must be green, or yellow with the effect accepted, before the bot is sent into an act) is not yet in a spec.
  In that draft the Act 1 skills are yellow (a known difference in d2d or in the bot with a bounded effect) except Frost Nova, which is red.
- Mana, cast timing, missile counts and damage rules were read from the original game's code or run in an emulator; see the
  `sk-*-trace.md` files in [research/](research/README.md). The drills compare d2d with those reads, not with a live original game.

## Known gaps and risks

- **d2d is not the original game.** It is an open-source reimplementation; deviations are recorded per topic in the trace
  documents ([research/d2d-fidelity-riskmap.md](research/d2d-fidelity-riskmap.md), the `sk-*-trace.md` set,
  [research/sorc-act-gate.md](research/sorc-act-gate.md)). The fidelity gap is measured, not closed ([d2d-baselines-early.md](d2d-baselines-early.md)).
- **Kill rates measured on d2d for Frost Nova and Chain Lightning are optimistic** until the hit-once fix is in the d2d build used (merged upstream as #158; whether the Chain Lightning hop path is fully covered is not re-checked here, U)
  (see the Frost Nova and Chain Lightning rows and the NEXTHIT item in [research/sorc-act-gate.md](research/sorc-act-gate.md)).
  The size of the inflation is modelled, not measured (that document tags it as inference).
- **The verified lab binary is a private fork** (virtual clock, a state dump, test verbs). The bot's soaks and drills ran on it.
  Whether an unmodified upstream d2d works with the bot is not verified. See [d2d-fork.md](d2d-fork.md) and [running.md](running.md).
- **Deaths happen.** The soak 2 death and the analysis of why are in
  [probes/postfix-soak2-analysis.md](probes/postfix-soak2-analysis.md); the rules that decide whether deaths fail a run are in
  [docs/contracts/death-rule.md](contracts/death-rule.md). Poverty after an early death (no potions, no
  town portal scrolls) is the main open loop.
- **The XP rate is well below the original-game baseline** on d2d, mostly bot-side (search and travel time, poverty
  recovery), per the analysis above.
- Known live failures at the time of writing include several d2d skill rules (Teleport targets, Energy Shield, skill delay), all in
  [probes/drills/README.md](probes/drills/README.md). The cave-mouth bounce listed here earlier is fixed (DRILLS19, row above).
- No container image is shipped; the d2d container is built locally only and a licence review comes before any shipping
  ([d2d-container/README.md](d2d-container/README.md)).

## What it deliberately does not do

- It never touches Battle.net, realms or chat, and nothing in this repo may add that (`AGENTS.md`). Offline single player only.
- No multiplayer, no hardcore (a switch exists, unverified), no Diablo II: Resurrected.
- It ships no game files; you need your own legitimate install ([../THIRD_PARTY.md](../THIRD_PARTY.md), [data.md](data.md)).
- It does not create or delete your characters; test cheats exist only for a small list of throwaway lab characters and are
  logged ([cheats.md](cheats.md)).

## Experimental or paused

Town and economy policy cards (trip rules, gold floor, post-death selling, area minimums) are paused until the high-criticality
primitives have a live pass or a fix card (REFOCUS lines in [BACKLOG.md](BACKLOG.md)). The idle-game layer (upgrades, prestige,
public page, stream) is parked ([../README.md](../README.md)). The d2d container is local only.

## Upstream work

Fixes to d2d are sent to its maintainer as small proven pull requests, one claim per commit, each with a differential against
the original game's code. Six early ones (#152, #153, #155, #156, #157, #158) are merged upstream; #159 (Frozen Armor freeze rule and freeze-end resume) and #160 (curse on an immune monster) are open ([research/openpr-log.md](research/openpr-log.md)). The private fork was rebuilt on the merged base ([forkrebase.md](forkrebase.md)). The queue and the rules are in [research/pr-bundle-queue.md](research/pr-bundle-queue.md); counts are
not copied here because they change daily. Our own patches to the d2d fork are never committed to this repository.
