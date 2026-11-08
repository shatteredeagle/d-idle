---
status: accepted
date: 2026-11-07
decision-makers: user (popup answers, 2026-11-07)
---
# 0019: Pacts are opt-in challenge rules with meta rewards; information and watch features unlock progressively

Builds on [ADR 0018](0018-fun-loop-expedition-ledger-2026-11-07.md) (it does not change its text). Designs: [research/pact-system.md](../research/pact-system.md) (F12), [research/info-unlocks.md](../research/info-unlocks.md) (F13), [research/unlockables.md](../research/unlockables.md) (F14).

## Decision

1. **Pacts (F12).** A Pact is an opt-in challenge or difficulty rule paid in the meta layer: tally multiplier, legacy charm points, Hall titles, cosmetics. A Pact never grants vanilla XP or loot, except through D2's own difficulty settings. Heat model: heat cap 12, rising to 20 with mastery; ranks 1 to 4 (5 for some); reward is linear up to the cap, then cosmetic only.
2. **Unlock order.** The first Pacts (bot-side challenges) open at the first Andariel kill. The PACT OF NIGHTMARES is ONE Pact: the same hero moves into Nightmare with a bonus; it opens when Baal is killed in Normal (the user first said Mephisto, then chose Baal after being told Baal is the 1.14d LoD final boss). A Pact of Hell as a sibling after Baal in Nightmare is a proposal, not decided.
3. **Hardcore** is a mode flag with its own hard confirmation, not a heat-costed Pact. Whether `/players` Pacts exist is decided after the live check (TASK-170 pactnative).
4. **First two Pacts to prototype:** Blood Raven's Tithe (no belt potions) and Andariel's Shadow (death limit), as policy-sheet rows (TASK-167 pactbotside).
5. **Away safety.** Typed, dated consent per Pact, a low-life floor, a town pause, and a separate Hardcore confirmation. Pacts are OFF in gates and measured runs.
6. **Information unlocks (F13).** Progressive view-only senses in a SIDE-PANEL minimap: this is the primary design (user: "side panel minimap is good v1, may be better overall"); the video overlay is deferred (TASK-150). Tiers 1 to 4 are earned by Codex firsts or level, tiers 5 and up are bought with tally. Senses never affect bot behaviour and are token-gated.
7. **Unlockables (F14).** Watch-UI features unlock through a data-driven ladder (TASK-153 unlockladder), the ten features of the research note. Basics are never gated and an unlock is never silent (a reveal and a log line).

## Consequences

- Pact work is docs and cards first (TASK-166 to 173); nothing here touches gates, the build tables or vanilla rules.
- TASK-150 is Deferred and parked; TASK-153 depends on TASK-149.
