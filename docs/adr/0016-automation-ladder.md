---
status: accepted
date: 2026-11-05
decision-makers: user (grill 0005)
---
# 0016: Automation is the progression: the tier ladder T0 to T5 replaces the gates as the ranking of work

## Context and Problem Statement

ADR 0015 ranks work by goals G-A to G-F, defines "done" as a fresh 24 h unattended Act 1 clear plus the A:B verdict, and parks the idle layer. The product discussion of 2026-11-05 chose a different centre: d.idle is an incremental Diablo II on d2d in which automation itself is the progression ([research/automation-ladder.md](../research/automation-ladder.md), DRAFT). The user ranked the shapes: automation as progression first, watch the hero second, job runner third; the abstract simulator is rejected, so the real engine keeps running the game.

The old gates judge automated progression: Gate 2 asks for clvl 16 and the Andariel criterion unattended (V docs/contracts/soak-gates.md), which is T3/T4 work. The ladder starts at T0, so those gates cannot be what the ladder waits for. State on 2026-11-05: Gate 2 g2c NOT PASSED, seeds 11 and 17 pass, seed 13 fails (V commit 7059d664); Andariel never cleared (V docs/capabilities.md via the draft). T0 to T1 code is mostly built, T2 to T4 partial, T5 not started; the ask mechanism, learned-rule store, altar and player screen do not exist (V draft sections 2 to 4).

Tags: V read in the repo, S secondary or memory, U unverified, D the user's decision. All numbers below are starting points for playtesting unless a contract owns them.

## Considered Options

**When ladder code starts**
* Now, as a complete refocus (chosen)
* When Gate 2 passes and a card row is free
* After all of G-A

**What the old gates become**
* Demoted: kept, run once T4 is reached, tier order ranks work (chosen)
* Parallel: gate chain keeps its priority beside the ladder
* Deleted

**How the hero asks**
* Fading thoughts: the hero never stops, carries on with its untrained default, the question fades (chosen)
* Hero waits safe in town with a timeout (the earlier decision, reversed here)
* Act on a default, then ask afterwards in town (first chosen, then folded into fading thoughts)

**What survives rebirth**
* Rules and all altar quality (chosen as the first pass)
* Rules only
* Rules and part of the quality (the intended later direction, extended to losing some rules)

## Decision Outcome

### Direction

1. d.idle is an incremental Diablo II on d2d. The player never controls the hero directly: no click-to-move, no casting. The player gives orders and answers questions.
2. Automation is the progression: T0 Order-taker, T1 Tidy, T2 Runner, T3 Planner, T4 Questor, T5 Farmer. Each tier needs the one below (a linear ladder). Tiers are capabilities, acts are content: the full ladder is built inside Act 1 Normal first, other acts follow as expansions.
3. Ladder code starts as soon as this record is accepted, beginning at T0. It does not wait for Gate 2. Ladder code is product work, not tooling, so the tooling freeze does not cover it.
4. Tier order replaces G-A to G-F as the ranking of work.
5. Gate 2 and Gate 3 are kept as the T4 acceptance, and Gate A stays separate. They run once T4 is reached; Gate 2 re-runs stop for now.
6. "Done" moves to the T5 product bar: a goal-driven farming hero. The T5 gate numbers are set when the T4 acceptance passes.
7. A bot defect is fixed now when it breaks play at the tier being built (the seed 13 cursor loop is the example). A fix that only serves Gate 2's level or Andariel floors waits.

### Heroes

8. A new hero always starts at clvl 1; there are no premade characters. An existing hero can be resumed at any time with level, items and quests intact. A hero's life ends only by hardcore death or when the player retires it. This replaces the draft's "every run starts fresh". A softcore hero keeps going after death under today's corpse rule. Measured runs still start fresh and a bot restart inside one still violates RESTARTS.
9. Hardcore stays unbuilt until the T4 acceptance passes. On rebirth, first pass: learned rules and altar quality both survive, stored per profile; level, items, quests and sacrificed goods are lost. Intended later direction: rebirth is a trauma that loses part of the hero's memory, which may include some learned rules as well as quality. Tuning starts after the first rebirth playtest, and that playtest must show the reborn early game is still dangerous.

### Learning

10. The hero never stops for a question. A question appears as a thought in the Thoughts feed and the hero carries on with its default. A thought fades after a time (start about 30 to 60 seconds, U). A faded thought neither counts toward a streak nor resets one; the next occurrence makes a new thought. This reverses "needs-input: the hero waits safe in town" of 2026-11-05.
11. The default is the untrained behaviour, plain and often poor. Not paying attention costs deaths and slow progress; that is the intended penalty. Irreversible choices (skill points, sacrifices, entering the next act) default to doing nothing. A poor default may cost time or lives but must never loop or stall.
12. The same answer in the same context N times in a row teaches a rule. N is 2, 3 or 5 for safe, normal, risky, as tunable data. A different answer resets the streak. A sacrifice can skip one repetition. A new act or difficulty is a new context.
13. Risk class is fixed per decision point in the catalogue. Answers are always fixed choices, never typed numbers. Every question offers "not sure, you decide", which uses the default and does not count. Risky points confirm before learning ("from now on I'll always do X here, OK?"); a no, or a confirm that fades, leaves the rule unlearned, and a no resets the streak.
14. Context: each decision point lists its fields in a reviewable data table; continuous values become a few named bands that do not overlap; start coarse and split a band only when play shows a wrong rule; the context is shown in plain language. Worked example, T0 retreat: act, difficulty, life band, pack danger, escape kit (has potions and portal, one of them, neither). T1 loot: act, difficulty, item kind with rune rank class. Area is left out of the retreat key. Class and build stay out unless a point says otherwise.

### Altar and economy

15. Anything can be sacrificed, under one worth rule with no second currency. Worth follows the Horadric Cube ratios (V internal/cubejobs), so sacrificing a rune never beats cubing it. Each quality level names a minimum item class, so gold and junk pay only low levels. Nothing a vendor sells is worth more at the altar than the gold it cost. The farm goal reserves the runes its runeword needs. All values are one data table.
16. Pace: each quality level is priced in the farming time of the tier below, and each tier takes longer than the last. First session shape: first rule learned within about 10 minutes (U); T0 unlocked and a first sacrifice made, with T1 in sight. Questions per 10 minutes and time to first unlock are logged, never pass or fail. The user judges the fun.
17. Altar is the term (it was also called the forge). The worth table and quality levels live in the bot, in this repo. The object lives only on a private fork branch and only shows the altar and removes the item. Gate and A:B binaries never contain it. It is built after the store and the Thoughts feed. Shipping it waits for the GPL reviewer of ADR 0003.

### Testing

18. No live person in a measured test; a run that took a live answer is not evidence. Gates and soaks use a committed rule file with all tiers unlocked. Tier drills use committed answer scripts, because they test the learning itself. The run record carries the file hash next to `writes after start: N`. An answer goes to the bot's own store, not game memory, so it is not a cheat; the crutch rule is unchanged.
19. Results must not depend on our own choice of answers. Tests use several answer profiles (cautious, bold, inconsistent), starting at 3 known and 2 holdout, mirroring the seed rule; holdouts are generated from a seed at test time. A fix must also pass the holdouts. Invariants hold for any answers: a bad answer may slow or kill the hero, never loop, crash or stall it. "Today's defaults" is one profile, useful against old records, and proves nothing alone. The full matrix runs at tier boundaries, not after every fix.

### Player screen

20. One screen from T0, built on `cmd/d2dview` (V docs/d2d-view.md): game frame, Thoughts feed, orders box, uptime. Answers go through a separate bot route so the viewer stays read-only.
21. The Thoughts feed replaces the question card: one stream of questions, idle dialogue, story and incidents. The journal is its permanent history, including missed questions.
22. Uptime is the share of time spent making progress; deaths, recovery and town stalls lower it.
23. A rules list shows every learned rule with its context, with a "why did you do that?" link from journal lines. Revoke comes as one rule or reset all. A rule never revokes itself: after a death the journal names the rule and offers revoke, and the player decides.

### Cards

24. Three new cards under a new goal G-G "Automation ladder": the decision-point store, the watch screen, the T0 slice. The WIP limit of 3 implementation agents stays. m3-act1run, m3-andariel, m3-merc and gameexe-retire-soak stay in the backlog as T4 and Gate A work and wait their turn in tier order; the coordinator parks them only if the card cap is still in force when the new cards are created.

### Gates and triggers

| Item | Trigger or gate |
|---|---|
| Ladder code starts | the user flips this record to accepted |
| T5 gate numbers | the T4 acceptance passes |
| Hardcore is built | the T4 acceptance passes |
| Partial-loss rebirth tuning | the first rebirth playtest |
| Decision-point catalogue final | the d2-sme reviews context fields, bands and risk classes (question in the grill outcome) |
| Shipping the altar object | the user names a GPL reviewer (ADR 0003) |
| End game content grill | the T5 gate passes |
| Potential tweaks list reopened | the user says the game is not fun |
| A rule file can express today's defaults | unchecked (U); condition on the store card |
| The hero can work while a thought is open | follows from item 10; to be proven by the T0 slice (U) |

### Potential tweaks (no trigger; opened only if the user says it is not fun)

* Freeze the game and ask, as an optional "focus" unlock. Needs lockstep, a private-fork feature, so it has the altar's limits. Placement and cost open.
* Ask before leaving town, as standing orders.
* Longer fade time as an altar upgrade.
* A new act or difficulty re-asking everything may be annoying or fun: check in playtest.
* A scripted first town trip.

### Parked

* The final product as a bespoke d2d fork carrying lockstep, cheats and game features (user expectation). Must be squared with ADR 0003 and the ADR 0015 rule that deviations are fixed upstream.
* End game content, once the core is solid; the altar is what enables it. Own grill, trigger above.
* Removing the card cap: the user said "the backlog shouldn't have a card limit, the WIP limit stays". Amends ADR 0015 item 11 and needs its own record; the cap stands until then.
* Several heroes at once.
* How story and dialogue are written. Text must be our own; check docs/contracts/licence-policy.md.
* Runeword details: what the hero does on finding the exact goal rune, and who picks the runeword.

## Consequences

* Amends ADR 0015: item 1 and its 2026-11-04 amendment ("done means" moves to the T5 bar; the 24 h Act 1 clear plus A:B stays as the T4 and Gate A acceptance), item 7 (tier order replaces the goals as the ranking), item 10 (the idle layer is un-parked as this ladder; container, stream-grid, d2dmesh and the rest stay as they are). Amends roadmap M6 and PLAN.md's parked meta-game line. The older files stay as written; this record is the pointer.
* Rule (a) of the ADR 0015 amendment still holds for the engine: game.exe is the spec for game rules. The altar is an added feature, kept out of measured binaries, not a deviation to design around.
* On acceptance the "Current truth" block of docs/decisions.md needs updating (done means, ranking, parked list) by the coordinator or the user; this grill adds only the index line.
* docs/research/decision-point-store-spec.md still describes a waiting state, a question card and timeout defaults and must be updated first, inside the store card. The proposal in automation-ladder-record-proposal.md is superseded by this record where they differ.
* Work on T4 and Gate A stops being next; the cards stay. Upstream fixes, basics drills and defect cards continue where they serve the tier being built.
* More test runs: seeds and answer profiles multiply. The store card sizes this; the full matrix runs only at tier boundaries.
* Rebirth adds a test state (rules present, character fresh).

## Sources

docs/grills/0005-automation-ladder/prebrief.md and outcome.md; docs/research/automation-ladder.md; docs/research/automation-ladder-record-proposal.md; docs/research/decision-point-store-spec.md; ADR 0003, 0009, 0015; docs/contracts/soak-gates.md; docs/d2d-view.md; docs/d2d-fork.md; docs/sme/play-safety.md; internal/cubejobs/cubejobs.go; docs/cards.md; docs/research/project-review-20261103.md; PLAN.md "Decisions".
