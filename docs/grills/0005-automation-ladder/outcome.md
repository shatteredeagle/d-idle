# Grill 0005 outcome: the automation ladder

Result: [ADR 0016](../../adr/0016-automation-ladder.md), status proposed, 2026-11-05. The user flips it to accepted. Notes of the grill session; the ADR is the record.

## Rounds and answers

The session ran one question at a time at the user's request. Numbers are the session's, not the prebrief's.

| Q | Topic | Answer |
|---|---|---|
| 1 | When ladder code starts | Start now, a complete refocus: the old gates were based on automated progression. Grill recommended waiting for Gate 2; overruled |
| 2 | What the old gates become | Yes: tier order ranks work, gates kept for T4, "done" moves to T5, defects fixed when they break the tier being built |
| 3 | Who answers in tests | Yes: files, never a live person. User added: results must not be skewed by our own answer choices. Led to answer profiles, holdouts, invariants for any answers |
| 4 | Mid-fight questions | Act now, ask after, first. Other options kept as a list with no trigger, reopened only if the user says it is not fun. Freeze-and-ask noted as an optional unlock. Later folded into Q13 |
| 5 | Altar and fidelity | Yes: brain in the bot, object on the fork, out of measured binaries. Comment: a bespoke fork is the likely final product, and end game content comes once the core is solid (parked) |
| 5b | Name | "The forge" was the working name of the altar. Term is altar. End game content gets a grill after the T5 gate passes |
| 6 | Rebirth | Rules and quality survive for now; later something closer to partial loss, a trauma |
| 6b | Later version | Can erase some learned rules too; tuning starts after the first rebirth playtest |
| 7 | "Every run" | User: must be able to resume a hero. New hero at clvl 1, resume any time, life ends by hardcore death or retiring |
| 8 | Draft section 7 items | Yes to streak reset, skip by sacrifice, new context per act, rune reservation. New-act re-asking to be judged in playtest |
| 9 | Worth rule | Yes: cube ratios, gold buys low levels only, vendor goods no profit. Tuned by playtest |
| 10 | Pace and first session | Yes. User idea: a feed of the hero's thoughts for questions, idle dialogue and story |
| 11 | Context keys | Yes, with the SME's written safety notes added (escape kit field). User asked for the SME; the grill cannot launch agents, so d2-sme review is a gate |
| 12 | Risk and N | Yes: fixed risk class, fixed choices, "not sure" answer, confirm on risky points |
| 13 | Unattended hero | User rejected the grill's waiting and timeout design: questions must not pause the flow, inattention should cost death or slow progress, thoughts should fade. Adopted |
| 14 | Player screen | Yes: one screen on d2dview, Thoughts feed replaces the question card, journal is its history, rules list, revoke one or all, no self-revoking rules |
| 15 | Cards | Yes: 3 new cards, goal G-G. User: the backlog should have no card limit (parked); the 4 T4 and Gate A cards wait in tier order |

Prebrief section 6 mapping: 1 to Q9; 2 and 11 to Q10; 3 to Q11; 4 to Q12; 5 to Q3; 6 to Q4 and Q13; 7 to Q2; 8 to Q6; 9 and 14 to Q1 and Q2; 10 to Q8 (details parked); 12 to Q14; 13 to Q15. Prebrief section 8 held no recommendation to compare.

## Corrections found during the grill

- Prebrief Q1 implies Hel is rarer than Ist; by standard rune order Ist (r24) is rarer than Hel (r15) (S, memory; r10 Thul is V in internal/cubejobs).
- The g2c record says a real claim needs 5 seeds; G-C says 3 (both V). Moot while Gate 2 waits for T4; settle it before Gate 2 runs again.
- docs/cards.md lists 13 active cards against the cap of 12 (V).
- The record proposal has `status: accepted`, says altar quality stays as if decided, and keeps the waiting hero. ADR 0016 replaces it where they differ.

## Parked

See the ADR: bespoke fork as final product; end game content; removing the card cap; several heroes at once; story writing; runeword details.

## For the coordinator (not done by this session)

1. Update "Current truth" in docs/decisions.md on acceptance.
2. Record the card cap removal if the user confirms it, and fix the "Gate A" entry and any other glossary text the ladder changes.
3. Ask the d2-sme: "For each T0 to T2 decision point in docs/research/decision-point-store-spec.md section 5, which game-state fields would a veteran actually weigh, which bands make two situations the same decision, and which risk class fits?"
4. The tooling-freeze note says it lasts until the g2c verdict and the user lifts it. The verdict is in (NOT PASSED). ADR 0016 treats ladder code as product work outside the freeze.

## Proposed cards (none created here)

| Card | Goal | Scope | Acceptance |
|---|---|---|---|
| Decision-point store | G-G | Update the spec for fading thoughts, then build the store, answer log, rules, revoke, offline. Check a rule file can express today's defaults | Streak, fade, "not sure", confirm and replay tests; two builds per the build-flexibility rule |
| Watch screen | G-G | Thoughts feed, orders box, uptime, rules list, journal on d2dview; answers by a separate bot route | Runs on the simulated runner (U), viewer stays read-only |
| T0 slice | G-G | Wire T0 decision points into combat with the untrained default | T0 drill with scripted answers across answer profiles, fresh clvl 1, writes after start 0 |

## Proposed glossary terms (GLOSSARY.md is not edited here)

**Automation tier**: one of six hero capabilities, T0 Order-taker to T5 Farmer, unlocked in order. Tiers are capabilities; acts are content. _Avoid_: "level".

**Decision point**: a named situation in the catalogue where the hero may ask, with fixed answers, a risk class, context fields and a default.

**Learn by doing**: teaching a rule by answering the same way in the same context N times in a row (2, 3 or 5 by risk). Deterministic count. _Avoid_: "training", "ML".

**Context key**: the decision point plus the banded values of its context fields; two situations with the same key count as the same decision.

**Thought**: one line in the Thoughts feed. A question thought has answer buttons and fades when not answered.

**Thoughts feed**: the single stream of questions, idle dialogue, story and incidents. Replaces the question card.

**Journal**: the permanent history of the Thoughts feed.

**Needs-input**: a question open in the Thoughts feed. The hero does not wait for it (ADR 0016 reversed the earlier "waits safe in town"). _Not to be confused with_: incident; the old log line `leveling: blocked: needs input`.

**Incident**: a failure the hero could not resolve, reported in plain language.

**Uptime**: the share of time spent making progress; deaths, recovery and town stalls lower it.

**Altar**: the in-game object that takes sacrifices, a custom d2d object on the private fork. _Avoid_: "forge" (the git host; also the Hellforge).

**Sacrifice**: spending an item at the altar to raise a tier's quality. _Not to be confused with_: selling to a vendor; the Paladin skill Sacrifice.

**Worth rule**: the one data table that values every sacrificed item, following the Horadric Cube ratios.

**Rebirth**: a hardcore death: the character resets, the profile keeps rules and quality (first pass).

**Order**: a task the player gives the hero. The only manual control. _Avoid_: "command" (the d2d socket has commands).

**New hero / resume**: a new hero starts at clvl 1; an existing hero can be resumed any time. _Avoid_: "run" here; it means a boss run or a measured run.

**Answer profile**: a scripted style of answering used in tests; known profiles and holdouts generated from a seed.

**Potential tweaks**: the list in ADR 0016 of set-aside ideas, opened only if the user says the game is not fun.
